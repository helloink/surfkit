package surfkit

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/helloink/surfkit/events"
)

// A Service defines the application running
type Service struct {

	// Name of the service.
	Name string

	// Current version in the form x.x.x{-abc}
	Version string

	// Pubsub Subscription. Use `Subscriptions` for multiple inputs.
	// Available modes are surfkit.PushSubscription and surfkit.PullSubscription.
	Subscription Subscription

	Subscriptions []Subscription

	// Defines the services (Pubsub) output.
	Output *Output

	Outputs []*Output

	// A Router can be used to attach URL handlers
	Router *mux.Router

	// Srv allows access to the underlying webserver.
	Srv *http.Server

	// SrvTimeout sets the read & write timeouts of the underlying webserver
	SrvTimeout time.Duration

	// SrvHandler allows to set the request handler. If set, make sure it
	// eventually wraps service.Router.
	SrvHandler http.Handler

	// A Publisher take care of sending events to Pubsub.
	// Use surfkit.PublishEvent for a convinient method to send events.
	Publisher *events.Publisher

	Publishers map[string]*events.Publisher

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
	for _, sub := range pubsubSubscriptions(s) {

		// Subscription Naming is an important thing...
		if sub.GetName() == "" {
			log.Fatal("Every Pubsub Subscription must have a name set.")
		}

		err = sub.Setup(s)
		if err != nil {
			log.Fatal("Failed to setup Pubsub: ", err)
		}
	}

	// Setup publishers
	s.Publishers = make(map[string]*events.Publisher)
	if s.Output != nil {
		eventType := s.Output.EventType
		publisher := setupPublisher(eventType)
		s.Publisher = publisher
		s.Publishers[eventType] = publisher
	}
	if s.Outputs != nil {
		for _, o := range s.Outputs {
			s.Publishers[o.EventType] = setupPublisher(o.EventType)
		}
	}

	// Invoke main service func
	fn()

	// Signal handling so we can gracefully shutdown service
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Enable Pubsub Listening
	for _, sub := range pubsubSubscriptions(s) {
		go func(s *Service, sub Subscription) {
			err := sub.Listen(s)
			if err != nil {
				log.Fatal("Failed to listen on Pubsub: ", err)
			}
		}(s, sub)
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
	for _, p := range s.Publishers {
		p.Stop()
	}

	// Cleanup Subscriptions
	for _, sub := range pubsubSubscriptions(s) {
		err := sub.Teardown(s)
		if err != nil {
			log.Println("Failed to teardown subscription:", err)
		}
	}

}

func convertEventTypeToTopic(eventType string) string {
	return strings.Replace(eventType, ".", "-", -1)
}

// pubsubSubscriptions as configured via the Surfkit interface.
func pubsubSubscriptions(s *Service) []Subscription {
	if s.Subscription != nil {
		return append(s.Subscriptions, s.Subscription)
	}

	return s.Subscriptions
}

func setupPublisher(eventType string) *events.Publisher {
	publisher := &events.Publisher{
		ProjectID: Env("PUBSUB_PROJECT_ID"),
		Topic:     eventType,
	}

	err := publisher.Setup()
	if err != nil {
		log.Fatal("Failed to setup Publisher: ", err)
	}

	return publisher
}
