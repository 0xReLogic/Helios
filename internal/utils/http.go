package utils

import (
	"net"
	"net/http"
	"strings"
)

// GetClientIP extracts the real client IP address from an HTTP request.
// It checks headers in order of priority: X-Forwarded-For, X-Real-IP, RemoteAddr.
// X-Forwarded-For format: "client, proxy1, proxy2, ..." - extracts first IP only
// For RemoteAddr, strips the port number using net.SplitHostPort.
// Supports both IPv4 and IPv6 addresses.
func GetClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if idx := strings.Index(xff, ","); idx > 0 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}

	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}
