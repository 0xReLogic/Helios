package plugins

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/0xReLogic/Helios/internal/logging"
)

func init() {
	RegisterBuiltin("request-id", func(name string, cfg map[string]interface{}) (Middleware, error) {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				b := make([]byte, 16)
				_, err := rand.Read(b)
				if err != nil {
					logger := logging.WithContext(r.Context())
					logger.Error().Err(err).Msg("failed to generate request ID")
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
