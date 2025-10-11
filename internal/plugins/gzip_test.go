package plugins

import "testing"

func TestParseGzipConfig(t *testing.T) {
	cfg := map[string]interface{}{
		"level":         float64(6),
		"min_size":      float64(1024),
		"content_types": []interface{}{"text/html", "application/json"},
	}

	level, minSize, contentTypes, err := parseGzipConfig(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if level != 6 {
		t.Errorf("expected level=6, got %d", level)
	}
	if minSize != 1024 {
		t.Errorf("expected minSize=1024, got %d", minSize)
	}
	if len(contentTypes) != 2 || contentTypes[0] != "text/html" {
		t.Errorf("unexpected contentTypes: %v", contentTypes)
	}
}
