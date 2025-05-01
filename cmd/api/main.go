package main

import (
	"flag"
	"log/slog"
	"os"

	"github.com/thisisjab/gchat-go/api"
	"github.com/thisisjab/gchat-go/internal/envreader"
)

func main() {
	env := envreader.New("GTALK_")

	logLevel := flag.String("log-level", env.Choice("LOG_LEVEL", []string{"debug", "info", "warning", "error"}, "info"), "server log level (debug, info, warning, error)")

	apiCfg := &api.Config{}
	loadAPIConfig(env, apiCfg)

	flag.Parse()

	logger := setupLogger(*logLevel)

	server := api.NewServer(apiCfg, logger)
	if err := server.Start(); err != nil {
		logger.Error("failed to start server", "error", err)
		os.Exit(1)
	}
}

func setupLogger(logLevel string) *slog.Logger {
	l := slog.LevelInfo

	switch logLevel {
	case "debug":
		l = slog.LevelDebug
	case "info":
		l = slog.LevelInfo
	case "warning":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	default:
		l = slog.LevelInfo
	}

	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: l}))
}

func loadAPIConfig(env *envreader.EnvReader, cfg *api.Config) {
	flag.StringVar(&cfg.Environment, "environment", env.Choice("ENVIRONMENT", []string{"development", "production"}, "development"), "server environment (development, production)")
	flag.IntVar(&cfg.Port, "port", env.Int("PORT", 8000), "server port")
}
