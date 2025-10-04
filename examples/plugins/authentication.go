package plugins

import (
	"net/http"
)

func init() {
	RegisterBuiltin("custom-auth", func(name string, cfg map[string]interface{}) (Middleware, error) {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				apiKey := r.Header.Get("X-API-Key")
				if apiKey != "super-secret-key" {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
				next.ServeHTTP(w, r)
			})
		}, nil
	})
}
