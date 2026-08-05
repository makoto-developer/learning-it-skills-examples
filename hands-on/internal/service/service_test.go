package service_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/example/linkshort/internal/service"
	"github.com/example/linkshort/internal/store"
)

var fixedTime = time.Date(2026, 4, 1, 9, 0, 0, 0, time.UTC)

// newTestService は毎回まっさらな Service を返す。
// キーと時刻を固定するので、テストの期待値に乱数と現在時刻が混ざらない。
func newTestService(t *testing.T) (*service.Service, store.Store) {
	t.Helper()

	memory := store.NewMemory()
	counter := 0
	svc := service.New(service.Config{
		Store: memory,
		NewKey: func() (string, error) {
			counter++
			return fmt.Sprintf("key%03d", counter), nil
		},
		Now:    func() time.Time { return fixedTime },
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	return svc, memory
}

func TestCreateLink(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantURL string
		wantErr error
	}{
		{name: "https を受け付ける", url: "https://example.com/a", wantURL: "https://example.com/a"},
		{name: "http も受け付ける", url: "http://example.com", wantURL: "http://example.com"},
		{name: "前後の空白を落とす", url: "  https://example.com/b  ", wantURL: "https://example.com/b"},
		{name: "空文字は弾く", url: "", wantErr: service.ErrInvalidArgument},
		{name: "空白だけも弾く", url: "   ", wantErr: service.ErrInvalidArgument},
		{name: "スキームなしは弾く", url: "example.com", wantErr: service.ErrInvalidArgument},
		{name: "javascript は弾く", url: "javascript:alert(1)", wantErr: service.ErrInvalidArgument},
		{name: "ホストなしは弾く", url: "https://", wantErr: service.ErrInvalidArgument},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _ := newTestService(t)

			got, err := svc.CreateLink(context.Background(), tt.url)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}
			if got.URL != tt.wantURL {
				t.Errorf("URL = %q, want %q", got.URL, tt.wantURL)
			}
			if got.Key == "" {
				t.Error("Key が空です")
			}
			if !got.CreatedAt.Equal(fixedTime) {
				t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, fixedTime)
			}
		})
	}
}

func TestGetLink(t *testing.T) {
	svc, _ := newTestService(t)
	created, err := svc.CreateLink(context.Background(), "https://example.com/x")
	if err != nil {
		t.Fatalf("下準備の CreateLink が失敗しました: %v", err)
	}

	tests := []struct {
		name    string
		key     string
		wantURL string
		wantErr error
	}{
		{name: "作ったキーは引ける", key: created.Key, wantURL: "https://example.com/x"},
		{name: "存在しないキー", key: "nosuchkey", wantErr: service.ErrNotFound},
		{name: "空のキー", key: "", wantErr: service.ErrInvalidArgument},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := svc.GetLink(context.Background(), tt.key)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && got.URL != tt.wantURL {
				t.Errorf("URL = %q, want %q", got.URL, tt.wantURL)
			}
		})
	}
}

func TestListLinksPagination(t *testing.T) {
	tests := []struct {
		name      string
		total     int
		pageSize  int
		wantCount int
		wantMore  bool
	}{
		{name: "0件", total: 0, pageSize: 10, wantCount: 0, wantMore: false},
		{name: "ページより少ない", total: 3, pageSize: 10, wantCount: 3, wantMore: false},
		{name: "ちょうど1ページ", total: 10, pageSize: 10, wantCount: 10, wantMore: false},
		{name: "1件あふれる", total: 11, pageSize: 10, wantCount: 10, wantMore: true},
		{name: "page_size 未指定は既定値", total: 25, pageSize: 0, wantCount: 20, wantMore: true},
		{name: "上限を超えたら丸める", total: 5, pageSize: 1000, wantCount: 5, wantMore: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _ := newTestService(t)
			seedLinks(t, svc, tt.total)

			got, err := svc.ListLinks(context.Background(), service.ListLinksRequest{PageSize: tt.pageSize})
			if err != nil {
				t.Fatalf("ListLinks: %v", err)
			}

			if len(got.Links) != tt.wantCount {
				t.Errorf("件数 = %d, want %d", len(got.Links), tt.wantCount)
			}
			if hasMore := got.NextPageToken != ""; hasMore != tt.wantMore {
				t.Errorf("NextPageToken の有無 = %v, want %v", hasMore, tt.wantMore)
			}
		})
	}
}

// ページを最後までたどると、全件がちょうど1回ずつ出てくることを確かめる。
func TestListLinksWalksEveryPageOnce(t *testing.T) {
	svc, _ := newTestService(t)
	const total = 23
	seedLinks(t, svc, total)

	seen := make(map[string]int)
	token := ""
	for page := 0; ; page++ {
		if page > total {
			t.Fatal("ページ送りが終わりません")
		}
		got, err := svc.ListLinks(context.Background(), service.ListLinksRequest{PageSize: 5, PageToken: token})
		if err != nil {
			t.Fatalf("ListLinks: %v", err)
		}
		for _, link := range got.Links {
			seen[link.Key]++
		}
		if got.NextPageToken == "" {
			break
		}
		token = got.NextPageToken
	}

	if len(seen) != total {
		t.Errorf("見えた件数 = %d, want %d", len(seen), total)
	}
	for key, count := range seen {
		if count != 1 {
			t.Errorf("key %s が %d 回出ました", key, count)
		}
	}
}

func TestListLinksRejectsBadInput(t *testing.T) {
	tests := []struct {
		name string
		req  service.ListLinksRequest
	}{
		{name: "負の page_size", req: service.ListLinksRequest{PageSize: -1}},
		{name: "壊れた page_token", req: service.ListLinksRequest{PageToken: "!!!not-base64!!!"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _ := newTestService(t)

			_, err := svc.ListLinks(context.Background(), tt.req)

			if !errors.Is(err, service.ErrInvalidArgument) {
				t.Fatalf("err = %v, want ErrInvalidArgument", err)
			}
		})
	}
}

// failingStore は「保存先が落ちた」状況を作るための差し替え実装。
// インターフェースで切ってあるので、DB を落とさなくても異常系を再現できる。
type failingStore struct {
	store.Store
	err error
}

func (f failingStore) Create(context.Context, store.Link) error { return f.err }

func TestCreateLinkPropagatesStoreFailure(t *testing.T) {
	boom := errors.New("disk on fire")
	svc := service.New(service.Config{
		Store:  failingStore{Store: store.NewMemory(), err: boom},
		NewKey: func() (string, error) { return "fixedkey", nil },
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	_, err := svc.CreateLink(context.Background(), "https://example.com")

	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want %v をラップした値", err, boom)
	}
}

// キーが常に衝突する状況では、無限に採番し直さず打ち切ることを確かめる。
func TestCreateLinkGivesUpOnRepeatedCollision(t *testing.T) {
	svc := service.New(service.Config{
		Store:  failingStore{Store: store.NewMemory(), err: store.ErrConflict},
		NewKey: func() (string, error) { return "always-same", nil },
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	_, err := svc.CreateLink(context.Background(), "https://example.com")

	if !errors.Is(err, service.ErrExhausted) {
		t.Fatalf("err = %v, want ErrExhausted", err)
	}
}

func seedLinks(t *testing.T, svc *service.Service, count int) {
	t.Helper()
	for i := range count {
		if _, err := svc.CreateLink(context.Background(), fmt.Sprintf("https://example.com/%d", i)); err != nil {
			t.Fatalf("下準備の CreateLink(%d) が失敗しました: %v", i, err)
		}
	}
}
