// link サービス: gRPC で短縮リンクを作り、Spanner に保存する。
package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	linkv1 "github.com/example/microservices/gen/link/v1"
	"github.com/example/microservices/services/link/internal/server"
	"github.com/example/microservices/services/link/internal/store"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthv1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	if err := run(); err != nil {
		slog.Error("起動に失敗しました", "error", err)
		os.Exit(1)
	}
}

func run() error {
	// SIGTERM を受けたら ctx が閉じる。Kubernetes はまずこれを送ってくる
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	database := env("SPANNER_DATABASE", "projects/learning-project/instances/learning-instance/databases/links")
	// 既定値は衝突しにくい番号にしてある。Kubernetes 上では 8080 を渡す
	addr := env("LISTEN_ADDR", ":19001")

	spannerStore, err := store.NewSpanner(ctx, database)
	if err != nil {
		return err
	}
	defer spannerStore.Close()

	grpcServer := grpc.NewServer()
	linkv1.RegisterLinkServiceServer(grpcServer, server.New(server.Config{Store: spannerStore}))

	// kubelet の probe と grpcurl のために足しておく
	healthv1.RegisterHealthServer(grpcServer, health.NewServer())
	reflection.Register(grpcServer)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", addr, "database", database)
		if serveErr := grpcServer.Serve(listener); serveErr != nil && !errors.Is(serveErr, grpc.ErrServerStopped) {
			errCh <- serveErr
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	// 処理中のリクエストを終わらせてから閉じる。落としてよいのはその後
	slog.Info("shutting down")
	done := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(20 * time.Second):
		slog.Warn("graceful stop がタイムアウトしました。強制的に閉じます")
		grpcServer.Stop()
	}
	return nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
