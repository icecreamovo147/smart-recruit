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

	devlogviewer "smart-recruit/dev-log-viewer"
	"smart-recruit/dev-log-viewer/internal/catalog"
	"smart-recruit/dev-log-viewer/internal/config"
	"smart-recruit/dev-log-viewer/internal/server"
	"smart-recruit/dev-log-viewer/internal/stream"
	"smart-recruit/dev-log-viewer/internal/tailer"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		slog.Error("dev-log-viewer stopped", "component", "http", "error", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	cfg, err := config.Load(args, os.LookupEnv)
	if err != nil {
		return err
	}
	serviceCatalog, err := catalog.New(cfg.Root)
	if err != nil {
		return err
	}
	staticFS, err := devlogviewer.StaticFS()
	if err != nil {
		return err
	}

	hub := stream.NewHub(stream.DefaultEventBuffer, stream.DefaultClientQueue)
	coordinator := tailer.NewCoordinator(cfg.Root, serviceCatalog.Definitions(), hub)
	ctx, stopCoordinator := context.WithCancel(context.Background())
	defer stopCoordinator()
	go coordinator.Start(ctx)

	httpServer := &http.Server{
		Addr:              cfg.Addr,
		Handler:           server.NewWithStreamAndStatic(serviceCatalog, hub, coordinator, staticFS).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("dev-log-viewer listening", "component", "http")
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signalCh)

	select {
	case err := <-errCh:
		return err
	case <-signalCh:
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return httpServer.Shutdown(ctx)
	}
}
