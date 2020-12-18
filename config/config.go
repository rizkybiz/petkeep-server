package config

import "github.com/namsral/flag"

//Config is a struct for configuring the API server
type Config struct {
	APIPort       string
	DBHost        string
	DBPort        string
	DBUser        string
	DBInsecure    bool
	CertPath      string
	DBName        string
	JWTSigningKey string
	LogLevel      string
	StatsdHost    string
	StatsdPort    string
	ServerHost    string
}

//Generate returns a new config from ENV, file, or flags
func Generate() Config {
	cfg := Config{}
	flag.StringVar(&cfg.APIPort, "api-port", "8080", "port of the petkeep API server")
	flag.StringVar(&cfg.DBHost, "api-database-host", "", "hostname or IP address of the cockroachdb server")
	flag.StringVar(&cfg.DBPort, "api-database-port", "3306", "port of the cockroachdb server")
	flag.StringVar(&cfg.DBUser, "api-database-user", "", "username for accessing the MYSQL database server")
	flag.StringVar(&cfg.CertPath, "api-cert-path", "certs/", "path where CockroachDB certs are stored")
	flag.StringVar(&cfg.DBName, "api-database-name", "petkeep", "name of the cockroachdb database")
	flag.StringVar(&cfg.JWTSigningKey, "api-jwt-signing-key", "", "key to sign JWT's (KEEP SECRET!)")
	flag.StringVar(&cfg.LogLevel, "api-log-level", "INFO", "log level of the server. INFO, DEBUG, etc.")
	flag.BoolVar(&cfg.DBInsecure, "api-insecure-database-connection", false, "enable insecure communication between api and cockroachdb")
	flag.StringVar(&cfg.StatsdHost, "api-statsd-host", "", "hostname or IP address for statsd server")
	flag.StringVar(&cfg.StatsdPort, "api-statsd-port", "8125", "port for statsd server")
	flag.StringVar(&cfg.ServerHost, "server-host", "localhost", "hostname to access the server")
	flag.Parse()
	return cfg
}
