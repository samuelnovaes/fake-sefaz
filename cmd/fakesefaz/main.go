package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vendermais/fake-sefaz/internal/config"
	"github.com/vendermais/fake-sefaz/internal/nfe"
	"github.com/vendermais/fake-sefaz/internal/scenario"
	"github.com/vendermais/fake-sefaz/internal/server"
	"github.com/vendermais/fake-sefaz/internal/store"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	settings := config.Load()

	documents := store.New()
	scenarios := scenario.New()
	service := nfe.NewService(documents, scenarios, nfe.Options{
		CancellationWindow:       settings.CancellationWindow,
		CancellationWindowNFCe:   settings.CancellationWindowNFCe,
		MaxDistributionDocuments: settings.MaxDistributionDocuments,
	}, time.Now)

	httpServer := &http.Server{
		Addr:              settings.Address,
		Handler:           server.New(documents, scenarios, service, logger),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		logger.Info("fake-sefaz listening", "address", settings.Address)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server stopped", "error", err)
			os.Exit(1)
		}
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	<-signals

	shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdown); err != nil {
		logger.Error("shutdown failed", "error", err)
	}
}
