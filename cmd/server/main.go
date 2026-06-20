package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/selivanov-tech/go-stt/internal/api"
	"github.com/selivanov-tech/go-stt/internal/config"
	"github.com/selivanov-tech/go-stt/internal/queue"
	"github.com/selivanov-tech/go-stt/internal/store"
	"github.com/selivanov-tech/go-stt/internal/worker"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))

	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		slog.Error("invalid config", "err", err)
		os.Exit(1)
	}

	st, err := store.Build(cfg.Store)
	if err != nil {
		slog.Error("store init failed", "err", err)
		os.Exit(1)
	}

	var sup *worker.Supervisor
	if cfg.WorkerAutostart {
		sup = worker.New(cfg.WorkerCmd, cfg.WorkerDir)
		if err := sup.Start(); err != nil {
			slog.Error("worker start failed", "err", err)
			os.Exit(1)
		}
		if err := worker.WaitHealthy(cfg.WorkerURL, 60*time.Second); err != nil {
			slog.Warn("worker not healthy", "err", err)
		}
	}

	q := queue.NewInProc(cfg.QueueWorkers, 256)
	srv := api.NewServer(cfg, q, st)

	httpServer := &http.Server{Addr: cfg.Addr, Handler: srv.Routes()}

	go func() {
		slog.Info("go-stt listening",
			"addr", cfg.Addr,
			"profile", cfg.Profile,
			"default_engine", cfg.DefaultEngine,
			"worker", cfg.WorkerURL,
		)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("listen failed", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	slog.Info("shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(ctx)
	_ = q.Shutdown(ctx)
	if sup != nil {
		sup.Stop()
	}
}
