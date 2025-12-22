package middleware

import "net/http"

type CORSConfig struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
}

// NewCORS создаёт middleware для CORS
func NewCORS(cfg CORSConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := "*"
			if len(cfg.AllowedOrigins) > 0 {
				origin = cfg.AllowedOrigins[0] // для простоты, можно расширить
			}

			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", join(cfg.AllowedMethods))
			w.Header().Set("Access-Control-Allow-Headers", join(cfg.AllowedHeaders))

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func join(items []string) string {
	if len(items) == 0 {
		return ""
	}
	s := items[0]
	for _, item := range items[1:] {
		s += ", " + item
	}
	return s
}
