package plugins

import (
	"fmt"
	"net/http"
)

func init() {
	RegisterBuiltin("custom-auth", func(name string, cfg map[string]interface{}) (Middleware, error) {
		apiKey, ok := cfg["apiKey"].(string)
		if !ok || apiKey == "" {
			return nil, fmt.Errorf("apiKey is required in config for %s plugin", name)
		}

		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("X-API-Key") != apiKey {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
				next.ServeHTTP(w, r)
			})
		}, nil
	})
}
