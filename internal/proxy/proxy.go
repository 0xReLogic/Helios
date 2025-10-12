package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/0xReLogic/Helios/internal/config"
	"github.com/0xReLogic/Helios/internal/logging"
)

// ReverseProxy represents the core reverse proxy functionality
type ReverseProxy struct {
	config *config.Config
	proxy  *httputil.ReverseProxy
}

// NewReverseProxy creates a new reverse proxy instance
func NewReverseProxy(cfg *config.Config) (*ReverseProxy, error) {
	// Use the first backend as the default target
	if len(cfg.Backends) == 0 {
		return nil, fmt.Errorf("no backends configured")
	}

	backendURL, err := url.Parse(cfg.Backends[0].Address)
	if err != nil {
		return nil, err
	}

	proxy := httputil.NewSingleHostReverseProxy(backendURL)

	// Add custom error handler
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		logging.WithContext(r.Context()).Error().Err(err).Msg("error proxying request")
		w.WriteHeader(http.StatusBadGateway)
		_, writeErr := w.Write([]byte("Backend server is not available"))
		if writeErr != nil {
			logging.WithContext(r.Context()).Error().Err(writeErr).Msg("error writing proxy error response")
		}
	}

	return &ReverseProxy{
		config: cfg,
		proxy:  proxy,
	}, nil
}

// ServeHTTP implements the http.Handler interface
func (rp *ReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logging.WithContext(r.Context()).Info().Str("method", r.Method).Str("path", r.URL.Path).Msg("proxying request")
	rp.proxy.ServeHTTP(w, r)
}
