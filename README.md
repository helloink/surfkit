# surfkit

A small framework to surf the world! and also to build golang service super quick.


## Getting started

```go

import "github.com/helloink/surfkit"

func main() {

	s := surfkit.Service{
		Name:    "my-service",
		Version: "1.0.0",

		Subscription: &surfkit.PushSubscription{
			Topic:      surfkit.Env("PUBSUB_TOPIC"),
			HandleFunc: handleMessages,
		},
	}

	surfkit.Run(&s, func(
		// Noop
	) {})
}

// Return `true` if you want the underlying pubsub message to be acknowledged (ack)
// and `false` for nack.
func handleMessages(s *surfkit.Service, e *events.CloudEvent) bool {
	return true
}
```
