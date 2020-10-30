package cmd

import (
	"errors"
	"log"

	"github.com/rizkybiz/petkeep-server/api"
	"github.com/spf13/cobra"
)

var (
	port          string
	dbHost        string
	dbPort        string
	dbUser        string
	dbPassword    string
	dbDatabase    string
	jwtSigningKey string
	logLevel      string
)

func init() {
	runCmd.PersistentFlags().StringVarP(&port, "port", "p", "8080", "This flag sets the port of the petkeep API server")
	runCmd.PersistentFlags().StringVarP(&dbHost, "database-host", "H", "", "The hostname or IP of the MYSQL database Server")
	runCmd.PersistentFlags().StringVarP(&dbPort, "database-port", "P", "3306", "The port of the MYSQL Database server")
	runCmd.PersistentFlags().StringVarP(&dbUser, "database-user", "u", "", "The username for accessing the MYSQL database server")
	runCmd.PersistentFlags().StringVarP(&dbPassword, "database-password", "W", "", "The password for accessing the MYSQL database server")
	runCmd.PersistentFlags().StringVarP(&dbDatabase, "database-name", "d", "petkeep", "The name of the MYSQL database")
	runCmd.PersistentFlags().StringVarP(&jwtSigningKey, "jwt-signing-key", "j", "", "The key to sign JWT's (KEEP SECRET!)")
	runCmd.PersistentFlags().StringVarP(&logLevel, "log-level", "l", "INFO", "This flag sets the log level of the server. INFO, DEBUG, etc.")
	rootCmd.AddCommand(runCmd)
}

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "This starts the petkeep API Server",
	RunE: func(cmd *cobra.Command, args []string) error {
		if jwtSigningKey == "" {
			err := errors.New("Must provide the JWT Signing Key to start the server")
			log.Fatal(err)
			return err
		}
		err := api.StartServer(port, dbHost, dbPort, dbUser, dbPassword, dbDatabase, jwtSigningKey)
		if err != nil {
			log.Fatalf("Could not start the API server: %s", err)
			return err
		}
		return nil
	},
}
