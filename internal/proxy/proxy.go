package proxy

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/0xReLogic/Helios/internal/config"
)

// ReverseProxy represents the core reverse proxy functionality
type ReverseProxy struct {
	config *config.Config
	proxy  *httputil.ReverseProxy
}

// NewReverseProxy creates a new reverse proxy instance
func NewReverseProxy(cfg *config.Config) (*ReverseProxy, error) {
	backendURL, err := url.Parse(cfg.Backend.Address)
	if err != nil {
		return nil, err
	}

	proxy := httputil.NewSingleHostReverseProxy(backendURL)

	// Add custom error handler
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("Error proxying request: %v", err)
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("Backend server is not available"))
	}

	return &ReverseProxy{
		config: cfg,
		proxy:  proxy,
	}, nil
}

// ServeHTTP implements the http.Handler interface
func (rp *ReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("Proxying request: %s %s", r.Method, r.URL.Path)
	rp.proxy.ServeHTTP(w, r)
}
