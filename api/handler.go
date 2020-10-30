package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/badoux/checkmail"
	"github.com/golang/gddo/httputil/header"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

func (s *server) respond(w http.ResponseWriter, r *http.Request, data interface{}, errMsg string, status int) {

	// This could really be useful for a singular place to do something like
	// check the "accepts" header in the request and respond appropriately.
	// For now it's a simple helper.
	if errMsg != "" {
		http.Error(w, errMsg, status)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if data != nil {
		err := json.NewEncoder(w).Encode(data)
		if err != nil {
			log.Printf("Error encoding data: %s", err)
			return
		}
	}
}

func (s *server) decode(w http.ResponseWriter, r *http.Request, v interface{}) error {

	// First check 'Content-Type' header and make sure we're receiving JSON
	if r.Header.Get("Content-Type") != "" {
		value, _ := header.ParseValueAndParams(r.Header, "Content-Type")
		if value != "application/json" {
			msg := "Content-Type header is not application/json"
			s.respond(w, r, nil, msg, http.StatusUnsupportedMediaType)
			return errors.New(msg)
		}
	}
	// Set the max request Body size
	r.Body = http.MaxBytesReader(w, r.Body, 1048576)

	// Create the decoder and attempt to decode the received JSON,
	// returning the corresponding error if there's a failure.
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	err := dec.Decode(&v)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		switch {
		case errors.As(err, &syntaxError):
			msg := fmt.Sprintf("Request body contains badly-formed JSON (at position %d)", syntaxError.Offset)
			s.respond(w, r, nil, msg, http.StatusBadRequest)
			return errors.New(msg)

		case errors.Is(err, io.ErrUnexpectedEOF):
			msg := fmt.Sprintf("Request body contains poorly-formed JSON")
			s.respond(w, r, nil, msg, http.StatusBadRequest)
			return errors.New(msg)

		case errors.As(err, &unmarshalTypeError):
			msg := fmt.Sprintf("Request body contains an invalid value for the %q field (at position %d)", unmarshalTypeError.Field, unmarshalTypeError.Offset)
			s.respond(w, r, nil, msg, http.StatusBadRequest)
			return errors.New(msg)

		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			msg := fmt.Sprintf("Request body contains unknown field %s", fieldName)
			s.respond(w, r, nil, msg, http.StatusBadRequest)
			return errors.New(msg)

		case errors.Is(err, io.EOF):
			msg := "Request body must not be empty"
			s.respond(w, r, nil, msg, http.StatusBadRequest)
			return errors.New(msg)

		case err.Error() == "http: request body too large":
			msg := "Request body must not be larger than 1MB"
			s.respond(w, r, nil, msg, http.StatusRequestEntityTooLarge)
			return errors.New(msg)

		default:
			return err
		}
	}

	return nil
}

func (s *server) handlerLogin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Create user and decode into it
		var usr user
		err := s.decode(w, r, &usr)
		if err != nil {
			log.Printf("error: %s", err)
			return
		}

		// Check for email and password
		if usr.Email == "" || usr.Password == "" {
			s.respond(w, r, nil, "must provide username and password", http.StatusBadRequest)
			return
		}

		// Check DB for user and compare passwords
		id, err := s.dbLogin(usr.Email, usr.Password)
		if err != nil {
			log.Println(err)
			s.respond(w, r, nil, "incorrect password", http.StatusUnauthorized)
			return
		}

		// Generate access token
		accTokenStr, err := generateToken(uint(id), time.Now().Add(time.Hour*24))
		if err != nil {
			log.Println(err)
			s.respond(w, r, nil, "internal server error", http.StatusInternalServerError)
			return
		}
		// Generate refresh token
		refrTokenStr, err := generateToken(uint(id), time.Now().Add(time.Hour*168))
		if err != nil {
			log.Println(err)
			s.respond(w, r, nil, "internal server error", http.StatusInternalServerError)
			return
		}
		// Respond with access and refresh tokens
		s.respond(w, r, token{AccessTkn: accTokenStr, RefreshTkn: refrTokenStr}, "", http.StatusOK)
	}
}

func (s *server) handlerUsersCreate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ts := time.Now()
		var usr user
		err := s.decode(w, r, &usr)
		if err != nil {
			log.Println(err)
			return
		}

		// Check for email and password
		if usr.Email == "" || usr.Password == "" {
			s.respond(w, r, nil, "must provide username and password", http.StatusBadRequest)
			return
		}

		// Validate the email address
		err = checkmail.ValidateFormat(usr.Email)
		if err != nil {
			log.Println(err)
			s.respond(w, r, nil, "email format is invalid", http.StatusBadRequest)
			return
		}

		//Create hashed version of password
		hashedPass, err := bcrypt.GenerateFromPassword([]byte(usr.Password), bcrypt.DefaultCost)
		if err != nil {
			log.Println(err)
			s.respond(w, r, nil, "internal server error", http.StatusInternalServerError)
			return
		}

		// Fill the user struct to send to the DB
		usr.Password = string(hashedPass)
		usr.CreatedAt = ts
		usr.UpdatedAt = ts

		//Create the user in the DB
		id, err := s.dbUsersCreate(usr)
		if err != nil {
			log.Println(err)
			s.respond(w, r, nil, "error creating user", http.StatusInternalServerError)
			return
		}

		// Set the users ID and return
		usr.ID = uint(id)
		s.respond(w, r, usr, "", http.StatusCreated)
	}
}

func (s *server) handlerUsersGetOne() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Get user ID from the authenticated context
		id := context.Get(r, "userID")
		idStr := fmt.Sprintf("%v", id)
		idInt, _ := strconv.Atoi(idStr)

		// Get user from database and respond
		u, err := s.dbUsersGetOne(int64(idInt))
		if err != nil {
			s.respond(w, r, nil, err.Error(), http.StatusInternalServerError)
		}
		s.respond(w, r, u, "", http.StatusOK)
	}
}

func (s *server) handlerResetPassword() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		//Get the users email from query params
		email := r.URL.Query().Get("email")
		if email == "" {
			s.respond(w, r, nil, "must provide email", http.StatusBadRequest)
		}
	}
}

func (s *server) handlerPetsGetAll() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Get user ID from the authenticated context
		id := context.Get(r, "userID")
		idStr := fmt.Sprintf("%v", id)
		idInt, _ := strconv.Atoi(idStr)

		// Get pets from the database and respond
		pets, err := s.dbPetsGetAll(int64(idInt))
		if err != nil {
			s.respond(w, r, nil, err.Error(), http.StatusInternalServerError)
		}
		s.respond(w, r, pets, "", http.StatusOK)
	}
}

func (s *server) handlerPetsGetOne() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Get user ID from the authenticated context
		id := context.Get(r, "userID")
		idStr := fmt.Sprintf("%v", id)
		idInt, _ := strconv.Atoi(idStr)

		// Get URL params
		params := mux.Vars(r)
		id := params["id"]
		if id == "" {
			s.respond(w, r, nil, "must provide a pet id", http.StatusBadRequest)
		}
	}
}

func (s *server) handlerPetsCreate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}

func (s *server) handlerPetsUpdate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}

func (s *server) handlerPetsDelete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}
