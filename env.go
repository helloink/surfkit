package surfkit

import (
	"log"
	"os"
)

// ServiceEnv contains configuration read from the environment.
type ServiceEnv struct {

	// The Port the service's http handler is listening on.
	Port string

	// The ID of the Project this service is running on.
	ProjectID string
}

// Env reads a variable from ENV or fails fatal
func Env(s string) string {
	val, ok := os.LookupEnv(s)
	if !ok {
		log.Fatalf("failed to read %s from env", s)
	}

	return val
}

// Read vital configuration from the environment and set fallbacks or fail
func assertEnvironment(s *Service) {

	s.Env = &ServiceEnv{}

	port, ok := os.LookupEnv("PORT")
	if !ok {
		port = "3000"
	}

	s.Env.Port = port

	// If a Pubsub Subscription is set, the project id must be set in ENV.
	if s.Subscription != nil {
		projectID, ok := os.LookupEnv("PUBSUB_PROJECT_ID")
		if !ok {
			log.Fatal("in order to use pubsub make sure PUBSUB_PROJECT_ID is available in ENV")
		}

		s.Env.ProjectID = projectID
	}
}
