package surfkit

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/helloink/surfkit/events"
)

// PubsubPushMessageEnvelope as received via an http endpoint from a pubsub server
type PubsubPushMessageEnvelope struct {
	Subscription string            `json:"subscription"`
	Message      PubsubPushMessage `json:"message"`
}

// PubsubPushMessage as contained in the payload of PubsubPushMessageEnvelope
type PubsubPushMessage struct {
	MessageID  string            `json:"messageId"`
	Attributes map[string]string `json:"attributes"`

	// Data holds the pubsub message payload encoded as base64
	Data string `json:"data"`
}

// DecodeData returns a base64 decoded version of the Data field
func (m *PubsubPushMessage) DecodeData() ([]byte, error) {
	return base64.StdEncoding.DecodeString(m.Data)
}

// A Subscription is a means to receive messages from a specific pubsub channel
type Subscription interface {

	// Setup is called during Service initialisation and shall be used
	// by the Subscription to create subscriptions and other prerequisites.
	Setup(s *Service) error

	// Listen allows a Subscription to continously receive new messages.
	// If not required by the implementation, just noop it.
	Listen(s *Service) error

	// Teardown is called during Service shutdown and shall be used to clean up.
	Teardown(s *Service) error

	// GetName returns the name of the Subscription.
	GetName() string
}

// A PushSubscription uses an HTTP endpoint and gets
// new messages pushed from the Pubsub server.
//
// Learn more about this here
// https://cloud.google.com/pubsub/docs/subscriber#push-subscription
//
type PushSubscription struct {

	// Name this Subscription.
	//
	// Naming a subscription is an import aspect and therefor shall be done
	// purposefully and well thought through. Using the Service's name is a safe bet,
	// if the service has only one input.
	//
	// Remember: Subscription naming is the foundation for
	// Pubsub based service loadbalancing.
	Name string

	// The Topic this subscription is attached to
	Topic string

	// A func that will be called as soon as a new message arrives on the attached `Topic`.
	HandleFunc func(s *Service, e *events.CloudEvent) bool

	// See https://godoc.org/cloud.google.com/go/pubsub#ReceiveSettings
	ReceiveSettings *pubsub.ReceiveSettings

	// How long Pub/Sub waits for the subscriber to acknowledge receipt before resending the message
	AckDeadline time.Duration

	// Experimential. Delete the Subscription on shutdown of the service.
	DeleteOnShutdown bool

	// Experimental. Delete the subscription after time.Duration (min 1day) of subscriber inactivity
	ExpirationPolicy time.Duration

	service *Service
}

// Setup receive routes and the subscription
func (p *PushSubscription) Setup(s *Service) error {
	p.service = s

	host, ok := os.LookupEnv("HOST")
	if ok {

		// This is a special mechanism built to make it easier to deploy Surfkit Services on Cloud Run.
		// When a service is freshly launched, its own URL is still unknown - Google assigns it after
		// the first successful setup. But, this URL is needed to subscribe to a Pubsub topic so that the
		// Pubsub server knows which URL to send messages to.
		//
		// Skipping the Subscription setup allows to have the service being deployed once, so its URL can be
		// retrieved and correctly set as the HOST env with the next deploy. Only when the URL is correct,
		// a Subscription is created.
		if strings.HasPrefix(host, "http") == false {
			log.Println("WARN: HOST not valid. Skipping Pubsub Push Activation")
			return nil
		}

	} else {
		host = fmt.Sprintf("http://%s:%s", s.Name, s.Env.Port)
	}

	path := fmt.Sprintf("/sk/v1/messages/%s", p.Name)
	endpoint := fmt.Sprintf("%s%s", host, path)

	s.Router.HandleFunc(path, p.incomingPubsubMessages).Methods("POST")

	ctx := context.Background()

	client, err := pubsub.NewClient(ctx, s.Env.ProjectID)
	if err != nil {
		return fmt.Errorf("failed to setup pubsub (%v)", err)
	}

	// Setup and configure the subscription object
	sub := client.Subscription(p.Name)

	if p.ReceiveSettings != nil {
		sub.ReceiveSettings = *p.ReceiveSettings
	}

	// Check if the subscription exists already
	ok, err = sub.Exists(ctx)
	if err != nil {
		return fmt.Errorf("failed to check subscription %s (%v)", p.Name, err)
	}

	// If it doesn't exists, well...
	if !ok {

		ackTime := 10 * time.Second
		if p.AckDeadline != 0 {
			ackTime = p.AckDeadline
		}

		cfg := pubsub.SubscriptionConfig{
			Topic:       client.Topic(p.Topic),
			AckDeadline: ackTime,

			PushConfig: pubsub.PushConfig{
				Endpoint: endpoint,
			},
		}

		// Experimental.
		if p.ExpirationPolicy != 0 {
			cfg.ExpirationPolicy = p.ExpirationPolicy
		}

		_, err := client.CreateSubscription(ctx, p.Name, cfg)

		if err != nil {
			return fmt.Errorf("failed to create subscription %s on %s (%v)", p.Name, p.Topic, err)
		}
	}

	log.Printf("Pubsub: Subscription (%s) endpoint to %s mounted at %s", p.Name, p.Topic, endpoint)
	return nil
}

