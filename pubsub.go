package srvkit

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
)

// An Event as delivered via Pubsub message
type Event struct {

	// Data is the event's payload. Use the #DataTo method
	// to turn it into a struct.
	Data []byte

	// Raw allows acces to the underlying message container. Either
	// an HTTP body or a pubsub.Message. Have fun
	Raw interface{}
}

// DataTo turns the Data field into the passed Type
func (e *Event) DataTo(obj interface{}) error {
	return json.Unmarshal(e.Data, obj)
}

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
type PushSubscription struct {
	Topic      string
	HandleFunc func(e *Event) bool
}

// Setup receive routes and the subscription
func (p *PushSubscription) Setup(s *Service) error {
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

	// Check if the subscription exists already
	sub := client.Subscription(s.Name)
	ok, err = sub.Exists(ctx)
	if err != nil {
		return fmt.Errorf("failed to check subscription (%v)", err)
	}

	// If it doesn't exists, well...
	if !ok {
		_, err := client.CreateSubscription(ctx, s.Name, pubsub.SubscriptionConfig{
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

	log.Printf("Pubsub: Endpoint mounted at %s", endpoint)

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

	e := &Event{
		Data: data,
		Raw:  ev,
	}

	ack := p.HandleFunc(e)
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