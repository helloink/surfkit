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
	// by the Subscription to create subscriptions and other prerequisites
	Setup(s *Service) error

	// Listen allows a Subscription to continously receive new messages.
	// If not required by the implementation, just noop it.
	Listen(s *Service) error
}

// A PushSubscription uses an HTTP endpoint and gets
// new messages pushed from the Pubsub server.
//
// Learn more about this here
// https://cloud.google.com/pubsub/docs/subscriber#push-subscription
//
type PushSubscription struct {

	// The Topic this subscription is attached to
	Topic string

	// A func that will be called as soon as a new message arrives on the attached `Topic`.
	HandleFunc func(s *Service, e *events.CloudEvent) bool

	// The name of this Subscription. This is by default the name of the Service and you should
	// probably keep it this way as you'll otherwise break the built in load balancing.
	//
	// ¯\_(ツ)_/¯ ? Go ahead.
	Name string

	service *Service
}

// Setup receive routes and the subscription
func (p *PushSubscription) Setup(s *Service) error {
	p.service = s

	host, ok := os.LookupEnv("HOST")
	if !ok {
		port, ok := os.LookupEnv("PORT")
		if !ok {
			port = "3000"
		}

		host = fmt.Sprintf("http://%s:%s", s.Name, port)
	}

	path := "/sk/v1/messages"
	endpoint := fmt.Sprintf("%s%s", host, path)

	s.Router.HandleFunc(path, p.incomingPubsubMessages).Methods("POST")

	// No assert required. See comment on PORT reading.
	projectID, _ := os.LookupEnv("PUBSUB_PROJECT_ID")

	ctx := context.Background()
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return fmt.Errorf("failed to setup pubsub (%v)", err)
	}

	subName := p.generateName()

	// Check if the subscription exists already
	sub := client.Subscription(subName)
	ok, err = sub.Exists(ctx)
	if err != nil {
		return fmt.Errorf("failed to check subscription (%v)", err)
	}

	// If it doesn't exists, well...
	if !ok {
		_, err := client.CreateSubscription(ctx, subName, pubsub.SubscriptionConfig{
			Topic:       client.Topic(p.Topic),
			AckDeadline: 10 * time.Second,

			PushConfig: pubsub.PushConfig{
				Endpoint: endpoint,
			},
		})

		if err != nil {
			return fmt.Errorf("failed to create subscription (%v)", err)
		}
	}

	log.Printf("Pubsub: Subscription (%s) endpoint mounted at %s", subName, endpoint)

	return nil
}

// Listen .. noop
func (p *PushSubscription) Listen(s *Service) error {
	// Noop
	return nil
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

// Generate a name for the subscription.
func (p *PushSubscription) generateName() string {
	if p.Name == "" {
		return p.service.Name
	}

	return p.Name
}
