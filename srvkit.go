package srvkit

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
)

const version = "1.0.0"

// A Service defines the application running
type Service struct {

	// Name of the service.
	Name string

	// Current version in the form x.x.x{-abc}
	Version string

	// Pubsub subscription.
	// Available modes are srvkit.PushSubscription and srvkit.PullSubscription.
	Subscription Subscription

	// A Router can be used to attach URL handlers
	Router *mux.Router

	// Srv allows access to the underlying webserver.
	Srv *http.Server
}

// Env reads a variable from ENV or fails fatal
func Env(s string) string {
	val, ok := os.LookupEnv(s)
	if !ok {
		log.Fatalf("failed to read %s from env", s)
	}

	return val
}

// Run executes the service's run loop.
//
// It will first do required setup, next run the passed function
// and eventually handle its teardown.
func Run(s *Service, fn func()) {
	var err error

	log.Printf("Booting %s v%s (srvkit %s)", s.Name, s.Version, version)

	// Make sure all required information is available in the environment
	assertEnvironment(s)

	// Setup the router so the service can attach handlers
	setupServer(s)

	// Setup the Pubsub subscription
	if s.Subscription != nil {
		err = s.Subscription.Setup(s)
		if err != nil {
			log.Fatal("Failed to setup Pubsub: ", err)
		}
	}

	// Invoke main service func
	fn()

	// Any service will eventually rest on a webserver. Any empty service,
	// meaning no pubsub or handler have been set, will only serve the /health endpoint.
	err = enableServer(s)
	if err != nil {
		log.Fatal("Failed to boot webserver: ", err)
	}

	log.Println("Good bye.")
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
		Handler:      s.Router,
		Addr:         fmt.Sprintf(":%s", port),
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
	}

	log.Printf("Server enabled on port %s", port)
	return s.Srv.ListenAndServe()
}
