package events

import (
	"context"
	"encoding/json"
	"fmt"

	"cloud.google.com/go/pubsub"
)

// A Publisher is used to send event messages to a specific topic
type Publisher struct {
	ProjectID string
	Topic     string

	client *pubsub.Client
	ctx    context.Context
	topic  *pubsub.Topic
}

// NewPublisher provides an initialised Publisher
func NewPublisher(projectID string, topic string) *Publisher {
	return &Publisher{
		ProjectID: projectID,
		Topic:     topic,
	}
}

// Setup the Publisher's internals. Required before `Send`
func (p *Publisher) Setup() error {
	p.ctx = context.Background()

	client, err := pubsub.NewClient(p.ctx, p.ProjectID)
	if err != nil {
		return fmt.Errorf("failed to setup pubsub client (%v)", err)
	}

	topic := client.Topic(p.Topic)
	ok, err := topic.Exists(p.ctx)
	if err != nil {
		return fmt.Errorf("failed to verify topic (%v)", err)
	}

	if !ok {
		topic, err = client.CreateTopic(p.ctx, p.Topic)
		if err != nil {
			return fmt.Errorf("failed to create topic (%v)", err)
		}
	}

	p.client = client
	p.topic = topic

	return nil
}

// Stop makes sure all messages are delivered before returning.
// Use it before existing the programm.
func (p *Publisher) Stop() {
	if p.topic != nil {
		p.topic.Stop()
	}
}

// Send a CloudEvent messages to Pubsub
func (p *Publisher) Send(e CloudEvent) error {

	pb, err := json.Marshal(e)
	if err != nil {
		return err
	}

	p.topic.Publish(p.ctx, &pubsub.Message{Data: pb})

	return nil
}
