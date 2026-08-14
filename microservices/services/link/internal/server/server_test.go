package server_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	linkv1 "github.com/example/microservices/gen/link/v1"
	"github.com/example/microservices/services/link/internal/server"
	"github.com/example/microservices/services/link/internal/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var baseTime = time.Date(2026, 4, 1, 9, 0, 0, 0, time.UTC)

// newTestServer は、キーと時刻を固定した Server を返す。
// 乱数と現在時刻が入ると、期待値を書けなくなる。
func newTestServer(t *testing.T, keys ...string) *server.Server {
	t.Helper()

	tick := 0
	memory := store.NewMemory(func() time.Time {
		tick++
		// 1秒ずつ進める。作成順が確定するので一覧の期待値が書ける
		return baseTime.Add(time.Duration(tick) * time.Second)
	})

	index := 0
	return server.New(server.Config{
		Store: memory,
		NewKey: func() (string, error) {
			if len(keys) == 0 {
				index++
				return fmt.Sprintf("key%03d", index), nil
			}
			// 渡されたキーを順に返し、尽きたら最後のものを返し続ける
			key := keys[min(index, len(keys)-1)]
			index++
			return key, nil
		},
	})
}

func TestCreateLink(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		wantCode codes.Code
	}{
		{name: "https を受け付ける", url: "https://example.com/a", wantCode: codes.OK},
		{name: "http も受け付ける", url: "http://example.com", wantCode: codes.OK},
		{name: "クエリ付きも受け付ける", url: "https://example.com/a?b=1#c", wantCode: codes.OK},
		{name: "空文字は弾く", url: "", wantCode: codes.InvalidArgument},
		{name: "スキームなしは弾く", url: "example.com", wantCode: codes.InvalidArgument},
		{name: "javascript は弾く", url: "javascript:alert(1)", wantCode: codes.InvalidArgument},
		{name: "file は弾く", url: "file:///etc/passwd", wantCode: codes.InvalidArgument},
		// ホストが付いていると「ホスト無し」の判定をすり抜ける。スキームの検証が要る
		{name: "ホスト付きの javascript も弾く", url: "javascript://example.com/%0aalert(1)", wantCode: codes.InvalidArgument},
		{name: "data スキームも弾く", url: "data://example.com/,hello", wantCode: codes.InvalidArgument},
		{name: "ホストなしは弾く", url: "https://", wantCode: codes.InvalidArgument},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestServer(t)

			got, err := svc.CreateLink(context.Background(), &linkv1.CreateLinkRequest{Url: tt.url})

			if status.Code(err) != tt.wantCode {
				t.Fatalf("コードが違います: got=%v want=%v (err=%v)", status.Code(err), tt.wantCode, err)
			}
			if tt.wantCode != codes.OK {
				return
			}
			if got.GetLink().GetUrl() != tt.url {
				t.Errorf("url が違います: got=%q want=%q", got.GetLink().GetUrl(), tt.url)
			}
			if got.GetLink().GetKey() == "" {
				t.Error("key が空です")
			}
			// 時刻はサーバー側で決まる。クライアントは渡していない
			if !got.GetLink().GetCreateTime().IsValid() {
				t.Error("create_time が入っていません")
			}
		})
	}
}

func TestCreateLink_キーが衝突したら採番し直す(t *testing.T) {
	ctx := context.Background()
	// 1回目と2回目で同じキーを返し、3回目に別のキーを返す
	svc := newTestServer(t, "dup", "dup", "fresh")

	first, err := svc.CreateLink(ctx, &linkv1.CreateLinkRequest{Url: "https://example.com/1"})
	if err != nil {
		t.Fatalf("1件目で失敗しました: %v", err)
	}
	if first.GetLink().GetKey() != "dup" {
		t.Fatalf("1件目のキーが違います: got=%q", first.GetLink().GetKey())
	}

	second, err := svc.CreateLink(ctx, &linkv1.CreateLinkRequest{Url: "https://example.com/2"})
	if err != nil {
		t.Fatalf("衝突したまま失敗しました: %v", err)
	}
	if second.GetLink().GetKey() != "fresh" {
		t.Errorf("採番し直せていません: got=%q want=%q", second.GetLink().GetKey(), "fresh")
	}
	// 先に入っていた行が上書きされていないこと
	if got, _ := svc.GetLink(ctx, &linkv1.GetLinkRequest{Key: "dup"}); got.GetLink().GetUrl() != "https://example.com/1" {
		t.Errorf("既存の行が壊れました: got=%q", got.GetLink().GetUrl())
	}
}

