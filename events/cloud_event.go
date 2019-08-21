package events

import (
	"log"
	"time"

	"github.com/google/uuid"
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
func NewCloudEvent(eventTyep string, payload interface{}) CloudEvent {
	id, err := uuid.NewRandom()
	if err != nil {
		log.Fatalln("Failed to generate UUID:", err)
	}

	return CloudEvent{
		ID:          id.String(),
		Specversion: specVersion,
		Time:        time.Now().UTC(),
		Data:        payload,
	}
}
