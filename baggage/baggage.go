package baggage

import (
	"context"
	"net/http"
	"os"
)

// baggage is a custom context key for storing the auth token.
type baggage struct{}

func withBaggage(ctx context.Context, i interface{}) context.Context {
	return context.WithValue(ctx, baggage{}, i)
}

// WithInfomationFromRequest sends the information as a baggage
func WithInfomationFromRequest(i interface{}) func(context.Context, *http.Request) context.Context {
	return func(ctx context.Context, r *http.Request) context.Context {
		return withBaggage(ctx, i)
	}
}

// WithInfomation sends the information as a baggage
func WithInfomation(i interface{}) func(context.Context) context.Context {
	return func(ctx context.Context) context.Context {
		return withBaggage(ctx, i)
	}
}

// WithTokenFromRequest sends the token as a baggage
func WithTokenFromRequest(ctx context.Context, r *http.Request) context.Context {
	return withBaggage(ctx, r.Header.Get("Authorization"))
}

// WithTokenFromEnv sends the token as a baggage
func WithTokenFromEnv(ctx context.Context) context.Context {
	return withBaggage(ctx, os.Getenv("API_KEY"))
}

// BaggageFromContext extracts the information from the context
func BaggageFromContext(ctx context.Context) interface{} {
	return ctx.Value(baggage{})
}
