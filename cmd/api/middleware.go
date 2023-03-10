package main

import (
	"errors"
	"fmt"
	"github.com/ejacobg/greenlight/internal/data"
	"github.com/ejacobg/greenlight/internal/validator"
	"golang.org/x/exp/slices"
	"golang.org/x/time/rate"
	"net"
	"net/http"
	"strings"
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
	// Ignore this middleware if rate limiting is disabled.
	if !app.config.limiter.enabled {
		return next
	}

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
			clients[ip] = &client{limiter: rate.NewLimiter(rate.Limit(app.config.limiter.rps), app.config.limiter.burst)}
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

// authenticate will read the Authorization header of the request, extract the authentication token, then attach the appropriate *User for that token.
// If the Authorization header does not exist, then the data.AnonymousUser value will be attached instead.
// If the authorization token is invalid, a 401 Unauthorized response will be returned.
func (app *application) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Adding the "Vary: Authorization" header tells any caches that this response may vary based on the value of the request's Authorization header.
		w.Header().Add("Vary", "Authorization")

		// Retrieve the Authorization header from the request. If it doesn't exist, this will return "".
		authorizationHeader := r.Header.Get("Authorization")

		// If the Authorization header was not given, attach the data.AnonymousUser value.
		if authorizationHeader == "" {
			r = app.contextSetUser(r, data.AnonymousUser)
			next.ServeHTTP(w, r)
			return
		}

		// If the Authorization header was given, confirm that its value is of the form: Bearer <token>
		headerParts := strings.Split(authorizationHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}

		// Extract the token value.
		token := headerParts[1]

		// Validate the token value.
		v := validator.New()
		if data.ValidateTokenPlaintext(v, token); !v.Valid() {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}

		// Get the user associated with this token. Note that this token is being used for authentication, so set the scope appropriately.
		user, err := app.models.Users.GetForToken(data.ScopeAuthentication, token)
		if err != nil {
			switch {
			case errors.Is(err, data.ErrRecordNotFound):
				app.invalidAuthenticationTokenResponse(w, r)
			default:
				app.serverErrorResponse(w, r, err)
			}
			return
		}

		// Apply the *User value to the request context.
		r = app.contextSetUser(r, user)

		next.ServeHTTP(w, r)
	})
}

// requireAuthenticatedUser checks if the user is anonymous. If they are, then a 401 Unauthorized response will be returned.
func (app *application) requireAuthenticatedUser(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := app.contextGetUser(r)

		if user.IsAnonymous() {
			app.authenticationRequiredResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// requireActivatedUser will restrict access to a handler to only those requests that have a valid *User value attached to them.
func (app *application) requireActivatedUser(next http.HandlerFunc) http.HandlerFunc {
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract the user from the request context.
		user := app.contextGetUser(r)

		// The user account must be activated to access this resource.
		if !user.Activated {
			app.inactiveAccountResponse(w, r)
			return
		}

		// Otherwise, this user is valid and activated.
		next.ServeHTTP(w, r)
	})

	// Wrapping with requireAuthenticatedUser will ensure that the *User value is non-anonymous.
	return app.requireAuthenticatedUser(fn)
}

// requirePermission will protect access to a handler if the request context's *User value does not have the requisite permissions.
func (app *application) requirePermission(code string, next http.HandlerFunc) http.HandlerFunc {
	fn := func(w http.ResponseWriter, r *http.Request) {
		// Retrieve the user from the request context.
		user := app.contextGetUser(r)

		// Get this user's permissions.
		permissions, err := app.models.Permissions.GetAllForUser(user.ID)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}

		// If the user does not have the required permission, return 403 Forbidden.
		if !permissions.Include(code) {
			app.notPermittedResponse(w, r)
			return
		}

		// If the user does have the permission, continue with the next middleware.
		next.ServeHTTP(w, r)
	}

	// Only activated users have permissions. If the user is not activated, reject this request.
	return app.requireActivatedUser(fn)
}

// enableCORS will tell the browser to grant our trusted origins the ability to read our responses. This method will also respond to any preflight CORS requests.
func (app *application) enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This response may change depending on the value of the "Origin" header.
		w.Header().Add("Vary", "Origin")

		// This response may change depending on if this header is present.
		w.Header().Add("Vary", "Access-Control-Request-Method")

		// Get the value of the request's Origin header.
		origin := r.Header.Get("Origin")

		// If the Origin header is present, check to see if it is one of our trusted origins.
		if origin != "" && slices.Contains(app.config.cors.trustedOrigins, origin) {
			// If the origin is trusted, then set our CORS header appropriately.
			w.Header().Set("Access-Control-Allow-Origin", origin)

			// If this is an OPTIONS request with the Origin and Access-Control-Request-Method headers set, then treat this as a preflight request.
			if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
				// We will send the same preflight response headers for all preflight requests.
				w.Header().Set("Access-Control-Allow-Methods", "OPTIONS, PUT, PATCH, DELETE")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")

				// End this request with a 200 OK response.
				w.WriteHeader(http.StatusOK)
				return
			}
		}

		// Call the next handler in the chain.
		next.ServeHTTP(w, r)
	})
}
