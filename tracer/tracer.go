package tracer

import (
	"context"
	"errors"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/oklog/ulid"
)

const (
	keyRequestID contextKey = iota
)

// contextKey is unexported to prevent collisions with context keys.
type contextKey int

// ErrInvalidHeader ....
var ErrInvalidHeader = errors.New("Header key is not provided or empty.")

// FromRequest ...
func FromRequest(r *http.Request) (string, error) {
	val := strings.Trim(r.Header.Get("X-Request-ID"), " ")

	if "" == val {
		return "", ErrInvalidHeader
	}

	return val, nil
}

// FromContext ...
func FromContext(ctx context.Context) (string, bool) {
	requestID := ctx.Value(keyRequestID).(string)

	if requestID == "" {
		return "", false
	}

	return requestID, true
}

// NewContext ...
func NewContext(ctx context.Context, requestID string) context.Context {
	if requestID == "" {
		return ctx
	}

	return context.WithValue(ctx, keyRequestID, requestID)
}

// GenerateRandomID ...
func GenerateRandomID() string {
	timeNow := time.Now()
	entropy := rand.New(rand.NewSource(timeNow.UnixNano()))

	return ulid.MustNew(ulid.Timestamp(timeNow), entropy).String()
}
