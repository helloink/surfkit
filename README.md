# srvkit

A small framework to build golang service super quick.

## Getting started

```go
func main() {

	s := srvkit.Service{
		Name:    "gustav-library",
		Version: "1.0.0",

		Subscription: &srvkit.PushSubscription{
			Topic:      srvkit.Env("PUBSUB_TOPIC"),
			HandleFunc: handleMessages,
		},
	}

	srvkit.Run(&s, func() {})
}

func handleMessages(e *srvkit.Event) bool {

    // Return `true` if you want the underlying pubsub message to be acknowledged (ack)
    // and `false` for nack.
	return true
}
```