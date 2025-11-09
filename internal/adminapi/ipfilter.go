package adminapi

import (
	"net"
	"net/http"

	"github.com/0xReLogic/Helios/internal/logging"
	"github.com/0xReLogic/Helios/internal/utils"
)

// IPFilter provides IP-based access control with allow/deny lists
type IPFilter struct {
	allowList []*net.IPNet
	denyList  []*net.IPNet
}

// NewIPFilter creates a new IP filter with the given allow and deny lists
func NewIPFilter(allowList, denyList []string) (*IPFilter, error) {
	filter := &IPFilter{
		allowList: make([]*net.IPNet, 0, len(allowList)),
		denyList:  make([]*net.IPNet, 0, len(denyList)),
	}

	// Parse allow list
	for _, cidr := range allowList {
		ipNet, err := parseCIDR(cidr)
		if err != nil {
			return nil, err
		}
		filter.allowList = append(filter.allowList, ipNet)
	}

	// Parse deny list
	for _, cidr := range denyList {
		ipNet, err := parseCIDR(cidr)
		if err != nil {
			return nil, err
		}
		filter.denyList = append(filter.denyList, ipNet)
	}

	return filter, nil
}

// parseCIDR parses a CIDR notation or single IP address
func parseCIDR(cidr string) (*net.IPNet, error) {
	// Check if it's already in CIDR notation
	_, ipNet, err := net.ParseCIDR(cidr)
	if err == nil {
		return ipNet, nil
	}

	// Try parsing as a single IP address
	ip := net.ParseIP(cidr)
	if ip == nil {
		return nil, err // Return original CIDR parse error
	}

	// Convert single IP to CIDR notation
	if ip.To4() != nil {
		// IPv4
		_, ipNet, _ = net.ParseCIDR(cidr + "/32")
	} else {
		// IPv6
		_, ipNet, _ = net.ParseCIDR(cidr + "/128")
	}

	return ipNet, nil
}

// IsAllowed checks if the given IP address is allowed
func (f *IPFilter) IsAllowed(ip string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	// Check deny list first (deny takes precedence)
	for _, ipNet := range f.denyList {
		if ipNet.Contains(parsedIP) {
			return false
		}
	}

	// If allow list is empty, allow all (except denied)
	if len(f.allowList) == 0 {
		return true
	}

	// Check allow list
	for _, ipNet := range f.allowList {
		if ipNet.Contains(parsedIP) {
			return true
		}
	}

	// Not in allow list
	return false
}

// Middleware returns an HTTP middleware that filters requests based on IP
func (f *IPFilter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := utils.GetClientIP(r)

		if !f.IsAllowed(clientIP) {
			logging.WithContext(r.Context()).Warn().
				Str("client_ip", clientIP).
				Str("path", r.URL.Path).
				Msg("IP blocked by filter")

			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte("Forbidden: IP address not allowed"))
			return
		}

		next.ServeHTTP(w, r)
	})
}
