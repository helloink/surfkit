package surfkit

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/helloink/surfkit/events"
)

const version = "1.2.1-d3"

// Output defines the single channel on which the service produces output, given it is a Pubsub output.
// Eventuall this should also cover HTTP Endpoints.
type Output struct {
	EventType string
}

// Env reads a variable from ENV or fails fatal
func Env(s string) string {
	val, ok := os.LookupEnv(s)
	if !ok {
		log.Fatalf("failed to read %s from env", s)
	}

	return val
}

// PublishEvent sends the provided payload, wrapped in a CloudEvent, to all subscribers of the topic.
// It uses the topic as defined by service.Output
func PublishEvent(s *Service, payload interface{}) error {
	eventSource := fmt.Sprintf("%s.%s", s.Name, s.Version)
	ce := events.NewCloudEvent(eventSource, s.Output.EventType, payload)
	err := s.Publisher.Send(ce)
	if err != nil {
		return fmt.Errorf("failed to send cloud event (%v)", err)
	}

	return nil
}

func assertEnvironment(s *Service) {

	// If a Pubsub Subscription is set, the project id must be set in ENV.
	if s.Subscription != nil {
		_, ok := os.LookupEnv("PUBSUB_PROJECT_ID")
		if !ok {
			log.Fatal("in order to use pubsub make sure PUBSUB_PROJECT_ID is available in ENV")
		}
	}
}

func setupServer(s *Service) {
	s.Router = mux.NewRouter()
	s.Router.HandleFunc("/", healthEndpoint).Methods("GET")

	s.SrvHandler = s.Router
}

func healthEndpoint(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func enableServer(s *Service) error {
	port, ok := os.LookupEnv("PORT")
	if !ok {
		port = "3000"
	}

	s.Srv = &http.Server{
		Handler:      s.SrvHandler,
		Addr:         fmt.Sprintf(":%s", port),
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
	}

	log.Printf("Server enabled on port %s", port)
	return s.Srv.ListenAndServe()
}
