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

	"github.com/vendermais/fake-sefaz/internal/authorizer"
	"github.com/vendermais/fake-sefaz/internal/config"
	"github.com/vendermais/fake-sefaz/internal/dfews"
	"github.com/vendermais/fake-sefaz/internal/nfe"
	"github.com/vendermais/fake-sefaz/internal/sat"
	"github.com/vendermais/fake-sefaz/internal/scenario"
	"github.com/vendermais/fake-sefaz/internal/server"
	"github.com/vendermais/fake-sefaz/internal/store"
	"github.com/vendermais/fake-sefaz/internal/xsd"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	settings := config.Load()

	engine := authorizer.New(store.New(), scenario.New(), authorizer.Options{
		CancellationWindow:       settings.CancellationWindow,
		CancellationWindowNFCe:   settings.CancellationWindowNFCe,
		SubstitutionWindow:       settings.SubstitutionWindow,
		OfflineDeadline:          settings.OfflineDeadline,
		MaxDistributionDocuments: settings.MaxDistributionDocuments,
	}, time.Now)

	handler := server.New(engine, logger, nfe.NewService(engine), dfews.NewService(engine))
	handler.Mount("POST /sat/{command}", sat.NewService(engine))

	if settings.SchemaDirectory != "" {
		schemas, err := xsd.Load(settings.SchemaDirectory)
		if err != nil {
			logger.Error("schemas could not be loaded", "directory", settings.SchemaDirectory, "error", err)
			os.Exit(1)
		}
		handler.UseSchemas(schemas)
		logger.Info("schemas loaded", "directory", settings.SchemaDirectory,
			"documents", schemas.Documents(), "roots", len(schemas.Roots()))
	}

	httpServer := &http.Server{
		Addr:              settings.Address,
		Handler:           handler,
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
