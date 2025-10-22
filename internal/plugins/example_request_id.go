package plugins

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
)

func init() {
	RegisterBuiltin("request-id", func(name string, cfg map[string]interface{}) (Middleware, error) {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				b := make([]byte, 16)
				_, err := rand.Read(b)
				if err != nil {
					fmt.Printf("Error generating request ID: %v\n", err)
					next.ServeHTTP(w, r)
					return
				}
				idStr := hex.EncodeToString(b)

				r.Header.Set("X-Request-ID", idStr)

				w.Header().Set("X-Request-ID", idStr)

				next.ServeHTTP(w, r)
			})
		}, nil
	})
}
