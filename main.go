package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"

	"github.com/go-errors/errors"
	"github.com/justinas/alice"

	"./server"
	"./tracer"
)

func main() {
	var (
		strictTrace = flag.Bool("strict", false, "Run traces in strict mode")
	)

	flag.Parse()

	stdWrappers := alice.New(Tracing(*strictTrace), Logging, Recovering)

	mux := http.NewServeMux()
	mux.Handle("/", stdWrappers.ThenFunc(indexHandler))

	srv := server.NewServer(":8080")

	if err := srv.Start(mux); err != nil {
		log.Fatal(err)
	}

	srv.Wait()

	if err := srv.Stop(); err != nil {
		log.Fatal(err)
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	requestID, ok := tracer.FromContext(r.Context())

	log.Println("REQUEST ID:", requestID)

	if ok {
		if "43" == requestID {
			panic("F*#$!")
		}

		fmt.Fprintf(w, "My Request-Id: %s\n", requestID)
	} else {
		fmt.Fprintln(w, "No Request-Id detected")
	}
}

// Recovering ...
func Recovering(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Println(errors.Wrap(err, 3).ErrorStack())

				http.Error(w, http.StatusText(500), 500)
			}
		}()

		h.ServeHTTP(w, r)
	})
}

// Logging ...
func Logging(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			log.Println("Logging: After")
		}()

		w.Header().Add("X-Pre", "Logging")
		log.Println("Logging: Before")

		log.Println("Logger: ", r.Method, r.URL.Path)

		h.ServeHTTP(w, r)
	})
}

// Tracing ...
func Tracing(isMandatory bool) func(h http.Handler) http.Handler {
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
