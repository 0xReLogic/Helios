package plugins

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

func init() {
	RegisterBuiltin("request-id", func(name string, cfg map[string]interface{}) (Middleware, error) {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Generate a new UUID
				b := make([]byte, 16)
				rand.Read(b)
				idStr := hex.EncodeToString(b)

				// Set the request header
				r.Header.Set("X-Request-ID", idStr)

				// Set the response header
				w.Header().Set("X-Request-ID", idStr)

				next.ServeHTTP(w, r)
			})
		}, nil
	})
}
