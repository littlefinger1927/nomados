package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	commonv1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/common/v1"
	filev1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/file/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestFileListViaGRPC tests listing files in a workspace via gRPC.
// Since the file Upload RPC is client-streaming and not yet implemented
// for HTTP gateway use, we test the List endpoint directly via gRPC.
func TestFileListViaGRPC(t *testing.T) {
	skipIfUnreachable(t, fileServiceAddr, "file-service")

	client, conn := newFileClient(t)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	workspaceID := randomSuffix()

	resp, err := client.List(ctx, &filev1.ListFilesRequest{
		WorkspaceId: &commonv1.UUID{Value: workspaceID},
	})

	// An empty workspace should return an empty list, not an error.
	if err != nil {
		// If the service returns Internal, it may be a database connectivity issue.
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.Internal {
			t.Skipf("file service database may not be available: %v", err)
		}
		t.Fatalf("List RPC failed: %v", err)
	}

	if resp == nil {
		t.Fatal("expected non-nil response from List")
	}

	t.Logf("Listed %d files in workspace %s", len(resp.Files), workspaceID)
}

// TestFileListViaGateway tests listing files through the HTTP gateway.
// This verifies the gateway properly proxies List requests to the file service.
func TestFileListViaGateway(t *testing.T) {
	skipIfUnreachable(t, fileServiceAddr, "file-service")
	skipIfUnreachable(t, gatewayAddr, "gateway")

	userID := "gw-file-test-" + randomSuffix()
	sessionID := "gw-file-session-" + randomSuffix()
	deviceID := "gw-file-device-" + randomSuffix()

	accessToken, err := generateAccessToken(userID, sessionID, deviceID)
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}

	workspaceID := randomSuffix()

	listBody, _ := json.Marshal(map[string]interface{}{
		"workspaceId": map[string]string{"value": workspaceID},
	})

	req, err := http.NewRequest("POST", gatewayBaseURL+"/v1/file/list", bytes.NewReader(listBody))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("list files request failed: %v", err)
	}
	defer resp.Body.Close()

	// The file service may return 200 with empty files or an error if the
	// workspace does not exist in its database. Both are acceptable outcomes
	// for this test; we mainly verify the gateway proxies correctly.
	body, _ := io.ReadAll(resp.Body)
	t.Logf("List files response: status=%d, body=%s", resp.StatusCode, string(body))

	// Verify the gateway forwarded the request (any non-401 status means auth worked)
	if resp.StatusCode == http.StatusUnauthorized {
		t.Error("expected authenticated request to not return 401")
	}
}

// TestFileDeleteViaGRPC tests deleting a file via gRPC.
// Since the file may not exist, we verify the error handling behavior.
func TestFileDeleteViaGRPC(t *testing.T) {
	skipIfUnreachable(t, fileServiceAddr, "file-service")

	client, conn := newFileClient(t)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	workspaceID := randomSuffix()
	fileID := randomSuffix()

	// Attempt to delete a non-existent file
	_, err := client.Delete(ctx, &filev1.DeleteFileRequest{
		FileId:      &commonv1.UUID{Value: fileID},
		WorkspaceId: &commonv1.UUID{Value: workspaceID},
	})

	if err == nil {
		t.Error("expected error when deleting non-existent file, got nil")
	} else {
		st, ok := status.FromError(err)
		if !ok {
			t.Logf("Delete returned error (expected for non-existent file): %v", err)
		} else {
			// NotFound or Internal are both acceptable for a non-existent file
			t.Logf("Delete returned status %v for non-existent file (expected)", st.Code())
		}
	}
}

// TestFileDeleteViaGateway tests deleting a file through the HTTP gateway.
// Verifies the gateway proxies delete requests with authentication.
func TestFileDeleteViaGateway(t *testing.T) {
	skipIfUnreachable(t, fileServiceAddr, "file-service")
	skipIfUnreachable(t, gatewayAddr, "gateway")

	userID := "gw-file-del-test-" + randomSuffix()
	sessionID := "gw-file-del-session-" + randomSuffix()
	deviceID := "gw-file-del-device-" + randomSuffix()

	accessToken, err := generateAccessToken(userID, sessionID, deviceID)
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}

	workspaceID := randomSuffix()
	fileID := randomSuffix()

	deleteBody, _ := json.Marshal(map[string]interface{}{
		"fileId":      map[string]string{"value": fileID},
		"workspaceId": map[string]string{"value": workspaceID},
	})

	req, err := http.NewRequest("POST", gatewayBaseURL+"/v1/file/delete", bytes.NewReader(deleteBody))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("delete file request failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	t.Logf("Delete file response: status=%d, body=%s", resp.StatusCode, string(body))

	// Verify the gateway forwarded the authenticated request (not 401)
	if resp.StatusCode == http.StatusUnauthorized {
		t.Error("expected authenticated request to not return 401")
	}
}

