package api

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/context"
)

func (s *server) isAuthenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := validateToken(r)
		if err != nil {
			log.Println(err)
			s.respond(w, r, nil, "unauthorized", http.StatusUnauthorized)
			return
		}
		context.Set(r, "userID", id)
		next.ServeHTTP(w, r)
	})
}

func (s *server) httpLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		latency := time.Since(start)
		s.logger.Info().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("user_agent", r.UserAgent()).
			Str("referrer", r.Referer()).
			Str("protocol", r.Proto).
			Dur("latency", latency).Send()
	})
}
