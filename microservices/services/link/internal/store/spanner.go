// Package store は短縮リンクの保存先を扱う。
package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

// ErrNotFound は該当するキーが無いことを表す。呼び出し側はこれを見て 404 に翻訳する。
var ErrNotFound = errors.New("link not found")

// ErrAlreadyExists はキーが衝突したことを表す。採番をやり直す判断に使う。
var ErrAlreadyExists = errors.New("link already exists")

type Link struct {
	Key        string
	URL        string
	CreateTime time.Time
}

type Spanner struct {
	client *spanner.Client
}

func NewSpanner(ctx context.Context, database string) (*Spanner, error) {
	client, err := spanner.NewClient(ctx, database)
	if err != nil {
		return nil, fmt.Errorf("spanner に接続できません: %w", err)
	}
	return &Spanner{client: client}, nil
}

func (s *Spanner) Close() { s.client.Close() }

func (s *Spanner) Create(ctx context.Context, link Link) error {
	// InsertStruct ではなく Insert を使うのは、commit timestamp を使うため
	mutation := spanner.Insert("links",
		[]string{"key", "url", "created_at"},
		[]any{link.Key, link.URL, spanner.CommitTimestamp},
	)
	if _, err := s.client.Apply(ctx, []*spanner.Mutation{mutation}); err != nil {
		if spanner.ErrCode(err) == codes.AlreadyExists {
			return ErrAlreadyExists
		}
		return fmt.Errorf("insert に失敗しました: %w", err)
	}
	return nil
}

func (s *Spanner) Get(ctx context.Context, key string) (Link, error) {
	row, err := s.client.Single().ReadRow(ctx, "links", spanner.Key{key}, []string{"key", "url", "created_at"})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return Link{}, ErrNotFound
		}
		return Link{}, fmt.Errorf("read に失敗しました: %w", err)
	}
	return scan(row)
}

// List は created_at の降順で limit 件返す。after が空でなければ、その時刻より古いものだけを返す。
func (s *Spanner) List(ctx context.Context, limit int, after time.Time) ([]Link, error) {
	stmt := spanner.Statement{
		SQL: `SELECT key, url, created_at FROM links
		      WHERE (@after IS NULL OR created_at < @after)
		      ORDER BY created_at DESC
		      LIMIT @limit`,
		Params: map[string]any{"limit": int64(limit), "after": nullable(after)},
	}

	iter := s.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	links := make([]Link, 0, limit)
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			return links, nil
		}
		if err != nil {
			return nil, fmt.Errorf("query に失敗しました: %w", err)
		}
		link, err := scan(row)
		if err != nil {
			return nil, err
		}
		links = append(links, link)
	}
}

func scan(row *spanner.Row) (Link, error) {
	var link Link
	if err := row.Columns(&link.Key, &link.URL, &link.CreateTime); err != nil {
		return Link{}, fmt.Errorf("行の読み取りに失敗しました: %w", err)
	}
	return link, nil
}

func nullable(t time.Time) spanner.NullTime {
	return spanner.NullTime{Time: t, Valid: !t.IsZero()}
}
