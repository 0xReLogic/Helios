package adminapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewIPFilter(t *testing.T) {
	tests := []struct {
		name      string
		allowList []string
		denyList  []string
		wantErr   bool
	}{
		{
			name:      "valid CIDR notation",
			allowList: []string{"192.168.1.0/24", "10.0.0.0/8"},
			denyList:  []string{"203.0.113.0/24"},
			wantErr:   false,
		},
		{
			name:      "single IP addresses",
			allowList: []string{"127.0.0.1", "192.168.1.100"},
			denyList:  []string{"10.0.0.1"},
			wantErr:   false,
		},
		{
			name:      "mixed CIDR and single IPs",
			allowList: []string{"127.0.0.1", "192.168.1.0/24"},
			denyList:  []string{"10.0.0.1", "172.16.0.0/12"},
			wantErr:   false,
		},
		{
			name:      "invalid CIDR",
			allowList: []string{"invalid-cidr"},
			denyList:  []string{},
			wantErr:   true,
		},
		{
			name:      "empty lists",
			allowList: []string{},
			denyList:  []string{},
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := NewIPFilter(tt.allowList, tt.denyList)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewIPFilter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && filter == nil {
				t.Error("NewIPFilter() returned nil filter without error")
			}
		})
	}
}

func TestIPFilter_IsAllowed(t *testing.T) {
	tests := []struct {
		name      string
		allowList []string
		denyList  []string
		testIP    string
		want      bool
	}{
		{
			name:      "IP in allow list",
			allowList: []string{"192.168.1.0/24"},
			denyList:  []string{},
			testIP:    "192.168.1.100",
			want:      true,
		},
		{
			name:      "IP not in allow list",
			allowList: []string{"192.168.1.0/24"},
			denyList:  []string{},
			testIP:    "10.0.0.1",
			want:      false,
		},
		{
			name:      "IP in deny list",
			allowList: []string{},
			denyList:  []string{"203.0.113.0/24"},
			testIP:    "203.0.113.50",
			want:      false,
		},
		{
			name:      "IP not in deny list (empty allow list)",
			allowList: []string{},
			denyList:  []string{"203.0.113.0/24"},
			testIP:    "192.168.1.1",
			want:      true,
		},
		{
			name:      "deny takes precedence over allow",
			allowList: []string{"192.168.1.0/24"},
			denyList:  []string{"192.168.1.100/32"},
			testIP:    "192.168.1.100",
			want:      false,
		},
		{
			name:      "localhost allowed",
			allowList: []string{"127.0.0.1"},
			denyList:  []string{},
			testIP:    "127.0.0.1",
			want:      true,
		},
		{
			name:      "large subnet allow",
			allowList: []string{"10.0.0.0/8"},
			denyList:  []string{},
			testIP:    "10.123.45.67",
			want:      true,
		},
		{
			name:      "invalid IP",
			allowList: []string{"192.168.1.0/24"},
			denyList:  []string{},
			testIP:    "invalid-ip",
			want:      false,
		},
		{
			name:      "empty lists allow all",
			allowList: []string{},
			denyList:  []string{},
			testIP:    "1.2.3.4",
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := NewIPFilter(tt.allowList, tt.denyList)
			if err != nil {
				t.Fatalf("NewIPFilter() error = %v", err)
			}

			got := filter.IsAllowed(tt.testIP)
			if got != tt.want {
				t.Errorf("IsAllowed(%s) = %v, want %v", tt.testIP, got, tt.want)
			}
		})
	}
}

func TestIPFilter_Middleware(t *testing.T) {
	tests := []struct {
		name           string
		allowList      []string
		denyList       []string
		clientIP       string
		expectedStatus int
	}{
		{
			name:           "allowed IP",
			allowList:      []string{"192.168.1.0/24"},
			denyList:       []string{},
			clientIP:       "192.168.1.100",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "blocked IP",
			allowList:      []string{"192.168.1.0/24"},
			denyList:       []string{},
			clientIP:       "10.0.0.1",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "denied IP",
			allowList:      []string{},
			denyList:       []string{"203.0.113.0/24"},
			clientIP:       "203.0.113.50",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "localhost allowed",
			allowList:      []string{"127.0.0.1"},
			denyList:       []string{},
			clientIP:       "127.0.0.1",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := NewIPFilter(tt.allowList, tt.denyList)
			if err != nil {
				t.Fatalf("NewIPFilter() error = %v", err)
			}

			// Create a test handler
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("OK"))
			})

			// Wrap with IP filter middleware
			filteredHandler := filter.Middleware(handler)

			// Create test request
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tt.clientIP + ":12345"

			// Record response
			rr := httptest.NewRecorder()
			filteredHandler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, tt.expectedStatus)
			}
		})
	}
}

func TestParseCIDR(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid CIDR",
			input:   "192.168.1.0/24",
			wantErr: false,
		},
		{
			name:    "valid single IPv4",
			input:   "192.168.1.1",
			wantErr: false,
		},
		{
			name:    "valid IPv6 CIDR",
			input:   "2001:db8::/32",
			wantErr: false,
		},
		{
			name:    "valid single IPv6",
			input:   "2001:db8::1",
			wantErr: false,
		},
		{
			name:    "invalid CIDR",
			input:   "invalid",
			wantErr: true,
		},
		{
			name:    "invalid IP",
			input:   "999.999.999.999",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ipNet, err := parseCIDR(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseCIDR() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && ipNet == nil {
				t.Error("parseCIDR() returned nil without error")
			}
		})
	}
}
