package adminapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/0xReLogic/Helios/internal/config"
	"github.com/0xReLogic/Helios/internal/loadbalancer"
	"github.com/0xReLogic/Helios/internal/metrics"
)

func newTestLB(t *testing.T) *loadbalancer.LoadBalancer {
	t.Helper()
	cfg := &config.Config{
		LoadBalancer: config.LoadBalancerConfig{Strategy: "round_robin"},
		HealthChecks: config.HealthChecksConfig{
			Active:  config.ActiveHealthCheckConfig{Enabled: false},
			Passive: config.PassiveHealthCheckConfig{Enabled: false},
		},
	}
	lb, err := loadbalancer.NewLoadBalancer(cfg)
	if err != nil {
		t.Fatalf("failed to create lb: %v", err)
	}
	return lb
}

func TestAdminAPI_Health_NoAuth(t *testing.T) {
	lb := newTestLB(t)
	mc := metrics.NewMetricsCollector()
	mux := NewMux(lb, "", mc)

	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestAdminAPI_Metrics_WithAuth(t *testing.T) {
	lb := newTestLB(t)
	mc := metrics.NewMetricsCollector()
	mux := NewMux(lb, "secret", mc)

	// Without token -> 401
	req := httptest.NewRequest(http.MethodGet, "/v1/metrics", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", rec.Code)
	}

	// With token -> 200 and valid JSON
	req2 := httptest.NewRequest(http.MethodGet, "/v1/metrics", nil)
	req2.Header.Set("Authorization", "Bearer secret")
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200 with token, got %d", rec2.Code)
	}
	var out metrics.Metrics
	if err := json.Unmarshal(rec2.Body.Bytes(), &out); err != nil {
		t.Fatalf("invalid metrics json: %v", err)
	}
}

func TestAdminAPI_Backends_Add_List_Remove_WithAuth(t *testing.T) {
	lb := newTestLB(t)
	mc := metrics.NewMetricsCollector()
	mux := NewMux(lb, "secret", mc)

	// Initially zero backends
	reqList := httptest.NewRequest(http.MethodGet, "/v1/backends", nil)
	reqList.Header.Set("Authorization", "Bearer secret")
	recList := httptest.NewRecorder()
	mux.ServeHTTP(recList, reqList)
	if recList.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recList.Code)
	}
	var list []loadbalancer.BackendInfo
	if err := json.Unmarshal(recList.Body.Bytes(), &list); err != nil {
		t.Fatalf("invalid list json: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected 0 backends, got %d", len(list))
	}

	// Add backend
	addPayload := config.BackendConfig{Name: "b1", Address: "http://127.0.0.1:65530"}
	buf, _ := json.Marshal(addPayload)
	reqAdd := httptest.NewRequest(http.MethodPost, "/v1/backends/add", bytes.NewReader(buf))
	reqAdd.Header.Set("Authorization", "Bearer secret")
	recAdd := httptest.NewRecorder()
	mux.ServeHTTP(recAdd, reqAdd)
	if recAdd.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", recAdd.Code)
	}

	// List should have one backend
	reqList2 := httptest.NewRequest(http.MethodGet, "/v1/backends", nil)
	reqList2.Header.Set("Authorization", "Bearer secret")
	recList2 := httptest.NewRecorder()
	mux.ServeHTTP(recList2, reqList2)
	if recList2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recList2.Code)
	}
	if err := json.Unmarshal(recList2.Body.Bytes(), &list); err != nil {
		t.Fatalf("invalid list json: %v", err)
	}
	if len(list) != 1 || list[0].Name != "b1" {
		t.Fatalf("expected 1 backend named b1, got %+v", list)
	}

	// Remove backend
	rmPayload := map[string]string{"name": "b1"}
	rmBuf, _ := json.Marshal(rmPayload)
	reqRm := httptest.NewRequest(http.MethodPost, "/v1/backends/remove", bytes.NewReader(rmBuf))
	reqRm.Header.Set("Authorization", "Bearer secret")
	recRm := httptest.NewRecorder()
	mux.ServeHTTP(recRm, reqRm)
	if recRm.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recRm.Code)
	}

	// List should be empty again
	reqList3 := httptest.NewRequest(http.MethodGet, "/v1/backends", nil)
	reqList3.Header.Set("Authorization", "Bearer secret")
	recList3 := httptest.NewRecorder()
	mux.ServeHTTP(recList3, reqList3)
	if recList3.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recList3.Code)
	}
	if err := json.Unmarshal(recList3.Body.Bytes(), &list); err != nil {
		t.Fatalf("invalid list json: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected 0 backends, got %d", len(list))
	}
}

func TestAdminAPI_Strategy_Set_WithAuth(t *testing.T) {
	lb := newTestLB(t)
	mc := metrics.NewMetricsCollector()
	mux := NewMux(lb, "secret", mc)

	// Valid strategy
	body := []byte(`{"strategy":"least_connections"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/strategy", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// Invalid strategy
	body2 := []byte(`{"strategy":"nope"}`)
	req2 := httptest.NewRequest(http.MethodPost, "/v1/strategy", bytes.NewReader(body2))
	req2.Header.Set("Authorization", "Bearer secret")
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec2.Code)
	}
}
