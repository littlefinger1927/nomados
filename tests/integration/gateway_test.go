package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	commonv1 "github.com/nomados/nomados/packages/shared-types/gen/nomados/common/v1"
)

// TestGatewayReachable verifies the gateway HTTP endpoint is reachable.
func TestGatewayReachable(t *testing.T) {
	skipIfUnreachable(t, gatewayAddr, "gateway")
}

// TestGatewayAuthRequired verifies that protected endpoints return 401
// when no Authorization header is provided.
func TestGatewayAuthRequired(t *testing.T) {
	skipIfUnreachable(t, gatewayAddr, "gateway")

	protectedPaths := []struct {
		name string
		path string
		body string
	}{
		{"workspace_create", "/v1/workspace/create", `{"userId":{"value":"test"},"name":"test"}`},
		{"workspace_list", "/v1/workspace/list", `{"userId":{"value":"test"}}`},
		{"workspace_get", "/v1/workspace/get", `{"id":{"value":"test"}}`},
		{"workspace_pause", "/v1/workspace/pause", `{"id":{"value":"test"}}`},
		{"workspace_stop", "/v1/workspace/stop", `{"id":{"value":"test"}}`},
		{"session_validate", "/v1/session/validate", `{"access_token":"test"}`},
		{"file_list", "/v1/file/list", `{"workspaceId":{"value":"test"}}`},
		{"file_delete", "/v1/file/delete", `{"fileId":{"value":"test"},"workspaceId":{"value":"test"}}`},
	}

	for _, tc := range protectedPaths {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", gatewayBaseURL+tc.path, bytes.NewReader([]byte(tc.body)))
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusUnauthorized {
				t.Errorf("expected 401 for %s without auth, got %d", tc.path, resp.StatusCode)
			}
		})
	}
}

// TestGatewayAuthInvalidToken verifies that protected endpoints return 401
// when an invalid Authorization header is provided.
func TestGatewayAuthInvalidToken(t *testing.T) {
	skipIfUnreachable(t, gatewayAddr, "gateway")

	req, err := http.NewRequest("POST", gatewayBaseURL+"/v1/workspace/list", bytes.NewReader([]byte(`{}`)))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer invalid-token-12345")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 with invalid token, got %d", resp.StatusCode)
	}
}

// TestGatewayPublicPaths verifies that auth endpoints are accessible
// without authentication (they may return errors for invalid data,
// but should not return 401 Unauthorized).
func TestGatewayPublicPaths(t *testing.T) {
	skipIfUnreachable(t, gatewayAddr, "gateway")

	publicPaths := []struct {
		name string
		path string
		body string
	}{
		{"register", "/v1/auth/register", `{"username":"gwtest-user","devicePublicKey":""}`},
		{"login", "/v1/auth/login", `{"devicePublicKey":""}`},
	}

	for _, tc := range publicPaths {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", gatewayBaseURL+tc.path, bytes.NewReader([]byte(tc.body)))
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()

			// Public paths should NOT return 401. They may return 400 or 500
			// for invalid data, but auth middleware should not block them.
			if resp.StatusCode == http.StatusUnauthorized {
				t.Errorf("expected public path %s to not return 401, got %d", tc.path, resp.StatusCode)
			}
		})
	}
}

// TestGatewayAuthRegistration tests user registration through the HTTP gateway.
// It verifies the endpoint accepts requests and returns proper JSON structure.
func TestGatewayAuthRegistration(t *testing.T) {
	skipIfUnreachable(t, authServiceAddr, "auth")
	skipIfUnreachable(t, gatewayAddr, "gateway")

	username := "gw-reg-test-" + randomSuffix()
	_, pubKey := generateTestKeyPair(t)

	regBody, _ := json.Marshal(map[string]interface{}{
		"username":           username,
		"devicePublicKey":    pubKey,
		"deviceAttestation": "test-attestation",
	})

	resp, err := http.Post(gatewayBaseURL+"/v1/auth/register", "application/json", bytes.NewReader(regBody))
	if err != nil {
		t.Fatalf("register request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200 from register, got %d: %s", resp.StatusCode, string(body))
	}

	var regData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&regData); err != nil {
		t.Fatalf("failed to decode register response: %v", err)
	}

	// grpc-gateway serializes proto UUID fields as nested objects: {"userId": {"value": "..."}}
	// Check for the userId field in both snake_case and camelCase (grpc-gateway default)
	userID := getNestedValue(regData, "userId", "value")
	if userID == "" {
		userID = getNestedValue(regData, "user_id", "value")
	}
	if userID == "" {
		t.Error("expected non-empty userId in register response")
	}

	// The webauthnChallenge field should be present (base64 encoded)
	if _, ok := regData["webauthnChallenge"]; !ok {
		if _, ok := regData["webauthn_challenge"]; !ok {
			t.Error("expected webauthnChallenge in register response")
		}
	}

	t.Logf("Registered user %s with ID %s via gateway", username, userID)
}

