package surfkit

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

const defaultTimeout = 60 * time.Second

func setupServer(s *Service) {
	s.Router = mux.NewRouter()
	s.Router.HandleFunc("/", healthEndpoint).Methods("GET")

	s.SrvHandler = s.Router
}

func healthEndpoint(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func enableServer(s *Service) error {

	timeout := getTimeout(s)

	s.Srv = &http.Server{
		Handler:      s.SrvHandler,
		Addr:         fmt.Sprintf(":%s", s.Env.Port),
		WriteTimeout: timeout,
		ReadTimeout:  timeout,
	}

	log.Printf("{SK}[DEBUG] Writetime: %v, Read: %v", s.Srv.WriteTimeout, s.Srv.ReadTimeout)

	log.Printf("Server enabled on port %s", s.Env.Port)
	return s.Srv.ListenAndServe()
}

func shutdownServer(s *Service) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() { cancel() }()

	err := s.Srv.Shutdown(ctx)
	if err != nil {
		log.Fatalf("Server Shutdown Failed: %+v", err)
	}
}

// getTimeout from user configuration or take defaults
func getTimeout(s *Service) time.Duration {
	if s.SrvTimeout == 0 {
		return defaultTimeout
	}

	return s.SrvTimeout
}
