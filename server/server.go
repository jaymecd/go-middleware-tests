package server

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
)

// Server ...
type Server struct {
	bind     string
	stopChan chan os.Signal
	srv      *http.Server
	ttl      time.Duration
}

// NewServer ...
func NewServer(bind string) *Server {
	return &Server{
		bind:     bind,
		stopChan: make(chan os.Signal),
		ttl:      5 * time.Second,
	}
}

// Start ...
func (s *Server) Start(mux http.Handler) error {
	if mux == nil {
		mux = http.DefaultServeMux
	}

	// subscribe to SIGINT signals
	signal.Notify(s.stopChan, os.Interrupt)

	s.srv = &http.Server{
		Addr:    s.bind,
		Handler: mux,
	}

	// move to errChan
	go func() {
		log.Fatal(s.srv.ListenAndServe())
	}()

	return nil
}

// Wait ...
func (s *Server) Wait() {
	<-s.stopChan // wait for SIGINT
}

// Stop ...
func (s *Server) Stop() error {
	// shut down gracefully, but wait no longer than 5 seconds before halting
	ctx, cancel := context.WithTimeout(context.Background(), s.ttl)
	defer cancel()

	err := s.srv.Shutdown(ctx)

	if err != nil {
		return err
	}

	return nil
}
