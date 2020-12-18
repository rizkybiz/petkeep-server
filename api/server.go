package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"

	// GorillaHandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/rizkybiz/petkeep-server/config"
	"github.com/rs/cors"
	"github.com/rs/zerolog"
	"gopkg.in/alexcesaro/statsd.v2"
)

var jwtSigningKey string

//Server is the API server for handling HTTP requests
type server struct {
	router     *mux.Router
	db         *sql.DB
	logger     zerolog.Logger
	statsd     *statsd.Client
	listenPort string
	serverHost string
}

func newServer(serverHost, listenPort string) *server {
	s := &server{serverHost: serverHost, listenPort: listenPort}
	s.routes()
	return s
}

//ServeHTTP fulfills the http.Server interface
func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

//StartServer starts the API server listening on a specific port, connected to cockroachdb
func StartServer(cfg config.Config) error {

	//Setup the signing key
	if cfg.JWTSigningKey == "" {
		return errors.New("must provide jwt signing key")
	}
	jwtSigningKey = cfg.JWTSigningKey

	// Create the server
	srv := newServer(cfg.ServerHost, cfg.APIPort)

	// Initialize the logger
	srv.logger = zerolog.New(os.Stdout).With().Timestamp().Logger()

	// Connect to the cockroach database
	err := srv.connectDB(cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.CertPath, cfg.DBName, cfg.DBInsecure)
	if err != nil {
		return err
	}
	defer srv.db.Close()

	if cfg.StatsdHost != "" {
		err := srv.newStatsdClient(cfg.StatsdHost, cfg.StatsdPort)
		if err != nil {
			return err
		}
		defer srv.statsd.Close()
	}

	// Set up CORS middleware
	handler := cors.Default().Handler(srv)

	// Start the server in the background
	go func() {
		srv.logger.Fatal().Err(http.ListenAndServe(fmt.Sprintf(":%s", cfg.APIPort), handler)).Msg("error handling tcp")
	}()

	// Create channel of os.Signal and wait for a signal interrupt,
	// if signal interrupt, fall through and stop server cleanly.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop

	return nil
}

func (s *server) newStatsdClient(addr, port string) error {
	c, err := statsd.New(
		statsd.Address(fmt.Sprintf("%s:%s", addr, port)),
		statsd.ErrorHandler(func(err error) {
			s.logger.Err(err).Msg("error sending statsd")
		}))
	if err != nil {
		s.logger.Err(err).Msg("error creating statsd client")
		return err
	}
	s.logger.Debug().Str("statsd_connection_string", fmt.Sprintf("%s:%s", addr, port)).Send()
	s.statsd = c
	return nil
}
