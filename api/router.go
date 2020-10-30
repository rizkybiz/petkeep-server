package api

import "github.com/gorilla/mux"

func (s *server) routes() {
	// Setup the new router
	s.router = mux.NewRouter()
	// Tell new router to use logging middleware
	s.router.Use(s.httpLogger)

	// Set up the "no auth needed" paths
	s.router.Path("/api/" + version + "/login").Handler(s.handlerLogin()).Methods("POST")
	s.router.Path("/api" + version + "/users").Handler(s.handlerUsersCreate()).Methods("POST")

	// Set up the top level api subrouter
	api := s.router.PathPrefix("/api/" + version).Subrouter()
	// Protect all endpoints beyond here with token checks
	api.Use(s.isAuthenticated)

	// Set up user paths
	users := api.PathPrefix("/users").Subrouter().StrictSlash(true)
	users.HandleFunc("/", s.handlerUsersGetOne()).Methods("GET")
	users.HandleFunc("/reset_password", s.handlerResetPassword()).Methods("POST")

	// Set up pets paths
	pets := api.PathPrefix("/pets").Subrouter().StrictSlash(true)
	pets.HandleFunc("/", s.handlerPetsGetAll()).Methods("GET")
	pets.HandleFunc("/{id}", s.handlerPetsGetOne()).Methods("GET")
	pets.HandleFunc("/", s.handlerPetsCreate()).Methods("POST")
	pets.HandleFunc("/{id}", s.handlerPetsUpdate()).Methods("PUT")
	pets.HandleFunc("/{id}", s.handlerPetsDelete()).Methods("DELETE")
}
