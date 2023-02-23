package main

import (
	"fmt"
	"golang.org/x/time/rate"
	"net/http"
)

func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// When a panic occurs and the runtime unwinds the stack, this function will be called.
		defer func() {
			// Under normal circumstances, this function will be called, so make sure a panic has actually occurred.
			if err := recover(); err != nil {
				// If a panic was detected, we will close our HTTP connection.
				w.Header().Set("Connection", "close")
				// Return the error with response code 500.
				app.serverErrorResponse(w, r, fmt.Errorf("%s", err))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (app *application) rateLimit(next http.Handler) http.Handler {
	// Create a new rate limiter with a refill rate of 2 events/tokens per second, with a maximum capacity of 4 events/tokens.
	limiter := rate.NewLimiter(2, 4)

	// All calls to this function will pull from the above rate limiter.
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if there are enough tokens to perform an event, returning a 429 Too Many Requests response if there aren't.
		if !limiter.Allow() {
			app.rateLimitExceededResponse(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}
