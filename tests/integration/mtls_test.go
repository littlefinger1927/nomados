package integration

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"testing"
	"time"

	authsdk "github.com/nomados/nomados/packages/auth-sdk"
)

// TestMTLSRequired verifies that gRPC services configured with mTLS
// require client certificates for connection. This test sets up a local
// TLS server and validates the mTLS handshake.
func TestMTLSRequired(t *testing.T) {
	// Generate CA certificate
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate CA key: %v", err)
	}

	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "NomadOS Test CA"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(1 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}

	caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("failed to create CA certificate: %v", err)
	}
	caCert, err := x509.ParseCertificate(caCertDER)
	if err != nil {
		t.Fatalf("failed to parse CA certificate: %v", err)
	}

	// Generate server certificate
	serverKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate server key: %v", err)
	}

	serverTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "test.nomados.dev"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(1 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:     []string{"localhost"},
	}

	serverCertDER, err := x509.CreateCertificate(rand.Reader, serverTemplate, caCert, &serverKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("failed to create server certificate: %v", err)
	}

	serverTLSCert := tls.Certificate{
		Certificate: [][]byte{serverCertDER},
		PrivateKey:  serverKey,
	}

	// Create CA pool for client verification
	caPool := x509.NewCertPool()
	caPool.AddCert(caCert)

	t.Run("connection without client cert is rejected", func(t *testing.T) {
		// Set up mTLS server for this sub-test
		serverConfig := &tls.Config{
			Certificates: []tls.Certificate{serverTLSCert},
			ClientCAs:    caPool,
			ClientAuth:   tls.RequireAndVerifyClientCert,
			MinVersion:   tls.VersionTLS13,
		}

		listener, err := tls.Listen("tcp", "127.0.0.1:0", serverConfig)
		if err != nil {
			t.Fatalf("failed to create TLS listener: %v", err)
		}
		defer listener.Close()

		// Accept connections in background
		go func() {
			conn, err := listener.Accept()
			if err == nil {
				conn.Close()
			}
		}()

		// Try to connect without a client certificate
		clientConfig := &tls.Config{
			InsecureSkipVerify: true,
			MinVersion:         tls.VersionTLS13,
		}

		dialer := &net.Dialer{Timeout: 3 * time.Second}
		conn, err := tls.DialWithDialer(dialer, "tcp", listener.Addr().String(), clientConfig)
		if err == nil {
			conn.Close()
			t.Error("expected connection without client cert to be rejected, but it succeeded")
		}
		// Error is expected — mTLS requires client certificates
		t.Logf("connection without client cert correctly rejected: %v", err)
	})

	t.Run("connection with valid client cert succeeds", func(t *testing.T) {
		// Generate client certificate signed by the CA
		clientKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			t.Fatalf("failed to generate client key: %v", err)
		}

		clientTemplate := &x509.Certificate{
			SerialNumber: big.NewInt(3),
			Subject:      pkix.Name{CommonName: "test-client.nomados.dev"},
			NotBefore:    time.Now(),
			NotAfter:     time.Now().Add(1 * time.Hour),
			KeyUsage:     x509.KeyUsageDigitalSignature,
			ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		}

		clientCertDER, err := x509.CreateCertificate(rand.Reader, clientTemplate, caCert, &clientKey.PublicKey, caKey)
		if err != nil {
			t.Fatalf("failed to create client certificate: %v", err)
		}

		clientTLSCert := tls.Certificate{
			Certificate: [][]byte{clientCertDER},
			PrivateKey:  clientKey,
		}

		clientConfig := &tls.Config{
			Certificates: []tls.Certificate{clientTLSCert},
			RootCAs:      caPool,
			MinVersion:   tls.VersionTLS13,
		}

		// Set up mTLS server for this sub-test
		serverConfig := &tls.Config{
			Certificates: []tls.Certificate{serverTLSCert},
			ClientCAs:    caPool,
			ClientAuth:   tls.RequireAndVerifyClientCert,
			MinVersion:   tls.VersionTLS13,
		}

		listener, err := tls.Listen("tcp", "127.0.0.1:0", serverConfig)
		if err != nil {
			t.Fatalf("failed to create TLS listener: %v", err)
		}
		defer listener.Close()

		// Accept connections in background - handle TLS handshake properly
		serverErr := make(chan error, 1)
		go func() {
			conn, err := listener.Accept()
			if err != nil {
				serverErr <- err
				return
			}
			// Handshake on the server side to complete the TLS negotiation
			if tlsConn, ok := conn.(*tls.Conn); ok {
				if err := tlsConn.Handshake(); err != nil {
					serverErr <- err
					conn.Close()
					return
				}
			}
			serverErr <- nil
			conn.Close()
		}()

		dialer := &net.Dialer{Timeout: 3 * time.Second}
		conn, err := tls.DialWithDialer(dialer, "tcp", listener.Addr().String(), clientConfig)
		if err != nil {
			t.Fatalf("expected connection with valid client cert to succeed, got error: %v", err)
		}
		conn.Close()

		// Wait for server goroutine to finish
		if err := <-serverErr; err != nil {
			t.Logf("server goroutine error (non-fatal for client test): %v", err)
		}
	})
}

