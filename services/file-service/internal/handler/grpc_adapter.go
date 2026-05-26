package handler

import (
	"context"
	"fmt"
	"io"

	commonv1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/common/v1"
	filev1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/file/v1"
	"github.com/nomados/nomados/services/file-service/internal/repository"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// FileServiceGRPCAdapter wraps FileServiceHandler and implements the
// filev1.FileServiceServer gRPC interface by translating between
// proto request/response types and the existing handler's struct types.
type FileServiceGRPCAdapter struct {
	filev1.UnimplementedFileServiceServer
	handler *FileServiceHandler
}

// NewFileServiceGRPCAdapter creates a new FileServiceGRPCAdapter.
func NewFileServiceGRPCAdapter(handler *FileServiceHandler) *FileServiceGRPCAdapter {
	return &FileServiceGRPCAdapter{
		handler: handler,
	}
}

// Upload handles client-streaming file upload. Phase 1: returns Unimplemented.
func (a *FileServiceGRPCAdapter) Upload(stream grpc.ClientStreamingServer[filev1.UploadRequest, filev1.UploadResponse]) error {
	return status.Error(codes.Unimplemented, "streaming upload not yet implemented")
}

// Download handles server-streaming file download. Phase 1: returns Unimplemented.
func (a *FileServiceGRPCAdapter) Download(req *filev1.DownloadRequest, stream grpc.ServerStreamingServer[filev1.DownloadResponse]) error {
	return status.Error(codes.Unimplemented, "streaming download not yet implemented")
}

// List handles the unary ListFiles RPC by translating proto types to handler
// struct types and back.
func (a *FileServiceGRPCAdapter) List(ctx context.Context, req *filev1.ListFilesRequest) (*filev1.ListFilesResponse, error) {
	workspaceID := req.GetWorkspaceId().GetValue()
	if workspaceID == "" {
		return nil, status.Error(codes.InvalidArgument, "workspace_id is required")
	}

	result, err := a.handler.ListFiles(ctx, &ListRequest{
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list files: %v", err)
	}

	protoFiles := make([]*filev1.FileInfo, len(result.Files))
	for i, f := range result.Files {
		protoFiles[i] = fileMetadataToFileInfo(f)
	}

	return &filev1.ListFilesResponse{
		Files: protoFiles,
	}, nil
}

// Delete handles the unary DeleteFile RPC by translating proto types to handler
// struct types and back.
func (a *FileServiceGRPCAdapter) Delete(ctx context.Context, req *filev1.DeleteFileRequest) (*commonv1.Empty, error) {
	fileID := req.GetFileId().GetValue()
	workspaceID := req.GetWorkspaceId().GetValue()

	if fileID == "" {
		return nil, status.Error(codes.InvalidArgument, "file_id is required")
	}
	if workspaceID == "" {
		return nil, status.Error(codes.InvalidArgument, "workspace_id is required")
	}

	_, err := a.handler.DeleteFile(ctx, &DeleteRequest{
		WorkspaceID: workspaceID,
		FileID:      fileID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete file: %v", err)
	}

	return &commonv1.Empty{}, nil
}

// fileMetadataToFileInfo converts an internal FileMetadata struct to a proto FileInfo.
func fileMetadataToFileInfo(f *repository.FileMetadata) *filev1.FileInfo {
	return &filev1.FileInfo{
		Id:        &commonv1.UUID{Value: f.ID.String()},
		Filename:  f.Filename,
		Size:       f.Size,
		CreatedAt:  f.CreatedAt.Unix(),
		UpdatedAt:  f.UpdatedAt.Unix(),
	}
}

// Ensure the adapter implements the FileServiceServer interface.
var _ filev1.FileServiceServer = (*FileServiceGRPCAdapter)(nil)

// Suppress unused import warnings for types used in streaming method signatures.
var (
	_ = fmt.Sprintf
	_ = io.EOF
)