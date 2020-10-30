package api

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
)

var jwtSigningKey string

//Server is the API server for handling HTTP requests
type server struct {
	router *mux.Router
	db     *sql.DB
	logger zerolog.Logger
}

func newServer() *server {
	s := &server{}
	s.routes()
	return s
}

//ServeHTTP fulfills the http.Server interface
func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

//StartServer starts the API server listening on a specific port, connected to MYSQL
func StartServer(port, dbHost, dbPort, dbUser, dbPassword, dbDatabase, jwtKey string) error {

	//Setup the signing key
	jwtSigningKey = jwtKey

	// Connect to the MYSQL database
	db, err := connectDB(dbHost, dbPort, dbUser, dbPassword, dbDatabase)
	if err != nil {
		return err
	}

	// Run DB migrations
	err = migrateDB(db)
	if err != nil {
		return err
	}

	// Create the server
	srv := newServer()
	// Add DB to the server
	srv.db = db
	// Initialize the logger
	srv.logger = zerolog.New(os.Stdout).With().Timestamp().Logger()

	// Start the server in the background
	if port == "" {
		return errors.New("You must provide a port")
	}
	go func() {
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), srv))
	}()

	// Create channel of os.Signal and wait for a signal interrupt,
	// if signal interrupt, fall through and stop server cleanly.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop
	stopServer(srv)

	return nil
}

func stopServer(s *server) {

	// Disconnect from the database
	err := disconnectDB(s.db)
	if err != nil {
		log.Fatalf("Could not disconnect from the database: %s", err)
	}
}
