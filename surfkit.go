package surfkit

import (
	"fmt"

	"github.com/helloink/surfkit/events"
)

const version = "1.10.0"

// Output defines the single channel on which the service produces output, given it is a Pubsub output.
// Eventually this should also cover HTTP Endpoints.
type Output struct {
	EventType string
}

// PublishEvent sends the provided payload, wrapped in a CloudEvent, to all subscribers of the topic.
// It uses the topic as defined by service.Output
func PublishEvent(s *Service, payload interface{}) error {
	return publish(s, s.Publisher, s.Output.EventType, payload)
}

// PublishEvent sends the provided payload, wrapped in a CloudEvent, to all subscribers of the given
// topic. The topic must be either the topic defined by service.Output or one of the topics defined
// by service.Outputs.
func PublishEventTo(s *Service, eventType string, payload interface{}) error {
	publisher, ok := s.Publishers[eventType]
	if !ok {
		return fmt.Errorf("unknown publisher: %s", eventType)
	}
	return publish(s, publisher, eventType, payload)
}

func publish(s *Service, p *events.Publisher, eventType string, payload interface{}) error {
	eventSource := fmt.Sprintf("%s.%s", s.Name, s.Version)
	ce := events.NewCloudEvent(eventSource, eventType, payload)

	err := p.Send(ce)
	if err != nil {
		return fmt.Errorf("failed to send cloud event (%v)", err)
	}

	return nil
}
