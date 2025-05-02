package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/thisisjab/gchat-go/api"
	"github.com/thisisjab/gchat-go/internal/database"
	"github.com/thisisjab/gchat-go/internal/envreader"
	"github.com/thisisjab/gchat-go/internal/mailer"
)

func main() {
	env := envreader.New("GTALK_")

	logLevel := flag.String("log-level", env.Choice("LOG_LEVEL", []string{"debug", "info", "warning", "error"}, "info"), "server log level (debug, info, warning, error)")

	apiCfg := &api.Config{}
	loadAPIConfig(env, apiCfg)

	dbCfg := &database.Config{}
	loadDatabaseConfig(env, dbCfg)

	mailerCfg := &mailer.Config{}
	loadMailerConfig(env, mailerCfg)

	flag.Parse()

	logger := setupLogger(*logLevel)

	database, err := database.OpenDB(*dbCfg)
	if err != nil {
		logger.Error("failed to open database", "error", err)
		os.Exit(1)
	}

	mailer := mailer.New(*mailerCfg)

	server := api.NewServer(apiCfg, database, mailer, logger)
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
	flag.StringVar(&cfg.Version, "version", env.String("VERSION", "1.0"), "server version (1.0 by default).")

	// CORS
	flag.StringVar(&cfg.Cors.AllowedHeaders, "cors-allowed-headers", env.String("CORS_ALLOWED_HEADERS", "Content-Type, Authorization"), "Allowed CORS headers (comma separated)")
	flag.StringVar(&cfg.Cors.AllowedMethods, "cors-allowed-methods", env.String("CORS_ALLOWED_METHODS", "POST, PATCH, DELETE"), "Allowed CORS methods (comma separated)")
	flag.Func("cors-trusted-origins", "Trusted CORS origins (space separated)", func(val string) error {
		if val == "" {
			val = env.String("CORS_TRUSTED_ORIGINS", "")
		}

		cfg.Cors.TrustedOrigins = strings.Fields(val)

		return nil
	})
}

func loadDatabaseConfig(env *envreader.EnvReader, cfg *database.Config) {
	flag.StringVar(&cfg.DSN, "db-dsn", env.String("DB_DSN", ""), "database dsn (postgres only)")
	flag.IntVar(&cfg.MaxOpenConns, "db-max-open-conn", env.Int("DB_MAX_OPEN_CONN", 20), "database max open connections (default: 20)")
	flag.IntVar(&cfg.MaxIdleConns, "db-max-idle-conn", env.Int("DB_MAX_IDLE_CONN", 20), "database max idle connections (default: 20)")
	flag.DurationVar(&cfg.MaxIdleTime, "db-max-idle-time", env.Duration("DB_MAX_IDLE_TIME", 15*time.Minute), "database max idle time (default: 15 minutes)")
}

func loadMailerConfig(env *envreader.EnvReader, cfg *mailer.Config) {
	flag.StringVar(&cfg.Host, "mailer-host", env.String("MAILER_HOST", ""), "mailer host")
	flag.IntVar(&cfg.Port, "mailer-port", env.Int("MAILER_PORT", 587), "mailer port")
	flag.StringVar(&cfg.Username, "mailer-username", env.String("MAILER_USERNAME", ""), "mailer username")
	flag.StringVar(&cfg.Password, "mailer-password", env.String("MAILER_PASSWORD", ""), "mailer password")

	senderName := flag.String("mailer-sender-name", env.String("MAILER_SENDER_NAME", ""), "mailer sender name")
	senderEmail := flag.String("mailer-sender-email", env.String("MAILER_SENDER_EMAIL", ""), "mailer sender email")

	cfg.Sender = fmt.Sprintf("%s <%s>", *senderName, *senderEmail)
}
