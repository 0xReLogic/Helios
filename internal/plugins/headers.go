package plugins

import (
	"fmt"
	"net/http"
)

// toStringMap converts a generic map to map[string]string if possible
func toStringMap(v interface{}) (map[string]string, error) {
	res := map[string]string{}
	if v == nil {
		return res, nil
	}
	m, ok := v.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("expected object for headers config")
	}
	for k, val := range m {
		s, ok := val.(string)
		if !ok {
			return nil, fmt.Errorf("header %s must be a string", k)
		}
		res[k] = s
	}
	return res, nil
}

// init registers a simple headers plugin
// Config example:
// plugins:
//   enabled: true
//   chain:
//     - name: headers
//       config:
//         set:
//           X-App: Helios
//         request_set:
//           X-From: LB
func init() {
	RegisterBuiltin("headers", func(name string, cfg map[string]interface{}) (Middleware, error) {
		setMap, err := toStringMap(cfg["set"])
		if err != nil {
			return nil, err
		}
		reqSetMap, err := toStringMap(cfg["request_set"])
		if err != nil {
			return nil, err
		}

		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// mutate request headers
				for k, v := range reqSetMap {
					r.Header.Set(k, v)
				}
				// set response headers before calling next so they are present
				for k, v := range setMap {
					w.Header().Set(k, v)
				}
				next.ServeHTTP(w, r)
			})
		}, nil
	})
}
