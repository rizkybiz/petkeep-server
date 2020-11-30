package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
			s.logger.Error().Err(err).Msg("error encoding data")
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

//userIDFromRequest is a helper to extract UserID from the HTTP Request context
func userIDFromRequest(r *http.Request) (int64, error) {
	id := context.Get(r, "userID")
	idStr := fmt.Sprintf("%v", id)
	idInt, err := strconv.Atoi(idStr)
	if err != nil {
		return 0, err
	}
	return int64(idInt), nil
}

// handlerLogin godoc
// @Summary Login a user
// @Description Login a user
// @Accept json
// @Produce json
// @Param user body userRequest true "Login User"
// @Success 200 {object} token
// @Router /login [post]
func (s *server) handlerLogin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Create user and decode into it
		var usr user
		err := s.decode(w, r, &usr)
		if err != nil {
			s.logger.Error().Err(err).Msg("error decoding")
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
			s.logger.Error().Err(err).Msg("error retrieving user from database")
			s.respond(w, r, nil, "incorrect password", http.StatusUnauthorized)
			return
		}

		// Generate access token
		accTokenStr, err := generateToken(uint(id), time.Now().Add(time.Hour*24))
		if err != nil {
			s.logger.Error().Err(err).Msg("error generating access token")
			s.respond(w, r, nil, "internal server error", http.StatusInternalServerError)
			return
		}
		// Generate refresh token
		refrTokenStr, err := generateToken(uint(id), time.Now().Add(time.Hour*168))
		if err != nil {
			s.logger.Error().Err(err).Msg("error generating refresh token")
			s.respond(w, r, nil, "internal server error", http.StatusInternalServerError)
			return
		}
		// Respond with access and refresh tokens
		s.respond(w, r, token{AccessTkn: accTokenStr, RefreshTkn: refrTokenStr}, "", http.StatusOK)
	}
}

// handlerUsersCreate godoc
// @Summary Create a user
// @Description Create a user
// @Tags Users
// @Accept json
// @Produce json
// @Param user body userRequest true "Create User"
// @Success 200 {object} userResponse
// @Router /users [post]
func (s *server) handlerUsersCreate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ts := time.Now()
		var usr user
		err := s.decode(w, r, &usr)
		if err != nil {
			s.logger.Error().Err(err).Msg("error decoding JSON")
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
			s.logger.Error().Err(err).Msg("error validating email")
			s.respond(w, r, nil, "email format is invalid", http.StatusBadRequest)
			return
		}

		//Create hashed version of password
		hashedPass, err := bcrypt.GenerateFromPassword([]byte(usr.Password), bcrypt.DefaultCost)
		if err != nil {
			s.logger.Error().Err(err).Msg("error hashing password")
			s.respond(w, r, nil, "internal server error", http.StatusInternalServerError)
			return
		}

		// Fill the user struct to send to the DB
		usr.Password = string(hashedPass)
		usr.CreatedAt = ts
		usr.UpdatedAt = ts

		//Create the user in the DB
		_, err = s.dbUsersCreate(usr)
		if err != nil {
			s.logger.Error().Err(err).Msg("error creating user in database")
			s.respond(w, r, nil, err.Error(), http.StatusInternalServerError)
			return
		}

		// Return userResponse
		usrResp := userResponse{
			Email:     usr.Email,
			CreatedAt: ts,
			UpdatedAt: ts,
		}
		s.respond(w, r, usrResp, "", http.StatusCreated)
	}
}

// handlerUsersGetOne godoc
// @Summary Get a user
// @Description Get a user
// @Tags Users
// @Produce json
// @Success 200 {object} userResponse
// @Security ApiKeyAuth
// @Router /users [get]
func (s *server) handlerUsersGetOne() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Get user ID from the authenticated context
		id, err := userIDFromRequest(r)
		if err != nil {
			s.logger.Error().Err(err).Msg("error retrieving user ID from context")
			s.respond(w, r, nil, "error retrieving user", http.StatusUnauthorized)
		}

		// Get user from database and respond
		u, err := s.dbUsersGetOne(int64(id))
		if err != nil {
			s.logger.Error().Err(err).Msg("error retrieving user from database")
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

// handlerPetsGetAll godoc
// @Summary Get all pets
// @Description Get all pets
// @Tags Pets
// @Produce json
// @Success 200 {array} pet
// @Security ApiKeyAuth
// @Router /pets [get]
func (s *server) handlerPetsGetAll() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Get user ID from the authenticated context
		id, err := userIDFromRequest(r)
		if err != nil {
			s.logger.Error().Err(err).Msg("error retrieving user ID from context")
			s.respond(w, r, nil, "error retrieving pets", http.StatusUnauthorized)
		}

		// Get pets from the database and respond
		pets, err := s.dbPetsGetAll(int64(id))
		if err != nil {
			s.logger.Error().Err(err).Msg("error retrieving pets from database")
			s.respond(w, r, nil, err.Error(), http.StatusInternalServerError)
		}
		s.respond(w, r, pets, "", http.StatusOK)
	}
}

