package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"

	"./server"
	"./tracer"
)

func main() {
	var (
		strictTrace = flag.Bool("strict", false, "Run traces in strict mode")
	)

	flag.Parse()

	mux := http.NewServeMux()
	mux.Handle("/", AdaptFunc(indexHandler, Tracing(*strictTrace), Logging()))

	srv := server.NewServer(":8080")

	log.Println("Starting server...")

	if err := srv.Start(mux); err != nil {
		log.Fatal(err)
	}

	log.Println("Server started ...")

	srv.Wait()

	log.Println("Shutting down server...")

	if err := srv.Stop(); err != nil {
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

			r = r.WithContext(tracer.NewContext(r.Context(), requestID))

			// switch out response writer for a recorder
			// for all subsequent handlers
			c := httptest.NewRecorder()
			h.ServeHTTP(c, r)

			// copy everything from response recorder
			// to actual response writer
			for k, v := range c.HeaderMap {
				w.Header()[k] = v
			}
			w.Header().Add("X-Post", "Tracing")
			w.WriteHeader(c.Code)
			c.Body.WriteTo(w)
		})
	}
}