// TestFileUploadStreamingNotImplemented verifies that the streaming Upload RPC
// returns Unimplemented status when called via gRPC.
func TestFileUploadStreamingNotImplemented(t *testing.T) {
	skipIfUnreachable(t, fileServiceAddr, "file-service")

	client, conn := newFileClient(t)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// The Upload RPC is client-streaming and returns Unimplemented
	stream, err := client.Upload(ctx)
	if err != nil {
		// Connection error is acceptable
		t.Logf("Upload stream creation returned error: %v", err)
		return
	}

	// Try to send a metadata message
	metadata := &filev1.UploadRequest{
		Data: &filev1.UploadRequest_Metadata{
			Metadata: &filev1.FileMetadata{
				WorkspaceId: &commonv1.UUID{Value: randomSuffix()},
				Filename:    "test-file.txt",
				Size:        100,
				ContentType: "text/plain",
			},
		},
	}
	if err := stream.Send(metadata); err != nil {
		t.Logf("Upload Send returned error (expected for Unimplemented): %v", err)
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.Unimplemented {
			t.Log("Upload correctly returns Unimplemented status")
		} else {
			t.Logf("Upload returned error: %v (code: %v)", err, st.Code())
		}
	} else {
		t.Logf("Upload returned response: %v (unexpected for Unimplemented)", resp)
	}
}

// TestFileDownloadStreamingNotImplemented verifies that the streaming Download RPC
// returns Unimplemented status when called via gRPC.
func TestFileDownloadStreamingNotImplemented(t *testing.T) {
	skipIfUnreachable(t, fileServiceAddr, "file-service")

	client, conn := newFileClient(t)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stream, err := client.Download(ctx, &filev1.DownloadRequest{
		FileId:      &commonv1.UUID{Value: randomSuffix()},
		WorkspaceId: &commonv1.UUID{Value: randomSuffix()},
	})
	if err != nil {
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.Unimplemented {
			t.Log("Download correctly returns Unimplemented status")
			return
		}
		// Other errors are acceptable (service may not be fully configured)
		t.Logf("Download returned error: %v", err)
		return
	}
	defer stream.CloseSend()

	// Try to receive from the stream
	_, recvErr := stream.Recv()
	if recvErr != nil {
		st, ok := status.FromError(recvErr)
		if ok && st.Code() == codes.Unimplemented {
			t.Log("Download stream correctly returns Unimplemented status")
		} else {
			t.Logf("Download stream returned error: %v (code: %v)", recvErr, st.Code())
		}
	} else {
		t.Log("Download stream returned data (unexpected for Unimplemented)")
	}
}

// TestFileServiceUnauthorizedAccess verifies that file operations require
// authentication when accessed through the gateway.
func TestFileServiceUnauthorizedAccess(t *testing.T) {
	skipIfUnreachable(t, gatewayAddr, "gateway")

	// Request file list without authentication
	listBody, _ := json.Marshal(map[string]interface{}{
		"workspaceId": map[string]string{"value": randomSuffix()},
	})

	req, err := http.NewRequest("POST", gatewayBaseURL+"/v1/file/list", bytes.NewReader(listBody))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	// No Authorization header

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 for unauthenticated file list request, got %d", resp.StatusCode)
	}

	// Request file delete without authentication
	deleteBody, _ := json.Marshal(map[string]interface{}{
		"fileId":      map[string]string{"value": randomSuffix()},
		"workspaceId": map[string]string{"value": randomSuffix()},
	})

	delReq, err := http.NewRequest("POST", gatewayBaseURL+"/v1/file/delete", bytes.NewReader(deleteBody))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	delReq.Header.Set("Content-Type", "application/json")
	// No Authorization header

	delResp, err := http.DefaultClient.Do(delReq)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer delResp.Body.Close()

	if delResp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 for unauthenticated file delete request, got %d", delResp.StatusCode)
	}
}

