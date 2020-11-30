package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/context"
)

func (s *server) isAuthenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := validateToken(r)
		if err != nil {
			s.logger.Err(err).Msg("error parsing token")
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
			Str("path", r.URL.Path[1:]).
			Str("user_agent", r.UserAgent()).
			Str("referrer", r.Referer()).
			Str("protocol", r.Proto).
			Dur("latency", latency).Send()
	})
}

func (s *server) httpTiming(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.statsd == nil {
			next.ServeHTTP(w, r)
			return
		}
		path := r.URL.Path[1:]
		method := r.Method
		c := s.statsd.Clone()
		timer := c.NewTiming()

		next.ServeHTTP(w, r)

		timer.Send(fmt.Sprintf("%s.%s", method, path))
	})
}

func (s *server) httpCount(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.statsd == nil {
			next.ServeHTTP(w, r)
			return
		}
		path := r.URL.Path[1:]
		method := r.Method
		c := s.statsd.Clone()

		next.ServeHTTP(w, r)
		c.Count(fmt.Sprintf("%s.%s", method, path), 1)
	})
}
