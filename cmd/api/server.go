package main

import (
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

	// Spin up a goroutine that will just listen for OS signals.
	// This goroutine will intercept the SIGINT and SIGTERM signals, logging and shutting down the app if found.
	go func() {
		quit := make(chan os.Signal, 1)

		// Listen for the given signals.
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		// This line will block until a signal is received.
		s := <-quit

		// Log the caught signal and exit.
		app.logger.PrintInfo("caught signal", map[string]string{
			"signal": s.String(),
		})
		os.Exit(0)
	}()

	app.logger.PrintInfo("starting server", map[string]string{
		"addr": srv.Addr,
		"env":  app.config.env,
	})

	return srv.ListenAndServe()
}
