package integration

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	authv1 "github.com/nomados/nomados/packages/shared-types/gen/auth/v1"
	sessionv1 "github.com/nomados/nomados/packages/shared-types/gen/session/v1"
	workspacev1 "github.com/nomados/nomados/packages/shared-types/gen/workspace/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// randomSuffix generates a short random suffix for unique test identifiers.
func randomSuffix() string {
	return uuid.New().String()[:8]
}

// Service addresses from environment with defaults.
var (
	authServiceAddr      = envOr("AUTH_SERVICE_ADDR", "localhost:50051")
	sessionServiceAddr   = envOr("SESSION_SERVICE_ADDR", "localhost:50052")
	workspaceServiceAddr = envOr("WORKSPACE_SERVICE_ADDR", "localhost:50053")
	vaultServiceAddr     = envOr("VAULT_SERVICE_ADDR", "localhost:50057")
	gatewayAddr          = envOr("GATEWAY_ADDR", "localhost:8080")
)

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// newAuthClient creates a gRPC client for the auth service.
// Skips the test if the service is not available.
func newAuthClient(t *testing.T) (authv1.AuthServiceClient, *grpc.ClientConn) {
	t.Helper()
	conn, err := grpc.NewClient(authServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("failed to connect to auth service at %s: %v", authServiceAddr, err)
	}
	return authv1.NewAuthServiceClient(conn), conn
}

// newSessionClient creates a gRPC client for the session service.
// Skips the test if the service is not available.
func newSessionClient(t *testing.T) (sessionv1.SessionServiceClient, *grpc.ClientConn) {
	t.Helper()
	conn, err := grpc.NewClient(sessionServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("failed to connect to session service at %s: %v", sessionServiceAddr, err)
	}
	return sessionv1.NewSessionServiceClient(conn), conn
}

// newWorkspaceClient creates a gRPC client for the workspace orchestrator.
// Skips the test if the service is not available.
func newWorkspaceClient(t *testing.T) (workspacev1.WorkspaceServiceClient, *grpc.ClientConn) {
	t.Helper()
	conn, err := grpc.NewClient(workspaceServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("failed to connect to workspace service at %s: %v", workspaceServiceAddr, err)
	}
	return workspacev1.NewWorkspaceServiceClient(conn), conn
}

// skipIfServiceUnavailable checks if a gRPC service is reachable.
// If not, it skips the test with a helpful message.
// NOTE: This function is not currently used by the integration tests,
// which use skipIfUnreachable instead. Kept for future use.
func skipIfServiceUnavailable(t *testing.T, addr, serviceName string) {
	t.Helper()
	_ = context.Background() // suppress unused import warning
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Skipf("%s service not available at %s: %v", serviceName, addr, err)
	}
	defer conn.Close()

	// Try to connect by issuing a simple ping
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if !conn.WaitForStateChange(ctx, conn.GetState()) {
		// Timeout means we couldn't connect
		t.Skipf("%s service not available at %s (connection timeout)", serviceName, addr)
	}
}

// skipIfUnreachable attempts a quick TCP connection to the address.
// If it fails, it skips the test.
func skipIfUnreachable(t *testing.T, addr, serviceName string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var d net.Dialer
	_, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		t.Skipf("%s service not available at %s: %v", serviceName, addr, err)
	}
}

// generateTestKeyPair generates an ECDSA P-256 key pair for test device keys.
func generateTestKeyPair(t *testing.T) (*ecdsa.PrivateKey, []byte) {
	t.Helper()
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatalf("failed to marshal public key: %v", err)
	}
	return privateKey, publicKeyBytes
}

// generateTestTLSCertificate creates a self-signed TLS certificate pair for testing mTLS.
func generateTestTLSCertificate(t *testing.T) (tls.Certificate, *x509.Certificate) {
	t.Helper()
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"NomadOS Test"},
			CommonName:   "test.nomados.dev",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(1 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:              []string{"localhost"},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatalf("failed to create certificate: %v", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		t.Fatalf("failed to parse certificate: %v", err)
	}

	tlsCert := tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  privateKey,
	}

	return tlsCert, cert
}

// generateInvalidTestCertificate creates an expired or otherwise invalid certificate for negative tests.
func generateInvalidTestCertificate(t *testing.T) tls.Certificate {
	t.Helper()
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Organization: []string{"Invalid Test Org"},
			CommonName:   "invalid.nomados.dev",
		},
		NotBefore:             time.Now().Add(-48 * time.Hour), // Expired: started 2 days ago
		NotAfter:              time.Now().Add(-24 * time.Hour), // Expired: ended 1 day ago
		KeyUsage:              x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatalf("failed to create invalid certificate: %v", err)
	}

	return tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  privateKey,
	}
}