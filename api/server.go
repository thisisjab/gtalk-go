package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"
)

type APIServer struct {
	config *Config
	logger *slog.Logger
}

type Config struct {
	Environment string
	Port        int
}

func NewServer(cfg *Config, logger *slog.Logger) *APIServer {
	return &APIServer{
		config: cfg,
		logger: logger,
	}
}

func (s *APIServer) Start() error {
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", s.config.Port),
		Handler:      s.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		ErrorLog:     slog.NewLogLogger(s.logger.Handler(), slog.LevelError),
	}

	shutdownErr := make(chan error)

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt)

		sig := <-quit

		s.logger.Info("shutting down server", "signal", sig.String())

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := srv.Shutdown(ctx)

		if err != nil {
			shutdownErr <- err
		}

		shutdownErr <- nil
	}()

	s.logger.Info("starting server", "addr", srv.Addr, "env", s.config.Environment)

	err := srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	err = <-shutdownErr
	if err != nil {
		return err
	}

	s.logger.Info("stopped server", "addr", srv.Addr)

	return nil

}
