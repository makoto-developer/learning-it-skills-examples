package store_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/example/microservices/services/link/internal/store"
)

// Spanner に対する結合テスト。
//
// メモリ実装で通っても、Spanner で通るとは限らない（型の制約、commit timestamp、
// クエリの書き方）。そこはエミュレータに対して実際に叩いて確かめる。
//
// エミュレータが無い環境ではスキップする。CI で必ず走らせたい場合は、
// エミュレータをサービスとして起動したうえで SPANNER_EMULATOR_HOST を設定する。
//
//	make emulator && make schema
//	SPANNER_EMULATOR_HOST=localhost:9010 go test ./services/link/internal/store/
func newSpannerStore(t *testing.T) *store.Spanner {
	t.Helper()

	if os.Getenv("SPANNER_EMULATOR_HOST") == "" {
		t.Skip("SPANNER_EMULATOR_HOST が未設定のためスキップします（make emulator && make schema）")
	}

	database := os.Getenv("SPANNER_DATABASE")
	if database == "" {
		database = "projects/learning-project/instances/learning-instance/databases/links"
	}

	ctx := context.Background()
	spannerStore, err := store.NewSpanner(ctx, database)
	if err != nil {
		t.Fatalf("エミュレータに繋がりません: %v", err)
	}
	t.Cleanup(spannerStore.Close)
	return spannerStore
}

// テストごとに固有のキーを使う。既存のデータと衝突させないため。
//
// 時刻を丸めて作ると、続けて走ったテスト同士で衝突してフレーキーになる
// （実際に踏んだ）。乱数にしておけば、実行順にも実行速度にも左右されない。
func uniqueKey(t *testing.T) string {
	t.Helper()
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		t.Fatalf("キーを作れません: %v", err)
	}
	return "t" + hex.EncodeToString(buf) // 13文字。列は STRING(16)
}

func TestSpanner_保存して読み戻せる(t *testing.T) {
	s := newSpannerStore(t)
	ctx := context.Background()
	key := uniqueKey(t)

	if err := s.Create(ctx, store.Link{Key: key, URL: "https://example.com/it"}); err != nil {
		t.Fatalf("作成に失敗しました: %v", err)
	}

	got, err := s.Get(ctx, key)
	if err != nil {
		t.Fatalf("取得に失敗しました: %v", err)
	}
	if got.URL != "https://example.com/it" {
		t.Errorf("url が違います: got=%q", got.URL)
	}
	// created_at はサーバー側（commit timestamp）で入る
	if got.CreateTime.IsZero() {
		t.Error("created_at が入っていません")
	}
}

func TestSpanner_同じキーは二度入らない(t *testing.T) {
	s := newSpannerStore(t)
	ctx := context.Background()
	key := uniqueKey(t)

	if err := s.Create(ctx, store.Link{Key: key, URL: "https://example.com/1"}); err != nil {
		t.Fatalf("1件目で失敗しました: %v", err)
	}

	err := s.Create(ctx, store.Link{Key: key, URL: "https://example.com/2"})
	if !errors.Is(err, store.ErrAlreadyExists) {
		t.Errorf("ErrAlreadyExists になりません: %v", err)
	}
}

func TestSpanner_無いキーはErrNotFound(t *testing.T) {
	s := newSpannerStore(t)

	_, err := s.Get(context.Background(), "no-such-key")
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("ErrNotFound になりません: %v", err)
	}
}

func TestSpanner_一覧は新しい順に返る(t *testing.T) {
	s := newSpannerStore(t)
	ctx := context.Background()

	keys := []string{uniqueKey(t) + "a", uniqueKey(t) + "b"}
	for _, key := range keys {
		if err := s.Create(ctx, store.Link{Key: key, URL: "https://example.com/" + key}); err != nil {
			t.Fatalf("作成に失敗しました: %v", err)
		}
		// commit timestamp を確実にずらす
		time.Sleep(10 * time.Millisecond)
	}

	links, err := s.List(ctx, 100, time.Time{})
	if err != nil {
		t.Fatalf("一覧に失敗しました: %v", err)
	}
	if len(links) < 2 {
		t.Fatalf("件数が足りません: %d", len(links))
	}
	for i := 1; i < len(links); i++ {
		if links[i-1].CreateTime.Before(links[i].CreateTime) {
			t.Fatalf("降順になっていません: %v の後に %v", links[i-1].CreateTime, links[i].CreateTime)
		}
	}
}

func TestSpanner_afterより古いものだけ返る(t *testing.T) {
	s := newSpannerStore(t)
	ctx := context.Background()

	all, err := s.List(ctx, 100, time.Time{})
	if err != nil {
		t.Fatalf("一覧に失敗しました: %v", err)
	}
	if len(all) < 2 {
		t.Skip("比較できるだけの件数がありません")
	}

	// 先頭（最も新しい）の時刻を境にすると、それより古いものだけが返る
	after := all[0].CreateTime
	got, err := s.List(ctx, 100, after)
	if err != nil {
		t.Fatalf("一覧に失敗しました: %v", err)
	}
	for _, link := range got {
		if !link.CreateTime.Before(after) {
			t.Errorf("境界より新しい行が混ざっています: %v >= %v", link.CreateTime, after)
		}
	}
}
