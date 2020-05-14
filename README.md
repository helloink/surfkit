# surfkit

A small framework to surf the world! and also to build golang services super
quick. It is opinionated and specifically built to support GCP pubsub based
eventing environments and mixed pubsub+http architectures.

A surfkit based service as zero or more incoming subscriptions and one optional
outgoing channel that it can publish to. Additionally a service can hook into
surfkit's http facility to gain a synchronous REST like interface.

## Quick start

In its most simple form a surfkit enabled service looks like this:

```go
package main

import "github.com/helloink/surfkit"

func main() { 

  s := surfkit.Service{ 
      Name:    "jupiter",
      Version: "0.1.0"
  }

  surfkit.Run(&s, func() {})
}
```

This will setup a simple service that does absolutely nothing when looking at
it. Under the hood, it has started a webserver that serves a health endpoint at
`/` and registered for signal handling.

### Runloop

Surfkit operates the runloop for you. This design requires a service to specify
inputs and outputs early on and how to react when those channels are poked.
Static configuration, like topic names, subscription channels and so on are
setup as attributes of a `Service`. This `Service` object is then passed on to
surfkit's `Run` function. Run will do some initial setup, like preparing the web
server and afterwards call the function you've passed in. This function, the
runloopFn, should be used to create all the objects, configuration and state
required to run your service. It must be written in a non-blocking way so it can
return back to surfkit which will enable subscriptions, the webserver, signal
handling and will eventually enter a runloop by listening on the configured http
channel.

## HTTP

Surfkit exposes access to its web server in multiple ways. The simplest way is
to attach a HandleFunc to the built-in [gorilla
router](https://github.com/gorilla/mux). This is done in the runloop function of
a service. The server itself will be enabled after the runloopFn returns.

```go
package main

import "github.com/helloink/surfkit"

func main() { 

  s := surfkit.Service{ 
      Name:    "jupiter",
      Version: "0.1.0"
  }

  surfkit.Run(&s, func() { 
    s.Router.HandleFunc("/api/1/entities", customHandleFunc).Methods("POST")
  })
}
```

You can also use your very own router by replacing surfkit's prepared gorilla
router. Keep in mind that going custom means that you'll have to handle requests
to the health endpoint by yourself.

```go
surfkit.Run(&s, func() { 
  s.SrvHandler = yourOwnRouter
})
```

Custom routers are useful if you want to wrap the built-in router with another
one. For example in order to use the [CORS](https://github.com/rs/cors) package:

```go
surfkit.Run(&s, func() {
  s.SrvHandler = cors.Default().Handler(s.Router)
})
```

Another neat trick is to use gorilla's simple middleware system and use one of
the prepackaged middlewares (though, there is only one for atm).

```go
surfkit.Run(&s, func() {
  s.Router.Use(middleware.Logging)
})
```

Make sure to import the specific package:

```github.com/helloink/surfkit/middleware```

## Pubsub messageing

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

	surfkit.Run(&s, func() {
		// Noop
	})
}

// Return `true` if you want the underlying pubsub message to be acknowledged (ack)
// and `false` for nack.
func handleMessages(s *surfkit.Service, e *events.CloudEvent) bool {
	return true
}
```
