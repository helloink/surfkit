package middleware

import (
	"net/http"
	"os"

	"github.com/gorilla/handlers"
)

// Logging Middleware that uses the Apache Common Log Format.
// Under the hood: https://godoc.org/github.com/gorilla/handlers#LoggingHandler
func Logging(next http.Handler) http.Handler {
	return handlers.LoggingHandler(os.Stdout, next)
}
