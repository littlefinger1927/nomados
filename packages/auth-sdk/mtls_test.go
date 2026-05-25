package authsdk

import "testing"

func TestMTLSConfigRequiresFiles(t *testing.T) {
	cfg := &MTLSConfig{
		CertFile: "/nonexistent/cert.pem",
		KeyFile:  "/nonexistent/key.pem",
		CAFile:   "/nonexistent/ca.pem",
	}
	_, err := cfg.TLSConfig()
	if err == nil {
		t.Error("expected error for nonexistent cert files")
	}
}