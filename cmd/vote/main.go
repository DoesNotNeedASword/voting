package main

import (
	partyclient "Voting/internal/clients/party"
	userclient "Voting/internal/clients/user"
	"Voting/internal/shared/config"
	"Voting/internal/shared/database"
	"Voting/internal/shared/logger"
	"Voting/internal/shared/migration"
	"Voting/internal/vote/handler/http"
	"Voting/internal/vote/repository"
	"Voting/internal/vote/service"
	"context"
	"errors"
	"log/slog"
	nethttp "net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}

	log := logger.New(cfg.LogLevel)
	if err := run(cfg, log); err != nil {
		log.Error("run server", "error", err)
		os.Exit(1)
	}
}

func run(cfg config.Config, log *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if cfg.RunMigrations {
		if err := migration.Run(cfg.DatabaseURL, log); err != nil {
			return err
		}
	}

	pool, err := database.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	voteRepository := repository.New(pool)
	partyService := service.New(voteRepository)
	userClient, err := userclient.New(cfg.UserServiceAddress)
	if err != nil {
		return err
	}
	partyClient, err := partyclient.New(cfg.PartyServiceAddress)
	if err != nil {
		return err
	}
	handler := http.New(partyService, userClient, partyClient)

	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	server := &nethttp.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("http server started", "addr", cfg.HTTPAddr)
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	case err := <-errCh:
		if errors.Is(err, nethttp.ErrServerClosed) {
			return nil
		}
		return err
	}
}
