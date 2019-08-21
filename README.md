# surfkit

A small framework to surf the world! and also to build golang service super quick.


## Getting started

```go
func main() {

	s := surfkit.Service{
		Name:    "gustav-library",
		Version: "1.0.0",

		Subscription: &surfkit.PushSubscription{
			Topic:      surfkit.Env("PUBSUB_TOPIC"),
			HandleFunc: handleMessages,
		},
	}

	surfkit.Run(&s, func() {})
}

func handleMessages(e *surfkit.Event) bool {

    // Return `true` if you want the underlying pubsub message to be acknowledged (ack)
    // and `false` for nack.
	return true
}
```