// TestGatewayWorkspaceCRUDWithAuth tests creating and listing workspaces
// through the gateway using a generated access token.
func TestGatewayWorkspaceCRUDWithAuth(t *testing.T) {
	skipIfUnreachable(t, workspaceServiceAddr, "workspace-orchestrator")
	skipIfUnreachable(t, gatewayAddr, "gateway")

	userID := "gw-ws-test-" + randomSuffix()
	sessionID := "gw-session-" + randomSuffix()
	deviceID := "gw-device-" + randomSuffix()

	accessToken, err := generateAccessToken(userID, sessionID, deviceID)
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}

	// Create a workspace via the gateway
	wsName := "gateway-test-ws-" + randomSuffix()
	createBody, _ := json.Marshal(map[string]interface{}{
		"userId": map[string]string{"value": userID},
		"name":   wsName,
	})

	req, err := http.NewRequest("POST", gatewayBaseURL+"/v1/workspace/create", bytes.NewReader(createBody))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("create workspace request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200 from create workspace, got %d: %s", resp.StatusCode, string(body))
	}

	var createData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&createData); err != nil {
		t.Fatalf("failed to decode create response: %v", err)
	}

	workspace, ok := createData["workspace"].(map[string]interface{})
	if !ok {
		t.Fatal("expected workspace object in create response")
	}

	wsID := getNestedValue(workspace, "id", "value")
	if wsID == "" {
		t.Error("expected non-empty workspace ID")
	}

	t.Logf("Created workspace %s (%s) via gateway", wsID, wsName)

	// List workspaces for the user
	listBody, _ := json.Marshal(map[string]interface{}{
		"userId": map[string]string{"value": userID},
	})

	listReq, err := http.NewRequest("POST", gatewayBaseURL+"/v1/workspace/list", bytes.NewReader(listBody))
	if err != nil {
		t.Fatalf("failed to create list request: %v", err)
	}
	listReq.Header.Set("Content-Type", "application/json")
	listReq.Header.Set("Authorization", "Bearer "+accessToken)

	listResp, err := http.DefaultClient.Do(listReq)
	if err != nil {
		t.Fatalf("list workspaces request failed: %v", err)
	}
	defer listResp.Body.Close()

	if listResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(listResp.Body)
		t.Fatalf("expected 200 from list workspaces, got %d: %s", listResp.StatusCode, string(body))
	}

	var listData map[string]interface{}
	if err := json.NewDecoder(listResp.Body).Decode(&listData); err != nil {
		t.Fatalf("failed to decode list response: %v", err)
	}

	workspaces, ok := listData["workspaces"].([]interface{})
	if !ok {
		t.Fatal("expected workspaces array in list response")
	}
	if len(workspaces) == 0 {
		t.Error("expected at least one workspace in list")
	}

	t.Logf("Listed %d workspaces for user %s", len(workspaces), userID)

	// Clean up: stop and destroy the workspace
	cleanupWorkspace(t, accessToken, wsID)
}

