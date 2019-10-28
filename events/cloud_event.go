package events

import (
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/tidwall/gjson"
)

// Guidelines to construct a proper CloudEvent model: https://github.com/cloudevents/spec/blob/v0.3/spec.md#required-attributes

// Houserules:

// Required:
// ID => YearMonthDayHourMinuteSecondMilliSecond_RandomString	e.g. Y2019M08D23H19M20S14MS30_HerEcOmEsARaND0mstR1nG
// source => service.version/UUID/SessionID/...    				e.g. alfred.1.0.0.a67d76776g7d67a
// specversion => 0.3
// type => controller.eventtype.comoponent.action				e.g. homepage.useraction.donecta.tapped, storiesservice.api.getstories.success, etc..

const specVersion = "0.3"

// CloudEvent represents an Event as described in https://github.com/cloudevents/spec/blob/v0.3/spec.md#event
type CloudEvent struct {
	ID          string      `json:"id"`
	Source      string      `json:"source"`
	Specversion string      `json:"specversion"`
	Type        string      `json:"type"`
	Time        time.Time   `json:"time"`
	Data        interface{} `json:"data"`
}

// NewCloudEvent returns a new and initialised CloudEvent
func NewCloudEvent(source string, eventType string, payload interface{}) CloudEvent {
	id, err := uuid.NewRandom()
	if err != nil {
		log.Fatalln("Failed to generate UUID:", err)
	}

	return CloudEvent{
		ID:          id.String(),
		Source:      source,
		Specversion: specVersion,
		Type:        eventType,
		Time:        time.Now().UTC(),
		Data:        payload,
	}
}

// DataTo turns the Data field into the passed Type
func (e *CloudEvent) DataTo(obj interface{}) error {
	pb, err := json.Marshal(e.Data)
	if err != nil {
		return err
	}

	return json.Unmarshal(pb, obj)
}

// GetData at a specific path.
// Check https://github.com/tidwall/gjson for syntax
func (e *CloudEvent) GetData(path string) gjson.Result {
	b, err := json.Marshal(e.Data)
	if err != nil {
		log.Fatalln("Failed to Marshal interface:", err)
		return gjson.Result{}
	}

	return gjson.GetBytes(b, path)
}
