package examples

import (
	"fmt"
	"net/http"

	"github.com/0xReLogic/Helios/internal/plugins"
)

func init() {
	plugins.RegisterBuiltin("custom-auth", func(name string, cfg map[string]interface{}) (plugins.Middleware, error) {
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
