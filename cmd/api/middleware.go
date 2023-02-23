package main

import (
	"fmt"
	"golang.org/x/time/rate"
	"net"
	"net/http"
	"sync"
	"time"
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
	type client struct {
		limiter *rate.Limiter
		// Add a timestamp to each map entry to determine if it needs to be deleted.
		lastSeen time.Time
	}
	var (
		mu      sync.Mutex
		clients = make(map[string]*client) // Not sure why we store pointers?
	)

	// Remove old entries every minute.
	go func() {
		for {
			time.Sleep(time.Minute)

			mu.Lock()
			// If a client hasn't been seen for more than 3 minutes, then remove them from the map.
			for ip, client := range clients {
				if time.Since(client.lastSeen) > 3*time.Minute {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract the client's IP address from the request.
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}

		mu.Lock()
		// Find the IP's rate limiter, creating one if it doesn't exist.
		if _, found := clients[ip]; !found {
			clients[ip] = &client{limiter: rate.NewLimiter(2, 4)}
		}

		// Update the last seen time for the client.
		clients[ip].lastSeen = time.Now()

		// Determine if there are enough tokens to make a request.
		if !clients[ip].limiter.Allow() {
			// Remove our access before leaving.
			mu.Unlock()
			app.rateLimitExceededResponse(w, r)
			return
		}

		// Manually unlock (as opposed to using defer) so that other goroutines don't have to wait until the response cycle is finished to be able to read.
		mu.Unlock()
		next.ServeHTTP(w, r)
	})
}