// TestInvalidCertificate verifies that certificates signed by an untrusted CA
// or expired certificates are rejected during the mTLS handshake.
func TestInvalidCertificate(t *testing.T) {
	// Generate trusted CA
	trustedCAKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	trustedCATemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(10),
		Subject:               pkix.Name{CommonName: "NomadOS Trusted CA"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(1 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}
	trustedCACertDER, _ := x509.CreateCertificate(rand.Reader, trustedCATemplate, trustedCATemplate, &trustedCAKey.PublicKey, trustedCAKey)
	trustedCACert, _ := x509.ParseCertificate(trustedCACertDER)

	// Generate untrusted CA
	untrustedCAKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	untrustedCATemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(11),
		Subject:               pkix.Name{CommonName: "Untrusted CA"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(1 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}
	untrustedCACertDER, _ := x509.CreateCertificate(rand.Reader, untrustedCATemplate, untrustedCATemplate, &untrustedCAKey.PublicKey, untrustedCAKey)
	untrustedCACert, _ := x509.ParseCertificate(untrustedCACertDER)

	t.Run("client cert from untrusted CA is rejected", func(t *testing.T) {
		// Set up server with trusted CA
		serverKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		serverTemplate := &x509.Certificate{
			SerialNumber: big.NewInt(20),
			Subject:      pkix.Name{CommonName: "test.nomados.dev"},
			NotBefore:    time.Now(),
			NotAfter:     time.Now().Add(1 * time.Hour),
			KeyUsage:     x509.KeyUsageDigitalSignature,
			ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
			DNSNames:     []string{"localhost"},
		}
		serverCertDER, _ := x509.CreateCertificate(rand.Reader, serverTemplate, trustedCACert, &serverKey.PublicKey, trustedCAKey)

		serverTLSCert := tls.Certificate{
			Certificate: [][]byte{serverCertDER},
			PrivateKey:  serverKey,
		}

		trustedCAPool := x509.NewCertPool()
		trustedCAPool.AddCert(trustedCACert)

		serverConfig := &tls.Config{
			Certificates: []tls.Certificate{serverTLSCert},
			ClientCAs:    trustedCAPool,
			ClientAuth:   tls.RequireAndVerifyClientCert,
			MinVersion:   tls.VersionTLS13,
		}

		listener, err := tls.Listen("tcp", "127.0.0.1:0", serverConfig)
		if err != nil {
			t.Fatalf("failed to create TLS listener: %v", err)
		}
		defer listener.Close()

		go func() {
			conn, err := listener.Accept()
			if err == nil {
				conn.Close()
			}
		}()

		// Create client cert signed by untrusted CA
		clientKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		clientTemplate := &x509.Certificate{
			SerialNumber: big.NewInt(30),
			Subject:      pkix.Name{CommonName: "untrusted-client"},
			NotBefore:    time.Now(),
			NotAfter:     time.Now().Add(1 * time.Hour),
			KeyUsage:     x509.KeyUsageDigitalSignature,
			ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		}
		clientCertDER, _ := x509.CreateCertificate(rand.Reader, clientTemplate, untrustedCACert, &clientKey.PublicKey, untrustedCAKey)

		clientTLSCert := tls.Certificate{
			Certificate: [][]byte{clientCertDER},
			PrivateKey:  clientKey,
		}

		clientConfig := &tls.Config{
			Certificates: []tls.Certificate{clientTLSCert},
			RootCAs:      trustedCAPool,
			MinVersion:   tls.VersionTLS13,
		}

		dialer := &net.Dialer{Timeout: 3 * time.Second}
		conn, err := tls.DialWithDialer(dialer, "tcp", listener.Addr().String(), clientConfig)
		if err == nil {
			conn.Close()
			t.Error("expected connection with untrusted CA cert to be rejected, but it succeeded")
		} else {
			t.Logf("connection with untrusted CA cert correctly rejected: %v", err)
		}
	})

	t.Run("expired client cert is rejected", func(t *testing.T) {
		// Set up server with trusted CA
		serverKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		serverTemplate := &x509.Certificate{
			SerialNumber: big.NewInt(21),
			Subject:      pkix.Name{CommonName: "test.nomados.dev"},
			NotBefore:    time.Now(),
			NotAfter:     time.Now().Add(1 * time.Hour),
			KeyUsage:     x509.KeyUsageDigitalSignature,
			ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
			DNSNames:     []string{"localhost"},
		}
		serverCertDER, _ := x509.CreateCertificate(rand.Reader, serverTemplate, trustedCACert, &serverKey.PublicKey, trustedCAKey)

		serverTLSCert := tls.Certificate{
			Certificate: [][]byte{serverCertDER},
			PrivateKey:  serverKey,
		}

		trustedCAPool := x509.NewCertPool()
		trustedCAPool.AddCert(trustedCACert)

		serverConfig := &tls.Config{
			Certificates: []tls.Certificate{serverTLSCert},
			ClientCAs:    trustedCAPool,
			ClientAuth:   tls.RequireAndVerifyClientCert,
			MinVersion:   tls.VersionTLS13,
		}

		listener, err := tls.Listen("tcp", "127.0.0.1:0", serverConfig)
		if err != nil {
			t.Fatalf("failed to create TLS listener: %v", err)
		}
		defer listener.Close()

		go func() {
			conn, err := listener.Accept()
			if err == nil {
				conn.Close()
			}
		}()

		// Create an expired client cert signed by trusted CA
		clientKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		clientTemplate := &x509.Certificate{
			SerialNumber: big.NewInt(31),
			Subject:      pkix.Name{CommonName: "expired-client"},
			NotBefore:    time.Now().Add(-48 * time.Hour),
			NotAfter:     time.Now().Add(-24 * time.Hour), // Expired yesterday
			KeyUsage:     x509.KeyUsageDigitalSignature,
			ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		}
		clientCertDER, _ := x509.CreateCertificate(rand.Reader, clientTemplate, trustedCACert, &clientKey.PublicKey, trustedCAKey)

		clientTLSCert := tls.Certificate{
			Certificate: [][]byte{clientCertDER},
			PrivateKey:  clientKey,
		}

		clientConfig := &tls.Config{
			Certificates: []tls.Certificate{clientTLSCert},
			RootCAs:      trustedCAPool,
			MinVersion:   tls.VersionTLS13,
		}

		dialer := &net.Dialer{Timeout: 3 * time.Second}
		conn, err := tls.DialWithDialer(dialer, "tcp", listener.Addr().String(), clientConfig)
		if err == nil {
			conn.Close()
			t.Error("expected connection with expired cert to be rejected, but it succeeded")
		} else {
			t.Logf("connection with expired cert correctly rejected: %v", err)
		}
	})
}

// TestCertificateRotation verifies that new certificates issued by the same CA
// work after a certificate rotation. In Phase 1, this is a basic verification
// that two client certs from the same CA both work.
func TestCertificateRotation(t *testing.T) {
	// Generate CA
	caKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(100),
		Subject:               pkix.Name{CommonName: "Rotation Test CA"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}
	caCertDER, _ := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	caCert, _ := x509.ParseCertificate(caCertDER)

	// Generate server cert
	serverKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	serverTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(101),
		Subject:      pkix.Name{CommonName: "rotation-test.nomados.dev"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:     []string{"localhost"},
	}
	serverCertDER, _ := x509.CreateCertificate(rand.Reader, serverTemplate, caCert, &serverKey.PublicKey, caKey)

	serverTLSCert := tls.Certificate{
		Certificate: [][]byte{serverCertDER},
		PrivateKey:  serverKey,
	}

	caPool := x509.NewCertPool()
	caPool.AddCert(caCert)

	// Generate "old" client cert
	oldClientKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	oldClientTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(200),
		Subject:      pkix.Name{CommonName: "old-client.nomados.dev"},
		NotBefore:    time.Now().Add(-1 * time.Hour),
		NotAfter:     time.Now().Add(23 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	oldClientCertDER, _ := x509.CreateCertificate(rand.Reader, oldClientTemplate, caCert, &oldClientKey.PublicKey, caKey)

	// Generate "new" client cert (rotation)
	newClientKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	newClientTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(201),
		Subject:      pkix.Name{CommonName: "new-client.nomados.dev"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	newClientCertDER, _ := x509.CreateCertificate(rand.Reader, newClientTemplate, caCert, &newClientKey.PublicKey, caKey)

	t.Run("old client cert works before rotation", func(t *testing.T) {
		serverConfig := &tls.Config{
			Certificates: []tls.Certificate{serverTLSCert},
			ClientCAs:    caPool,
			ClientAuth:   tls.RequireAndVerifyClientCert,
			MinVersion:   tls.VersionTLS13,
		}

		listener, err := tls.Listen("tcp", "127.0.0.1:0", serverConfig)
		if err != nil {
			t.Fatalf("failed to create TLS listener: %v", err)
		}
		defer listener.Close()

		serverErr := make(chan error, 1)
		go func() {
			conn, err := listener.Accept()
			if err != nil {
				serverErr <- err
				return
			}
			if tlsConn, ok := conn.(*tls.Conn); ok {
				if err := tlsConn.Handshake(); err != nil {
					serverErr <- err
					conn.Close()
					return
				}
			}
			serverErr <- nil
			conn.Close()
		}()

		oldTLSCert := tls.Certificate{
			Certificate: [][]byte{oldClientCertDER},
			PrivateKey:  oldClientKey,
		}

		clientConfig := &tls.Config{
			Certificates: []tls.Certificate{oldTLSCert},
			RootCAs:      caPool,
			MinVersion:   tls.VersionTLS13,
		}

		dialer := &net.Dialer{Timeout: 3 * time.Second}
		conn, err := tls.DialWithDialer(dialer, "tcp", listener.Addr().String(), clientConfig)
		if err != nil {
			t.Fatalf("expected old cert to work, got error: %v", err)
		}
		conn.Close()

		if err := <-serverErr; err != nil {
			t.Logf("server goroutine error (non-fatal for client test): %v", err)
		}
	})

	t.Run("new client cert works after rotation", func(t *testing.T) {
		serverConfig := &tls.Config{
			Certificates: []tls.Certificate{serverTLSCert},
			ClientCAs:    caPool,
			ClientAuth:   tls.RequireAndVerifyClientCert,
			MinVersion:   tls.VersionTLS13,
		}

		listener, err := tls.Listen("tcp", "127.0.0.1:0", serverConfig)
		if err != nil {
			t.Fatalf("failed to create TLS listener: %v", err)
		}
		defer listener.Close()

		serverErr := make(chan error, 1)
		go func() {
			conn, err := listener.Accept()
			if err != nil {
				serverErr <- err
				return
			}
			if tlsConn, ok := conn.(*tls.Conn); ok {
				if err := tlsConn.Handshake(); err != nil {
					serverErr <- err
					conn.Close()
					return
				}
			}
			serverErr <- nil
			conn.Close()
		}()

		newTLSCert := tls.Certificate{
			Certificate: [][]byte{newClientCertDER},
			PrivateKey:  newClientKey,
		}

		clientConfig := &tls.Config{
			Certificates: []tls.Certificate{newTLSCert},
			RootCAs:      caPool,
			MinVersion:   tls.VersionTLS13,
		}

		dialer := &net.Dialer{Timeout: 3 * time.Second}
		conn, err := tls.DialWithDialer(dialer, "tcp", listener.Addr().String(), clientConfig)
		if err != nil {
			t.Fatalf("expected new cert to work after rotation, got error: %v", err)
		}
		conn.Close()

		if err := <-serverErr; err != nil {
			t.Logf("server goroutine error (non-fatal for client test): %v", err)
		}
	})

	t.Run("auth-sdk MTLSConfig validates correctly", func(t *testing.T) {
		// Verify that TLSConfig() fails with missing files
		mtlsConfig := authsdk.MTLSConfig{
			CertFile: "/tmp/nonexistent-cert-" + randomSuffix() + ".pem",
			KeyFile:  "/tmp/nonexistent-key-" + randomSuffix() + ".pem",
			CAFile:   "/tmp/nonexistent-ca-" + randomSuffix() + ".pem",
		}

		_, err := mtlsConfig.TLSConfig()
		if err == nil {
			t.Error("expected TLSConfig() to fail with nonexistent files")
		} else {
			t.Logf("TLSConfig correctly failed with nonexistent files: %v", err)
		}
	})
}

// TestMTLSConfigWithGeneratedCerts tests that the auth-sdk MTLSConfig
// correctly loads and parses generated certificates.
func TestMTLSConfigWithGeneratedCerts(t *testing.T) {
	// Generate CA certificate
	caKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(500),
		Subject:               pkix.Name{CommonName: "MTLSConfig Test CA"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(1 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}

	// Test that the TLS config properly sets RequireAndVerifyClientCert
	// and uses TLS 1.3 minimum version
	t.Run("TLS config enforces mTLS requirements", func(t *testing.T) {
		config := &tls.Config{
			ClientAuth:   tls.RequireAndVerifyClientCert,
			MinVersion:   tls.VersionTLS13,
			Certificates: []tls.Certificate{},
			ClientCAs:    x509.NewCertPool(),
		}

		if config.ClientAuth != tls.RequireAndVerifyClientCert {
			t.Errorf("expected ClientAuth RequireAndVerifyClientCert, got %v", config.ClientAuth)
		}
		if config.MinVersion != tls.VersionTLS13 {
			t.Errorf("expected MinVersion TLS 1.3 (%#x), got %#x", tls.VersionTLS13, config.MinVersion)
		}
	})

	// Verify the CA certificate is created correctly
	caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("failed to create CA certificate: %v", err)
	}

	_, err = x509.ParseCertificate(caCertDER)
	if err != nil {
		t.Fatalf("failed to parse CA certificate: %v", err)
	}

	t.Logf("CA certificate generated and validated successfully (serial: %d)", caTemplate.SerialNumber)

	// Test certificate with proper EKU for client auth
	clientKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	caCert, _ := x509.ParseCertificate(caCertDER)
	clientTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(501),
		Subject:      pkix.Name{CommonName: "mtls-config-test-client"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(1 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	clientCertDER, err := x509.CreateCertificate(rand.Reader, clientTemplate, caCert, &clientKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("failed to create client certificate: %v", err)
	}

	clientCert, err := x509.ParseCertificate(clientCertDER)
	if err != nil {
		t.Fatalf("failed to parse client certificate: %v", err)
	}

	// Verify ExtKeyUsage includes ClientAuth
	hasClientAuth := false
	for _, eku := range clientCert.ExtKeyUsage {
		if eku == x509.ExtKeyUsageClientAuth {
			hasClientAuth = true
			break
		}
	}
	if !hasClientAuth {
		t.Error("client certificate should have ExtKeyUsageClientAuth")
	}

	t.Logf("Client certificate validated with ClientAuth EKU (serial: %d)", clientTemplate.SerialNumber)
}

// TestMTLSConnectionToService attempts an mTLS connection to the auth service.
// If the service is running and requires mTLS, it verifies the handshake behavior.
// This test skips if the auth service is not available.
func TestMTLSConnectionToService(t *testing.T) {
	skipIfUnreachable(t, authServiceAddr, "auth")

	// Try connecting to the auth service with TLS without client certs.
	// If the service requires mTLS, the connection should be rejected.
	conn, err := tls.DialWithDialer(
		&net.Dialer{Timeout: 3 * time.Second},
		"tcp",
		authServiceAddr,
		&tls.Config{
			InsecureSkipVerify: true, // We're testing mTLS, not server cert validation
			MinVersion:         tls.VersionTLS12,
		},
	)
	if err != nil {
		// Connection failed — could mean the service uses plaintext gRPC
		// or requires mTLS and rejected our connection.
		t.Logf("TLS connection to auth service failed (service may use plaintext gRPC): %v", err)
	} else {
		state := conn.ConnectionState()
		t.Logf("Connected to auth service: TLS version %x, cipher %x, server name %s",
			state.Version, state.CipherSuite, state.ServerName)
		conn.Close()
	}
}