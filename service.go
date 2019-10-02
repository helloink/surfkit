package surfkit

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/gorilla/mux"
	"github.com/helloink/surfkit/events"
)

// A Service defines the application running
type Service struct {

	// Name of the service.
	Name string

	// Current version in the form x.x.x{-abc}
	Version string

	// Pubsub subscription.
	// Available modes are surfkit.PushSubscription and surfkit.PullSubscription.
	Subscription Subscription

	// Defines the services (Pubsub) output.
	Output *Output

	// A Router can be used to attach URL handlers
	Router *mux.Router

	// Srv allows access to the underlying webserver.
	Srv *http.Server

	// SrvHandler allows to set the request handler. If set, make sure it
	// eventually wraps service.Router.
	SrvHandler http.Handler

	// A Publisher take care of sending events to Pubsub.
	// Use surfkit.PublishEvent for a convinient method to send events.
	Publisher *events.Publisher

	// Env contains configuration read from the environment and is automatically set
	Env *ServiceEnv
}

// Run executes the service's run loop.
//
// It will first do required setup, next run the passed function
// and eventually handle its teardown.
func Run(s *Service, fn func()) {
	var err error

	log.Printf("Booting %s v%s (surfkit %s)", s.Name, s.Version, version)

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

	// Setup the Publisher
	if s.Output != nil {

		s.Publisher = &events.Publisher{
			ProjectID: Env("PUBSUB_PROJECT_ID"),
			Topic:     s.Output.EventType,
		}

		err := s.Publisher.Setup()
		if err != nil {
			log.Fatal("Failed to setup Output Publisher: ", err)
		}
	}

	// Invoke main service func
	fn()

	// Signal handling so we can gracefully shutdown service
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Enable Pubsub Listening
	if s.Subscription != nil {
		err := s.Subscription.Listen(s)
		if err != nil {
			log.Fatal("Failed to listen on Pubsub: ", err)
		}
	}

	// Any service will eventually rest on a webserver. Any empty service,
	// meaning no pubsub or handler have been set, will only serve the /health endpoint.
	go func() {
		err := enableServer(s)
		if err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to boot webserver: ", err)
		}
	}()

	<-done
	log.Println("Initiating Teardown...")

	shutdownServer(s)
	s.Teardown()

	log.Println("Good bye.")
}

// Teardown is called so the service can do cleanup work before finally going down.
func (s *Service) Teardown() {

	// Stop Publishers
	if s.Publisher != nil {
		s.Publisher.Stop()
	}

	// Cleanup Subscriptions
	if s.Subscription != nil {
		err := s.Subscription.Teardown(s)
		if err != nil {
			log.Println("Failed to teardown subscription:", err)
		}
	}
}

func convertEventTypeToTopic(eventType string) string {
	return strings.Replace(eventType, ".", "-", -1)
}
