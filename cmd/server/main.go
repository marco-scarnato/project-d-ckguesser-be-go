package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/marco-scarnato/multiservices-backend/libs/platform/health"
	"github.com/marco-scarnato/multiservices-backend/libs/platform/logging"
	"github.com/marco-scarnato/multiservices-backend/libs/platform/messaging"
	"github.com/marco-scarnato/multiservices-backend/libs/platform/mongo"
	"github.com/marco-scarnato/multiservices-backend/libs/platform/postgres"
	serviceconfig "github.com/marco-scarnato/multiservices-backend/services/project-d-ckguesser-be-go/internal/config"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		addr := os.Getenv("HTTP_ADDR")
		if addr == "" {
			addr = ":8081"
		}
		if err := health.Probe(addr, "/health/ready", 2*time.Second); err != nil {
			fmt.Fprintf(os.Stderr, "healthcheck failed: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "ckguesser-be stopped: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := serviceconfig.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}
	logger, err := logging.New(cfg.LogLevel, cfg.ServiceName, cfg.AppEnv, cfg.AppVersion)
	if err != nil {
		return fmt.Errorf("configure logging: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	natsClient, err := messaging.Connect(ctx, cfg.NATSURL, cfg.ServiceName, cfg.StartupTimeout)
	if err != nil {
		logger.Error("NATS connection failed", "error", err)
		return err
	}

	dbCheckers := make(map[string]health.Checker)

	// Connect PostgreSQL if enabled
	var pgClient *postgres.Client
	if cfg.PostgresEnabled {
		pgClient, err = postgres.Connect(ctx, cfg.PostgresDSN, cfg.PostgresMaxConns, cfg.DBConnectTimeout)
		if err != nil {
			_ = natsClient.Close(context.Background())
			logger.Error("PostgreSQL connection failed", "error", err)
			return fmt.Errorf("connect to postgres: %w", err)
		}
		defer pgClient.Close()
		dbCheckers["postgres"] = pgClient
		logger.Info("connected to dedicated PostgreSQL")
	}

	// Connect MongoDB if enabled
	var mongoClient *mongo.Client
	if cfg.MongoEnabled {
		mongoClient, err = mongo.Connect(ctx, cfg.MongoURI, cfg.MongoDatabase, cfg.MongoMaxPoolSize, cfg.DBConnectTimeout)
		if err != nil {
			_ = natsClient.Close(context.Background())
			logger.Error("MongoDB connection failed", "error", err)
			return fmt.Errorf("connect to mongo: %w", err)
		}
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
			defer cancel()
			_ = mongoClient.Close(shutdownCtx)
		}()
		dbCheckers["mongodb"] = mongoClient
		logger.Info("connected to dedicated MongoDB")
	}

	// Standardized health check server with checkers
	healthServer := health.NewWithCheckers(natsClient.Ready, dbCheckers)

	// Standardized NATS health responder
	subs, err := health.RegisterNATSHealthResponder(
		natsClient.Conn(),
		logger,
		cfg.ServiceName,
		natsClient.Ready,
		dbCheckers,
		cfg.ReadinessTimeout,
	)
	if err != nil {
		_ = natsClient.Close(context.Background())
		return fmt.Errorf("subscribe to NATS: %w", err)
	}
	defer func() {
		for _, sub := range subs {
			if sub != nil {
				_ = sub.Drain()
			}
		}
	}()

	httpServer := &http.Server{
		Handler:           healthServer.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    cfg.MaxPayloadBytes,
	}
	listener, err := net.Listen("tcp", cfg.HTTPAddr)
	if err != nil {
		_ = natsClient.Close(context.Background())
		return fmt.Errorf("listen on %s: %w", cfg.HTTPAddr, err)
	}
	healthServer.SetReady(true)
	logger.Info("service started", "http_addr", cfg.HTTPAddr)

	serverErr := make(chan error, 1)
	go func() { serverErr <- httpServer.Serve(listener) }()

	select {
	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		logger.Info("shutdown requested")
	}

	healthServer.BeginShutdown()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP shutdown failed", "error", err)
	}
	if err := natsClient.Close(shutdownCtx); err != nil {
		logger.Error("NATS shutdown failed", "error", err)
	}
	logger.Info("service stopped")
	return nil
}