func TestCreateLink_衝突し続けたら諦める(t *testing.T) {
	ctx := context.Background()
	// 常に同じキーしか返さない採番
	svc := newTestServer(t, "always-same")

	if _, err := svc.CreateLink(ctx, &linkv1.CreateLinkRequest{Url: "https://example.com/1"}); err != nil {
		t.Fatalf("1件目で失敗しました: %v", err)
	}

	_, err := svc.CreateLink(ctx, &linkv1.CreateLinkRequest{Url: "https://example.com/2"})
	if status.Code(err) != codes.ResourceExhausted {
		t.Errorf("コードが違います: got=%v want=%v", status.Code(err), codes.ResourceExhausted)
	}
}

func TestCreateLink_採番の失敗はInternal(t *testing.T) {
	svc := server.New(server.Config{
		Store:  store.NewMemory(nil),
		NewKey: func() (string, error) { return "", errors.New("乱数が読めません") },
	})

	_, err := svc.CreateLink(context.Background(), &linkv1.CreateLinkRequest{Url: "https://example.com"})
	if status.Code(err) != codes.Internal {
		t.Errorf("コードが違います: got=%v want=%v", status.Code(err), codes.Internal)
	}
}

func TestGetLink(t *testing.T) {
	ctx := context.Background()
	svc := newTestServer(t, "abc")
	if _, err := svc.CreateLink(ctx, &linkv1.CreateLinkRequest{Url: "https://example.com/x"}); err != nil {
		t.Fatalf("前提の作成に失敗しました: %v", err)
	}

	tests := []struct {
		name     string
		key      string
		wantCode codes.Code
		wantURL  string
	}{
		{name: "あるキーは引ける", key: "abc", wantCode: codes.OK, wantURL: "https://example.com/x"},
		{name: "無いキーは NotFound", key: "nosuch", wantCode: codes.NotFound},
		{name: "空のキーは InvalidArgument", key: "", wantCode: codes.InvalidArgument},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := svc.GetLink(ctx, &linkv1.GetLinkRequest{Key: tt.key})

			if status.Code(err) != tt.wantCode {
				t.Fatalf("コードが違います: got=%v want=%v", status.Code(err), tt.wantCode)
			}
			if tt.wantCode == codes.OK && got.GetLink().GetUrl() != tt.wantURL {
				t.Errorf("url が違います: got=%q want=%q", got.GetLink().GetUrl(), tt.wantURL)
			}
		})
	}
}

func TestListLinks_ページサイズ(t *testing.T) {
	ctx := context.Background()
	svc := newTestServer(t)
	for i := range 5 {
		if _, err := svc.CreateLink(ctx, &linkv1.CreateLinkRequest{Url: fmt.Sprintf("https://example.com/%d", i)}); err != nil {
			t.Fatalf("前提の作成に失敗しました: %v", err)
		}
	}

	tests := []struct {
		name     string
		pageSize int32
		want     int
	}{
		{name: "指定した件数だけ返る", pageSize: 2, want: 2},
		{name: "0 なら既定値が使われる", pageSize: 0, want: 5},
		{name: "負の値でも既定値が使われる", pageSize: -1, want: 5},
		{name: "上限を超えても壊れない", pageSize: 1000, want: 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := svc.ListLinks(ctx, &linkv1.ListLinksRequest{PageSize: tt.pageSize})
			if err != nil {
				t.Fatalf("失敗しました: %v", err)
			}
			if len(got.GetLinks()) != tt.want {
				t.Errorf("件数が違います: got=%d want=%d", len(got.GetLinks()), tt.want)
			}
		})
	}
}

