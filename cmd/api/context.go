package main

import (
	"context"
	"github.com/ejacobg/greenlight/internal/data"
	"net/http"
)

// It is recommended to use a custom type for our context keys.
type contextKey string

// We will use a predefined constant rather than using the literal value every time.
const userContextKey = contextKey("user")

// contextSetUser returns a copy of the given request with the user data attached to its context.
func (app *application) contextSetUser(r *http.Request, user *data.User) *http.Request {
	ctx := context.WithValue(r.Context(), userContextKey, user)
	return r.WithContext(ctx)
}

// contextGetUser will extract a User value from a request's context, panicking if it doesn't exist. Only call this helper when you expect a User value to be present.
func (app *application) contextGetUser(r *http.Request) *data.User {
	user, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		panic("missing user value in request context")
	}
	return user
}