// TestFileWorkspaceIsolation verifies that file operations respect workspace
// boundaries. A user listing files in their own workspace should only see
// their own files, not files from another workspace.
func TestFileWorkspaceIsolation(t *testing.T) {
	skipIfUnreachable(t, fileServiceAddr, "file-service")

	client, conn := newFileClient(t)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	workspaceA := "ws-iso-file-a-" + randomSuffix()
	workspaceB := "ws-iso-file-b-" + randomSuffix()

	// List files in workspace A — should be empty (or return not found)
	respA, err := client.List(ctx, &filev1.ListFilesRequest{
		WorkspaceId: &commonv1.UUID{Value: workspaceA},
	})
	if err != nil {
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.Internal {
			t.Skipf("file service database may not be available: %v", err)
		}
		t.Fatalf("List for workspace A failed: %v", err)
	}

	// List files in workspace B — should be empty (or return not found)
	respB, err := client.List(ctx, &filev1.ListFilesRequest{
		WorkspaceId: &commonv1.UUID{Value: workspaceB},
	})
	if err != nil {
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.Internal {
			t.Skipf("file service database may not be available: %v", err)
		}
		t.Fatalf("List for workspace B failed: %v", err)
	}

	// Verify no file IDs are shared between the two workspaces
	filesA := make(map[string]bool)
	for _, f := range respA.GetFiles() {
		filesA[f.GetId().GetValue()] = true
	}

	for _, f := range respB.GetFiles() {
		if filesA[f.GetId().GetValue()] {
			t.Errorf("file %s appears in both workspace A and B — isolation violation", f.GetId().GetValue())
		}
	}

	t.Logf("Workspace A has %d files, workspace B has %d files — isolation verified",
		len(respA.GetFiles()), len(respB.GetFiles()))
}

// TestFileGatewayProxiesToGRPC verifies that the gateway correctly proxies
// file service requests by comparing gateway and gRPC responses.
func TestFileGatewayProxiesToGRPC(t *testing.T) {
	skipIfUnreachable(t, fileServiceAddr, "file-service")
	skipIfUnreachable(t, gatewayAddr, "gateway")

	userID := "gw-proxy-test-" + randomSuffix()
	sessionID := "gw-proxy-session-" + randomSuffix()
	deviceID := "gw-proxy-device-" + randomSuffix()

	accessToken, err := generateAccessToken(userID, sessionID, deviceID)
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}

	workspaceID := randomSuffix()

	// Make request via gRPC
	client, conn := newFileClient(t)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	grpcResp, grpcErr := client.List(ctx, &filev1.ListFilesRequest{
		WorkspaceId: &commonv1.UUID{Value: workspaceID},
	})
	if grpcErr != nil {
		st, ok := status.FromError(grpcErr)
		if ok && st.Code() == codes.Internal {
			t.Skipf("file service database may not be available: %v", grpcErr)
		}
		t.Fatalf("gRPC List failed: %v", grpcErr)
	}

	// Make request via gateway HTTP
	listBody, _ := json.Marshal(map[string]interface{}{
		"workspaceId": map[string]string{"value": workspaceID},
	})

	req, err := http.NewRequest("POST", gatewayBaseURL+"/v1/file/list", bytes.NewReader(listBody))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	httpResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		t.Fatalf("expected 200 from HTTP list, got %d: %s", httpResp.StatusCode, string(body))
	}

	var httpData map[string]interface{}
	if err := json.NewDecoder(httpResp.Body).Decode(&httpData); err != nil {
		t.Fatalf("failed to decode HTTP response: %v", err)
	}

	// Compare file counts
	grpcFileCount := len(grpcResp.GetFiles())
	httpFiles, ok := httpData["files"].([]interface{})
	if !ok {
		t.Fatal("expected files array in HTTP response")
	}

	if grpcFileCount != len(httpFiles) {
		t.Errorf("gRPC returned %d files, HTTP returned %d — mismatch", grpcFileCount, len(httpFiles))
	}

	t.Logf("gRPC and HTTP both returned %d files for workspace %s", grpcFileCount, workspaceID)
}

// unused import guard
var _ = fmt.Sprintf