func TestListLinks_ページをたどると全件が重複なく取れる(t *testing.T) {
	ctx := context.Background()
	svc := newTestServer(t)
	const total = 7
	for i := range total {
		if _, err := svc.CreateLink(ctx, &linkv1.CreateLinkRequest{Url: fmt.Sprintf("https://example.com/%d", i)}); err != nil {
			t.Fatalf("前提の作成に失敗しました: %v", err)
		}
	}

	seen := map[string]bool{}
	token := ""
	for page := 0; ; page++ {
		if page > total {
			t.Fatal("ページが終わりません。next_page_token が進んでいない可能性があります")
		}
		got, err := svc.ListLinks(ctx, &linkv1.ListLinksRequest{PageSize: 3, PageToken: token})
		if err != nil {
			t.Fatalf("%d ページ目で失敗しました: %v", page, err)
		}
		for _, link := range got.GetLinks() {
			if seen[link.GetKey()] {
				t.Errorf("同じキーが2回出ました: %s", link.GetKey())
			}
			seen[link.GetKey()] = true
		}
		token = got.GetNextPageToken()
		if token == "" {
			break
		}
	}

	if len(seen) != total {
		t.Errorf("全件取れていません: got=%d want=%d", len(seen), total)
	}
}

func TestListLinks_壊れたトークンはInvalidArgument(t *testing.T) {
	svc := newTestServer(t)

	_, err := svc.ListLinks(context.Background(), &linkv1.ListLinksRequest{PageToken: "!!!not-base64!!!"})
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("コードが違います: got=%v want=%v", status.Code(err), codes.InvalidArgument)
	}
}

// 保存先が落ちている状況を作り、Internal に変換されることを確かめる。
func TestListLinks_保存先の障害はInternal(t *testing.T) {
	svc := server.New(server.Config{Store: failingStore{}})

	_, err := svc.ListLinks(context.Background(), &linkv1.ListLinksRequest{})
	if status.Code(err) != codes.Internal {
		t.Errorf("コードが違います: got=%v want=%v", status.Code(err), codes.Internal)
	}
}

type failingStore struct{}

var errBroken = errors.New("保存先に繋がりません")

func (failingStore) Create(context.Context, store.Link) error { return errBroken }
func (failingStore) Get(context.Context, string) (store.Link, error) {
	return store.Link{}, errBroken
}
func (failingStore) List(context.Context, int, time.Time) ([]store.Link, error) {
	return nil, errBroken
}

// ページサイズの上限は、返る件数だけを見ても確かめられない
//（保存件数が少ないと、クランプしてもしなくても同じ数になる）。
// そこで、保存先が受け取った limit を記録して直接確かめる。
type spyStore struct {
	store.Link
	gotLimit int
}

func (s *spyStore) Create(context.Context, store.Link) error { return nil }
func (s *spyStore) Get(context.Context, string) (store.Link, error) {
	return store.Link{}, store.ErrNotFound
}
func (s *spyStore) List(_ context.Context, limit int, _ time.Time) ([]store.Link, error) {
	s.gotLimit = limit
	return nil, nil
}

func TestListLinks_ページサイズは上限と既定値に丸められる(t *testing.T) {
	tests := []struct {
		name      string
		pageSize  int32
		wantLimit int // サーバーは「次ページの有無」を見るため +1 して問い合わせる
	}{
		{name: "指定した値がそのまま渡る", pageSize: 5, wantLimit: 6},
		{name: "0 は既定値の 20 になる", pageSize: 0, wantLimit: 21},
		{name: "負の値も既定値になる", pageSize: -1, wantLimit: 21},
		{name: "上限の 100 に丸められる", pageSize: 1000, wantLimit: 101},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spy := &spyStore{}
			svc := server.New(server.Config{Store: spy})

			if _, err := svc.ListLinks(context.Background(), &linkv1.ListLinksRequest{PageSize: tt.pageSize}); err != nil {
				t.Fatalf("失敗しました: %v", err)
			}
			if spy.gotLimit != tt.wantLimit {
				t.Errorf("保存先へ渡した limit が違います: got=%d want=%d", spy.gotLimit, tt.wantLimit)
			}
		})
	}
}
