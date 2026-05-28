package proxy

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

// FileProxy reverse-proxies file upload/download HTTP requests to the
// file-service HTTP server. grpc-gateway doesn't support streaming RPCs,
// so browsers need this reverse proxy for multipart upload and file download.
type FileProxy struct {
	target *url.URL
	proxy   *httputil.ReverseProxy
}

// NewFileProxy creates a reverse proxy to the file-service HTTP server.
func NewFileProxy(fileHTTPAddr string) *FileProxy {
	target, err := url.Parse("http://" + fileHTTPAddr)
	if err != nil {
		log.Fatalf("invalid file HTTP address: %v", err)
	}
	return &FileProxy{
		target: target,
		proxy:  httputil.NewSingleHostReverseProxy(target),
	}
}

// ServeHTTP proxies the request to the file-service HTTP server.
func (fp *FileProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.Host = fp.target.Host
	fp.proxy.ServeHTTP(w, r)
}