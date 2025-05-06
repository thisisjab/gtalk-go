package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/thisisjab/gchat-go/internal/data"
	"github.com/thisisjab/gchat-go/internal/mailer"
)

type APIServer struct {
	config *Config
	mailer *mailer.Mailer
	models *data.Models
	logger *slog.Logger
	wg     sync.WaitGroup
}

type Config struct {
	Cors struct {
		AllowedHeaders string
		AllowedMethods string
		TrustedOrigins []string
	}
	Environment string
	Port        int
	RateLimiter struct {
		Rps     int
		Burst   int
		Enabled bool
	}
	Version string
}

func NewServer(cfg *Config, db *sql.DB, mailer *mailer.Mailer, logger *slog.Logger) *APIServer {
	return &APIServer{
		config: cfg,
		mailer: mailer,
		models: data.NewModels(db),
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

		s.logger.Info("completing background tasks", "addr", srv.Addr)
		s.wg.Wait()

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

// background runs a function in a goroutine and logs any errors.
func (s *APIServer) background(fn func()) {
	s.wg.Add(1)

	go func() {
		defer s.wg.Done()

		defer func() {
			if err := recover(); err != nil {
				s.logger.Error(fmt.Sprintf("error while running background task: %v", err))
			}
		}()

		fn()
	}()
}
