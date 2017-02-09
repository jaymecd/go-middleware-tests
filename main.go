package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"./tracer"
)

func main() {
	var (
		strictTrace = flag.Bool("strict", false, "Run traces in strict mode")
	)

	flag.Parse()

	// subscribe to SIGINT signals
	stopChan := make(chan os.Signal)
	signal.Notify(stopChan, os.Interrupt)

	mux := http.NewServeMux()
	mux.Handle("/", AdaptFunc(indexHandler, Tracing(*strictTrace), Logging()))

	srv := &http.Server{Addr: ":8080", Handler: mux}

	// service connections
	go func() {
		log.Println("Starting server...")
		log.Fatal(srv.ListenAndServe())
	}()

	<-stopChan // wait for SIGINT

	// shut down gracefully, but wait no longer than 5 seconds before halting
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Println("Shutting down server...")

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal(err)
	}

	log.Println("Server gracefully stopped")
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	requestID, ok := tracer.FromContext(r.Context())

	log.Println("REQUEST ID:", requestID)

	if ok {
		fmt.Fprintf(w, "My Request-Id: %s\n", requestID)
	} else {
		fmt.Fprintln(w, "No Request-Id detected")
	}
}

// From https://medium.com/@matryer/writing-middleware-in-golang-and-how-go-makes-it-so-much-fun-4375c1246e81#.qhnydpwrp

// Adapter ...
type Adapter func(http.Handler) http.Handler

// Adapt ...
func Adapt(h http.Handler, adapters ...Adapter) http.Handler {
	for _, adapter := range adapters {
		h = adapter(h)
	}
	return h
}

// AdaptFunc ...
func AdaptFunc(fn http.HandlerFunc, adapters ...Adapter) http.Handler {
	return Adapt(http.HandlerFunc(fn), adapters...)
}

// Logging ...
func Logging() Adapter {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				w.Header().Add("X-Post", "Logging")
				log.Println("Logging: After")
			}()

			w.Header().Add("X-Pre", "Logging")
			log.Println("Logging: Before")

			log.Println("Logger: ", r.Method, r.URL.Path)

			h.ServeHTTP(w, r)
		})
	}
}

// Tracing ...
func Tracing(isMandatory bool) Adapter {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				w.Header().Add("X-Port", "Tracing")
				log.Println("Tracing: After")
			}()

			w.Header().Add("X-Pre", "Tracing")
			log.Println("Tracing: Before")

			requestID, err := tracer.FromRequest(r)

			if err != nil {
				if isMandatory {
					log.Printf("ERROR: %s", err)
					http.Error(w, err.Error(), 400)
					return
				}

				requestID = tracer.GenerateRandomID()
			}

			ctx := tracer.NewContext(r.Context(), requestID)

			h.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
