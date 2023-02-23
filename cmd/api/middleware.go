package main

import (
	"fmt"
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
