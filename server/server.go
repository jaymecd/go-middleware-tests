package server

import (
	"context"
	"log"
	"net"
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
		ttl:      2 * time.Second,
	}
}

// Start ...
func (s *Server) Start(mux http.Handler) error {
	log.Println("Starting server...")

	if mux == nil {
		mux = http.DefaultServeMux
	}

	lnc := make(chan net.Listener, 1)
	errc := make(chan error, 1)

	ln, err := net.Listen("tcp", s.bind)
	if err != nil {
		return err
	}

	lnc <- ln

	s.srv = &http.Server{
		Addr:    s.bind,
		Handler: mux,
	}

	go func() {
		errc <- s.srv.Serve(ln)
	}()

	select {
	case err := <-errc:
		return err
	case ln = <-lnc:
		log.Printf("Listening on %s\n", ln.Addr())
		return nil
	}
}

// Wait ...
func (s *Server) Wait() {
	// subscribe to SIGINT signals
	signal.Notify(s.stopChan, os.Interrupt)

	log.Println("Accepting connections ...")

	<-s.stopChan // wait for SIGINT
}

// Stop ...
func (s *Server) Stop() error {
	log.Println("Shutting down server...")

	// shut down gracefully, but wait no longer than 5 seconds before halting
	ctx, cancel := context.WithTimeout(context.Background(), s.ttl)
	defer cancel()

	err := s.srv.Shutdown(ctx)

	if err != nil {
		return err
	}

	log.Println("Server gracefully stopped")

	return nil
}
