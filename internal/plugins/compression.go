package plugins

import (
	"fmt"
	"net/http"
)

func parseGzipConfig(cfg map[string]interface{}) (int, int, []string, error) {
	// numbers are unmarshalled into float64 by default
	levelFloat, ok := cfg["level"].(float64)
	if !ok {
		return 0, 0, nil, fmt.Errorf("expected level for gzip config")
	}
	level := int(levelFloat)

	minSizeFloat, ok := cfg["min_size"].(float64)
	if !ok {
		return 0, 0, nil, fmt.Errorf("expected min_size for gzip config")
	}
	minSize := int(minSizeFloat)

	rawTypes, ok := cfg["content_types"].([]interface{})
	if !ok {
		return 0, 0, nil, fmt.Errorf("expected content_types to be a list of strings")
	}

	contentTypes := make([]string, 0, len(rawTypes))
	for _, v := range rawTypes {
		s, ok := v.(string)
		if !ok {
			return 0, 0, nil, fmt.Errorf("all content_types must be string")
		}
		contentTypes = append(contentTypes, s)
	}
	return level, minSize, contentTypes, nil
}

// Config example :
// plugins:
//
//	enabled: true
//	chain:
//	  - name: gzip
//	    config:
//	      level: 6  # Compression level (1=fast, 9=best)
//	      min_size: 1024  # Only compress responses >= 1KB
//	      content_types:
//	        - "text/html"
//	        - "text/css"
//	        - "application/json"
//	        - "application/javascript"
func init() {
	RegisterBuiltin("gzip", func(name string, cfg map[string]interface{}) (Middleware, error) {
		level, minSize, contentTypes, err := parseGzipConfig(cfg)
		if err != nil {
			return nil, err
		}
		return func(next http.Handler) http.Handler {

			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, r)

			})
		}, nil
	})
}