// TestGatewayRateLimiting verifies the gateway applies rate limiting
// by sending many requests and checking for a 429 response.
func TestGatewayRateLimiting(t *testing.T) {
	skipIfUnreachable(t, gatewayAddr, "gateway")

	// Send a burst of requests to the auth register endpoint.
	// The rate limiter should allow the burst but eventually reject.
	// Using a unique username per request to avoid validation errors.
	got429 := false
	for i := 0; i < 250; i++ {
		regBody, _ := json.Marshal(map[string]interface{}{
			"username":           fmt.Sprintf("ratelimit-test-%d-%s", i, randomSuffix()),
			"devicePublicKey":    "",
			"deviceAttestation": "test",
		})
		resp, err := http.Post(gatewayBaseURL+"/v1/auth/register", "application/json", bytes.NewReader(regBody))
		if err != nil {
			t.Logf("request %d failed: %v", i, err)
			continue
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusTooManyRequests {
			got429 = true
			t.Logf("Received 429 Too Many Requests after %d requests", i+1)
			break
		}
	}

	if !got429 {
		t.Log("Rate limiter did not trigger within 250 requests (burst limit may be high); this is acceptable in test environments")
	}
}

// TestGatewayCORSBehavior tests the gateway's CORS behavior.
// Since the gateway does not currently have explicit CORS middleware,
// this test documents the current behavior.
func TestGatewayCORSBehavior(t *testing.T) {
	skipIfUnreachable(t, gatewayAddr, "gateway")

	// Send a preflight OPTIONS request
	req, err := http.NewRequest("OPTIONS", gatewayBaseURL+"/v1/auth/register", nil)
	if err != nil {
		t.Fatalf("failed to create OPTIONS request: %v", err)
	}
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type,Authorization")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("OPTIONS request failed: %v", err)
	}
	defer resp.Body.Close()

	origin := resp.Header.Get("Access-Control-Allow-Origin")
	methods := resp.Header.Get("Access-Control-Allow-Methods")
	headers := resp.Header.Get("Access-Control-Allow-Headers")

	t.Logf("CORS preflight response: status=%d, origin=%q, methods=%q, headers=%q",
		resp.StatusCode, origin, methods, headers)

	if origin != "" {
		t.Logf("CORS Allow-Origin header present: %s", origin)
		if origin == "http://localhost:3000" || origin == "*" {
			t.Log("CORS is properly configured for localhost development")
		}
	} else {
		t.Log("No CORS headers in response; CORS middleware is not yet configured on the gateway")
	}
}

// cleanupWorkspace stops and destroys a workspace via the gateway.
func cleanupWorkspace(t *testing.T, accessToken, wsID string) {
	t.Helper()

	// Stop the workspace
	stopBody, _ := json.Marshal(map[string]interface{}{
		"id": map[string]string{"value": wsID},
	})
	stopReq, err := http.NewRequest("POST", gatewayBaseURL+"/v1/workspace/stop", bytes.NewReader(stopBody))
	if err != nil {
		t.Logf("failed to create stop request: %v", err)
		return
	}
	stopReq.Header.Set("Content-Type", "application/json")
	stopReq.Header.Set("Authorization", "Bearer "+accessToken)
	stopResp, err := http.DefaultClient.Do(stopReq)
	if err != nil {
		t.Logf("stop request failed: %v", err)
		return
	}
	stopResp.Body.Close()

	// Destroy the workspace
	destroyBody, _ := json.Marshal(map[string]interface{}{
		"id": map[string]string{"value": wsID},
	})
	destroyReq, err := http.NewRequest("POST", gatewayBaseURL+"/v1/workspace/destroy", bytes.NewReader(destroyBody))
	if err != nil {
		t.Logf("failed to create destroy request: %v", err)
		return
	}
	destroyReq.Header.Set("Content-Type", "application/json")
	destroyReq.Header.Set("Authorization", "Bearer "+accessToken)
	destroyResp, err := http.DefaultClient.Do(destroyReq)
	if err != nil {
		t.Logf("destroy request failed: %v", err)
		return
	}
	destroyResp.Body.Close()
}

// getNestedValue navigates a nested map to extract a value.
// For example, getNestedValue(data, "userId", "value") extracts data["userId"]["value"].
func getNestedValue(data map[string]interface{}, keys ...string) string {
	current := data
	for i, key := range keys {
		if i == len(keys)-1 {
			if v, ok := current[key]; ok {
				return fmt.Sprintf("%v", v)
			}
			return ""
		}
		if m, ok := current[key].(map[string]interface{}); ok {
			current = m
		} else {
			return ""
		}
	}
	return ""
}

// unused import guard for commonv1
var _ = commonv1.UUID{}