package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/nomados/nomados/services/gateway-service/internal/config"
)

// Route defines a path prefix to backend service mapping.
type Route struct {
	Prefix    string
	TargetURL *url.URL
	Proxy     *httputil.ReverseProxy
}

// Router holds all route definitions and dispatches requests.
type Router struct {
	routes []*Route
}

// NewRouter creates a router with routes based on the provided config.
func NewRouter(cfg *config.Config) *Router {
	routes := []*Route{
		newRoute("/auth", cfg.AuthAddr),
		newRoute("/session", cfg.SessionAddr),
		newRoute("/workspace", cfg.WorkspaceAddr),
		newRoute("/browser", cfg.BrowserAddr),
		newRoute("/stream", cfg.StreamingAddr),
		newRoute("/file", cfg.FileAddr),
		newRoute("/vault", cfg.VaultAddr),
	}
	return &Router{routes: routes}
}

func newRoute(prefix, backendAddr string) *Route {
	targetURL, _ := url.Parse(fmt.Sprintf("http://%s", backendAddr))
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Customize the Director to preserve the original path
	defaultDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		defaultDirector(req)
		req.Host = targetURL.Host
	}

	return &Route{
		Prefix:    prefix,
		TargetURL: targetURL,
		Proxy:     proxy,
	}
}

// ServeHTTP routes the request to the appropriate backend service.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	for _, route := range r.routes {
		if strings.HasPrefix(req.URL.Path, route.Prefix+"/") || req.URL.Path == route.Prefix {
			route.Proxy.ServeHTTP(w, req)
			return
		}
	}
	http.NotFound(w, req)
}