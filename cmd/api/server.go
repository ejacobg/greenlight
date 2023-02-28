package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// serve will create and run a server for our application.
func (app *application) serve() error {
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", app.config.port),
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// Channel used to receive the error returned by the Shutdown() method.
	shutdownError := make(chan error)

	// Spin up a goroutine that will just listen for OS signals.
	// This goroutine will intercept the SIGINT and SIGTERM signals, logging and shutting down the app if found.
	go func() {
		// Listen and intercept the given signals.
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		s := <-quit // This line will block until a signal is received.

		// Log the caught signal and exit.
		app.logger.PrintInfo("shutting down server", map[string]string{
			"signal": s.String(),
		})

		// Allow any in-flight requests 5 seconds to finish their work before shutting them down.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Block while waiting for the server to shut down.
		err := srv.Shutdown(ctx)
		if err != nil {
			// Report any shutdown errors we come across.
			// Note that if we find an error, we will no longer wait for the background tasks.
			shutdownError <- err
		}

		// Once the server has shut down, wait for any remaining background tasks.
		app.logger.PrintInfo("completing background tasks", map[string]string{
			"addr": srv.Addr,
		})

		app.wg.Wait()

		// Once all tasks have finish, continue with the shutdown.
		shutdownError <- nil
	}()

	app.logger.PrintInfo("starting server", map[string]string{
		"addr": srv.Addr,
		"env":  app.config.env,
	})

	// Calling Shutdown() will return http.ErrServerClosed. If the server is closed for another reason, return the error.
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	// Otherwise, wait for the shutdown to complete. Return any errors.
	if err := <-shutdownError; err != nil {
		return err
	}

	// Write a log indicating successful shutdown.
	app.logger.PrintInfo("stopped server", map[string]string{
		"addr": srv.Addr,
	})

	return nil
}
