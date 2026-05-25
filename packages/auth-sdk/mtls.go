package authsdk

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

type MTLSConfig struct {
	CertFile string
	KeyFile  string
	CAFile   string
}

func (c *MTLSConfig) TLSConfig() (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("load key pair: %w", err)
	}

	caCert, err := os.ReadFile(c.CAFile)
	if err != nil {
		return nil, fmt.Errorf("read CA cert: %w", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to append CA cert")
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    caCertPool,
		ClientAuth:    tls.RequireAndVerifyClientCert,
		MinVersion:   tls.VersionTLS13,
	}, nil
}