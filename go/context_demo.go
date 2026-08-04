package main

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// context を第1引数で受け取り、待つ側は必ず ctx.Done() も見る
func callAPI(ctx context.Context, name string, took time.Duration) (string, error) {
	select {
	case <-time.After(took):
		return fmt.Sprintf("%s の結果", name), nil
	case <-ctx.Done():
		return "", fmt.Errorf("%s の呼び出しを中断: %w", name, ctx.Err())
	}
}

func runContext() {
	section("05. context — タイムアウトとキャンセル")

	fmt.Println("■ 300ms で終わる処理に 500ms のタイムアウト")
	show(500*time.Millisecond, 300*time.Millisecond)

	fmt.Println()
	fmt.Println("■ 800ms かかる処理に 500ms のタイムアウト")
	show(500*time.Millisecond, 800*time.Millisecond)

	fmt.Println()
	fmt.Println("タイムアウトしても、呼ばれた側が ctx.Done() を見ていなければ")
	fmt.Println("処理は動き続ける。context は「やめてくれ」という通知でしかない。")
}

func show(timeout, took time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel() // 早く終わっても必ず呼ぶ。呼ばないと context が解放されない

	start := time.Now()
	result, err := callAPI(ctx, "user-service", took)
	elapsed := time.Since(start).Round(10 * time.Millisecond)

	if err != nil {
		fmt.Printf("  %v で失敗: %v\n", elapsed, err)
		fmt.Println("  errors.Is(err, context.DeadlineExceeded) =", errors.Is(err, context.DeadlineExceeded))
		return
	}
	fmt.Printf("  %v で成功: %s\n", elapsed, result)
}