// handlerPetsGetOne godoc
// @Summary Get one pet
// @Description Get one pet
// @Tags Pets
// @Produce json
// @Param PetID path int true "Get Pet"
// @Success 200 {object} pet
// @Security ApiKeyAuth
// @Router /pets/{PetID} [get]
func (s *server) handlerPetsGetOne() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Get user ID from the authenticated context
		id, err := userIDFromRequest(r)
		if err != nil {
			s.logger.Error().Err(err).Msg("error retrieving user ID from context")
			s.respond(w, r, nil, "error retrieving pet", http.StatusUnauthorized)
		}

		// Get URL params
		params := mux.Vars(r)
		petIDStr := params["id"]
		if petIDStr == "" {
			s.respond(w, r, nil, "must provide a pet id", http.StatusBadRequest)
		}
		petIDInt, _ := strconv.Atoi(petIDStr)

		// Get pet from db
		pet, err := s.dbPetsGetOne(id, int64(petIDInt))
		if err != nil {
			s.logger.Error().Err(err).Msg("error retrieving pet from database")
			s.respond(w, r, nil, err.Error(), http.StatusInternalServerError)
		}
		// Respond with pet record
		s.respond(w, r, pet, "", http.StatusOK)
	}
}

// handlerPetsCreate godoc
// @Summary Create a pet
// @Description Create a pet
// @Tags Pets
// @Accept json
// @Produce json
// @Param pet body petRequest true "Create Pet"
// @Success 200 {object} pet
// @Security ApiKeyAuth
// @Router /pets [post]
func (s *server) handlerPetsCreate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ts := time.Now()

		// Get user ID from the authenticated context
		userID, err := userIDFromRequest(r)
		if err != nil {
			s.logger.Error().Err(err).Msg("error retrieving user ID from context")
			s.respond(w, r, nil, "error creating pet", http.StatusUnauthorized)
		}

		var pet pet

		// Get JSON body and decode into a pet
		err = s.decode(w, r, &pet)
		if err != nil {
			s.logger.Error().Err(err).Msg("error decoding JSON")
		}
		pet.CreatedAt = ts
		pet.UpdatedAt = ts

		// Create pet in the db
		id, err := s.dbPetsCreate(pet, userID)
		if err != nil {
			s.logger.Error().Err(err).Msg("error creating pet in databse")
			s.respond(w, r, nil, "error creating pet", http.StatusInternalServerError)
			return
		}

		// Set ID's and respond
		pet.UserID = uint(userID)
		pet.ID = uint(id)
		s.respond(w, r, pet, "", http.StatusCreated)
	}
}

// handlerPetsUpdate godoc
// @Summary Update a pet
// @Description Update a pet
// @Tags Pets
// @Accept json
// @Produce json
// @Param PetID path int true "Pet ID"
// @Param pet body pet true "Updated Pet"
// @Success 200 {object} pet
// @Security ApiKeyAuth
// @Router /pets/{PetID} [put]
func (s *server) handlerPetsUpdate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ts := time.Now()

		// Get user ID from the authenticated context
		id, err := userIDFromRequest(r)
		if err != nil {
			s.logger.Error().Err(err).Msg("error retrieving user ID from context")
			s.respond(w, r, nil, "error retrieving user", http.StatusUnauthorized)
		}

		// Get URL params
		params := mux.Vars(r)
		petIDStr := params["id"]
		if petIDStr == "" {
			s.respond(w, r, nil, "must provide a pet id", http.StatusBadRequest)
		}
		petIDInt, _ := strconv.Atoi(petIDStr)

		var pet pet

		//Get JSON body and decode into pet
		err = s.decode(w, r, pet)
		if err != nil {
			s.logger.Error().Err(err).Msg("error decoding JSON")
		}
		pet.UpdatedAt = ts
		pet.ID = uint(petIDInt)

		//Update pet in the db
		err = s.dbPetsUpdate(pet, id)
		if err != nil {
			s.logger.Error().Err(err).Msg("error updating pet in databse")
			s.respond(w, r, nil, "error updating pet", http.StatusInternalServerError)
			return
		}

		//Respond with the updated pet record
		s.respond(w, r, pet, "", http.StatusOK)
	}
}

// handlerPetsDelete godoc
// @Summary Delete a pet
// @Description Delete a pet
// @Tags Pets
// @Param PetID path int true "Deleted Pet"
// @Success 200 {object} emptyBody
// @Security ApiKeyAuth
// @Router /pets/{PetID} [delete]
func (s *server) handlerPetsDelete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Get user ID from the authenticated context
		id, err := userIDFromRequest(r)
		if err != nil {
			s.logger.Error().Err(err).Msg("error retrieving user ID from context")
			s.respond(w, r, nil, "error retrieving user", http.StatusUnauthorized)
		}

		// Get URL params
		params := mux.Vars(r)
		petIDStr := params["id"]
		if petIDStr == "" {
			s.respond(w, r, nil, "must provide a pet id", http.StatusBadRequest)
		}
		petIDInt, _ := strconv.Atoi(petIDStr)

		//Delete the pet from the db
		rows, err := s.dbPetsDelete(int64(petIDInt), id)
		if err != nil || rows == 0 {
			s.respond(w, r, nil, "could not delete pet", http.StatusBadRequest)
		}

		s.respond(w, r, nil, "", http.StatusOK)
	}
}
