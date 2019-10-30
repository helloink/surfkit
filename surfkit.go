package surfkit

import (
	"fmt"

	"github.com/helloink/surfkit/events"
)

const version = "1.3.0-d1"

// Output defines the single channel on which the service produces output, given it is a Pubsub output.
// Eventuall this should also cover HTTP Endpoints.
type Output struct {
	EventType string
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