// Listen .. noop
func (p *PushSubscription) Listen(s *Service) error {
	// Noop
	return nil
}

// Teardown the subscription.
func (p *PushSubscription) Teardown(s *Service) error {
	if p.DeleteOnShutdown == true {
		return deleteSubscription(s, p.Name)
	}

	return nil
}

// GetName of this Subscription
func (p *PushSubscription) GetName() string {
	return p.Name
}

func (p *PushSubscription) incomingPubsubMessages(w http.ResponseWriter, r *http.Request) {
	var err error

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		p.respondWithError(w, "Failed to read body", err)
		return
	}

	var ev PubsubPushMessageEnvelope
	err = json.Unmarshal(body, &ev)
	if err != nil {
		p.respondWithError(w, "Failed to decode json body", err)
		return
	}

	data, err := ev.Message.DecodeData()
	if err != nil {
		p.respondWithError(w, "Failed to decode message data", err)
		return
	}

	var e *events.CloudEvent
	err = json.Unmarshal(data, &e)
	if err != nil {
		p.respondWithError(w, "Failed to unmarshal message data", err)
		return
	}

	ack := p.HandleFunc(p.service, e)
	if ack {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusNotAcceptable)
	}

}

func (p *PushSubscription) respondWithError(w http.ResponseWriter, m string, err error) {
	log.Printf("%s (%v)", m, err)
	w.WriteHeader(http.StatusNotAcceptable)
}

// A PullSubscription continiously pulls messages from a pubsub server.
// Learn more here https://cloud.google.com/pubsub/docs/subscriber#pull-subscription
type PullSubscription struct {

	// The Topic this subscription is attached to
	Topic string

	// A func that will be called as soon as a new message arrives on the attached `Topic`.
	HandleFunc func(s *Service, e *events.CloudEvent) bool

	// The name of this Subscription. This is by default the name of the Service and you should
	// probably keep it this way as you'll otherwise break the built-in load balancing.
	//
	// ¯\_(ツ)_/¯ ? Go ahead.
	Name string

	// How long Pub/Sub waits for the subscriber to acknowledge receipt before resending the message
	AckDeadline time.Duration

	// Experimential. Delete the Subscription on shutdown of the service.
	DeleteOnShutdown bool

	// Experimental. Delete the subscription after time.Duration (min 1day) of subscriber inactivity
	ExpirationPolicy time.Duration

	service *Service
}

// Setup Subscription
func (p *PullSubscription) Setup(s *Service) error {
	p.service = s

	return nil
}

// Listen for new messages on Pubsub
func (p *PullSubscription) Listen(s *Service) error {

	ctx := context.Background()

	client, err := pubsub.NewClient(ctx, s.Env.ProjectID)
	if err != nil {
		return fmt.Errorf("failed to setup pubsub (%v)", err)
	}

	// Check if the subscription exists already
	sub := client.Subscription(p.Name)
	ok, err := sub.Exists(ctx)
	if err != nil {
		return fmt.Errorf("failed to check subscription %s (%v)", p.Name, err)
	}

	// If it doesn't exists, well...
	if !ok {

		ackTime := 10 * time.Second
		if p.AckDeadline != 0 {
			ackTime = p.AckDeadline
		}

		cfg := pubsub.SubscriptionConfig{
			Topic:       client.Topic(p.Topic),
			AckDeadline: ackTime,
		}

		// Experimental.
		if p.ExpirationPolicy != 0 {
			cfg.ExpirationPolicy = p.ExpirationPolicy
		}

		sub, err = client.CreateSubscription(ctx, p.Name, cfg)

		if err != nil {
			return fmt.Errorf("failed to create subscription %s (%v)", p.Name, err)
		}
	}

	err = sub.Receive(ctx, func(ctx context.Context, m *pubsub.Message) {
		var e *events.CloudEvent
		err = json.Unmarshal(m.Data, &e)
		if err != nil {
			log.Printf("Failed to unmarshal pubsub message (%v)", err)
			m.Nack()
			return
		}

		if p.HandleFunc(p.service, e) {
			m.Ack()
		} else {
			m.Nack()
		}
	})

	if err != nil {
		return fmt.Errorf("failed to listen for new messages (%v)", err)
	}

	log.Printf("Pubsub: Subscription (%s) listening to %s", p.Name, p.Topic)
	return nil
}

// Teardown the subscription.
func (p *PullSubscription) Teardown(s *Service) error {
	if p.DeleteOnShutdown == true {
		return deleteSubscription(s, p.Name)
	}

	return nil
}

// GetName of this Subscription
func (p *PullSubscription) GetName() string {
	return p.Name
}

func deleteSubscription(s *Service, name string) error {
	ctx := context.Background()

	client, err := pubsub.NewClient(ctx, s.Env.ProjectID)
	if err != nil {
		return fmt.Errorf("failed to setup pubsub (%v)", err)
	}

	sub := client.Subscription(name)
	return sub.Delete(ctx)
}
