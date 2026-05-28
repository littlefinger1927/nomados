package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
	commonv1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/common/v1"
	filev1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/file/v1"
)

// HTTPHandler provides REST endpoints for file upload/download.
// This is needed because grpc-gateway doesn't support streaming RPCs,
// so browsers need a regular HTTP endpoint for file transfers.
type HTTPHandler struct {
	handler *FileServiceHandler
}

// NewHTTPHandler creates a new HTTPHandler.
func NewHTTPHandler(handler *FileServiceHandler) *HTTPHandler {
	return &HTTPHandler{handler: handler}
}

// RegisterRoutes registers HTTP routes on the given mux.
func (h *HTTPHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/v1/file/upload", h.handleUpload)
	mux.HandleFunc("/v1/file/download/", h.handleDownload)
	mux.HandleFunc("/v1/file/list", h.handleList)
	mux.HandleFunc("/v1/file/delete", h.handleDelete)
}

func (h *HTTPHandler) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.Header.Get("X-User-Id")
	if userID == "" {
		http.Error(w, "unauthorized: missing user identity", http.StatusUnauthorized)
		return
	}

	// Parse multipart form (max 100MB)
	if err := r.ParseMultipartForm(100 << 20); err != nil {
		http.Error(w, fmt.Sprintf("failed to parse form: %v", err), http.StatusBadRequest)
		return
	}

	workspaceID := r.FormValue("workspace_id")
	if workspaceID == "" {
		http.Error(w, "workspace_id is required", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get file: %v", err), http.StatusBadRequest)
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	resp, err := h.handler.UploadFile(r.Context(), &UploadRequest{
		WorkspaceID: workspaceID,
		UserID:      userID,
		Filename:    header.Filename,
		ContentType: contentType,
		Data:         file,
		Size:         header.Size,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to upload file: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"file_id": resp.FileID,
	})
}

func (h *HTTPHandler) handleDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.Header.Get("X-User-Id")
	if userID == "" {
		http.Error(w, "unauthorized: missing user identity", http.StatusUnauthorized)
		return
	}

	// Extract workspace_id and file_id from query params
	workspaceID := r.URL.Query().Get("workspace_id")
	fileID := r.URL.Query().Get("file_id")
	if workspaceID == "" || fileID == "" {
		http.Error(w, "workspace_id and file_id are required", http.StatusBadRequest)
		return
	}

	resp, err := h.handler.DownloadFile(r.Context(), &DownloadRequest{
		WorkspaceID: workspaceID,
		FileID:      fileID,
		UserID:      userID,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to download file: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp.Data.Close()

	w.Header().Set("Content-Type", resp.Metadata.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, resp.Metadata.Filename))
	if resp.Metadata.Size > 0 {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", resp.Metadata.Size))
	}

	io.Copy(w, resp.Data)
}

func (h *HTTPHandler) handleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.Header.Get("X-User-Id")
	if userID == "" {
		http.Error(w, "unauthorized: missing user identity", http.StatusUnauthorized)
		return
	}

	var req struct {
		WorkspaceID string `json:"workspace_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("failed to decode request: %v", err), http.StatusBadRequest)
		return
	}

	result, err := h.handler.ListFiles(r.Context(), &ListRequest{
		WorkspaceID: req.WorkspaceID,
		UserID:      userID,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to list files: %v", err), http.StatusInternalServerError)
		return
	}

	// Convert to proto-style response for consistency
	protoFiles := make([]*filev1.FileInfo, len(result.Files))
	for i, f := range result.Files {
		protoFiles[i] = &filev1.FileInfo{
			Id:        &commonv1.UUID{Value: f.ID.String()},
			Filename:  f.Filename,
			Size:       f.Size,
			CreatedAt:  f.CreatedAt.Unix(),
			UpdatedAt:  f.UpdatedAt.Unix(),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"files": protoFiles,
	})
}

func (h *HTTPHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.Header.Get("X-User-Id")
	if userID == "" {
		http.Error(w, "unauthorized: missing user identity", http.StatusUnauthorized)
		return
	}

	var req struct {
		WorkspaceID string `json:"workspace_id"`
		FileID      string `json:"file_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("failed to decode request: %v", err), http.StatusBadRequest)
		return
	}

	_, err := h.handler.DeleteFile(r.Context(), &DeleteRequest{
		WorkspaceID: req.WorkspaceID,
		FileID:      req.FileID,
		UserID:      userID,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to delete file: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{})
}

// ensure uuid package is referenced
var _ = uuid.Nil