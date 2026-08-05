// Command server は短縮 URL サービスを HTTP + JSON で公開する。
// 業務ロジックは internal/service にあり、ここは入口の組み立てだけを担う。
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

	"github.com/example/linkshort/internal/service"
	"github.com/example/linkshort/internal/store"
)

const shutdownTimeout = 10 * time.Second

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	svc := service.New(service.Config{Store: store.NewMemory(), Logger: logger})
	server := &http.Server{
		Addr:              envOr("ADDR", ":8080"),
		Handler:           withRequestLog(newRouter(svc), logger),
		ReadHeaderTimeout: 5 * time.Second,
	}

	if err := run(server, logger); err != nil {
		logger.Error("server stopped with error", "error", err)
		os.Exit(1)
	}
}

// run は SIGTERM を受けたら受付を止め、処理中のリクエストを待ってから終わる。
// Kubernetes は Pod を消す前に SIGTERM を送るので、これが無いと 502 が出る。
func run(server *http.Server, logger *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		logger.Info("server listening", "addr", server.Addr)
		errCh <- server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		logger.Info("shutdown signal received")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	}
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
