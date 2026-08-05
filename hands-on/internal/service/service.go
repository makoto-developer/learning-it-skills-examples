// Package service は短縮 URL サービスの業務ロジックを持つ。
// HTTP にも gRPC にも依存しないので、入口を差し替えてもこの層は動く。
package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/example/linkshort/internal/store"
)

const (
	defaultPageSize = 20
	maxPageSize     = 100
	createMaxRetry  = 3
	maxURLLength    = 2048
)

var (
	ErrInvalidArgument = errors.New("invalid argument")
	ErrNotFound        = errors.New("link not found")
	ErrExhausted       = errors.New("could not allocate a key")
)

// Link は外に出す1件分の表現。store.Link とは別に持つことで、
// 保存先の都合(内部 ID など)が API に漏れないようにする。
type Link struct {
	Key       string    `json:"key"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
}

type ListLinksRequest struct {
	PageSize  int
	PageToken string
}

type ListLinksResponse struct {
	Links         []Link `json:"links"`
	NextPageToken string `json:"next_page_token"`
}

// Config は Service の依存をまとめる。引数を増やさずに差し替えられるようにする。
type Config struct {
	Store  store.Store
	NewKey func() (string, error)
	Now    func() time.Time
	Logger *slog.Logger
}

type Service struct {
	store  store.Store
	newKey func() (string, error)
	now    func() time.Time
	logger *slog.Logger
}

func New(cfg Config) *Service {
	svc := &Service{
		store:  cfg.Store,
		newKey: cfg.NewKey,
		now:    cfg.Now,
		logger: cfg.Logger,
	}
	if svc.newKey == nil {
		svc.newKey = RandomKey
	}
	if svc.now == nil {
		svc.now = time.Now
	}
	if svc.logger == nil {
		svc.logger = slog.Default()
	}
	return svc
}

func (s *Service) CreateLink(ctx context.Context, rawURL string) (Link, error) {
	target, err := normalizeURL(rawURL)
	if err != nil {
		return Link{}, err
	}

	created, err := s.storeWithNewKey(ctx, target)
	if err != nil {
		return Link{}, err
	}

	s.logger.InfoContext(ctx, "link created", "key", created.Key, "url", created.URL)
	return toLink(created), nil
}

// storeWithNewKey はキー衝突のときだけ採番からやり直す。
func (s *Service) storeWithNewKey(ctx context.Context, target string) (store.Link, error) {
	for attempt := range createMaxRetry {
		key, err := s.newKey()
		if err != nil {
			return store.Link{}, fmt.Errorf("generate key: %w", err)
		}

		record := store.Link{Key: key, URL: target, CreatedAt: s.now().UTC()}
		switch err := s.store.Create(ctx, record); {
		case err == nil:
			return record, nil
		case errors.Is(err, store.ErrConflict):
			s.logger.WarnContext(ctx, "key collision", "key", key, "attempt", attempt+1)
		default:
			return store.Link{}, fmt.Errorf("create link: %w", err)
		}
	}
	return store.Link{}, ErrExhausted
}

func (s *Service) GetLink(ctx context.Context, key string) (Link, error) {
	if key == "" {
		return Link{}, fmt.Errorf("%w: key is required", ErrInvalidArgument)
	}

	record, err := s.store.Get(ctx, key)
	if errors.Is(err, store.ErrNotFound) {
		return Link{}, ErrNotFound
	}
	if err != nil {
		return Link{}, fmt.Errorf("get link: %w", err)
	}
	return toLink(record), nil
}

func (s *Service) ListLinks(ctx context.Context, req ListLinksRequest) (ListLinksResponse, error) {
	size, err := normalizePageSize(req.PageSize)
	if err != nil {
		return ListLinksResponse{}, err
	}
	after, err := decodePageToken(req.PageToken)
	if err != nil {
		return ListLinksResponse{}, err
	}

	// 次ページの有無を知るために1件多く取る
	records, err := s.store.List(ctx, size+1, after)
	if err != nil {
		return ListLinksResponse{}, fmt.Errorf("list links: %w", err)
	}

	next := ""
	if len(records) > size {
		next = encodePageToken(records[size-1].Key)
		records = records[:size]
	}

	links := make([]Link, 0, len(records))
	for _, record := range records {
		links = append(links, toLink(record))
	}
	return ListLinksResponse{Links: links, NextPageToken: next}, nil
}

func toLink(record store.Link) Link {
	return Link{Key: record.Key, URL: record.URL, CreatedAt: record.CreatedAt}
}

func normalizePageSize(size int) (int, error) {
	switch {
	case size < 0:
		return 0, fmt.Errorf("%w: page_size must not be negative", ErrInvalidArgument)
	case size == 0:
		return defaultPageSize, nil
	case size > maxPageSize:
		return maxPageSize, nil
	default:
		return size, nil
	}
}

func encodePageToken(key string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(key))
}

// 不透明なトークンにしておくと、あとで並び順を変えても互換性を壊さずに済む。
func decodePageToken(token string) (string, error) {
	if token == "" {
		return "", nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return "", fmt.Errorf("%w: malformed page_token", ErrInvalidArgument)
	}
	return string(decoded), nil
}

func normalizeURL(rawURL string) (string, error) {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return "", fmt.Errorf("%w: url is required", ErrInvalidArgument)
	}
	if len(trimmed) > maxURLLength {
		return "", fmt.Errorf("%w: url is too long", ErrInvalidArgument)
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("%w: url is not parsable", ErrInvalidArgument)
	}
	// javascript: や file: を通すと、転送先がそのまま攻撃面になる
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("%w: url must be http or https", ErrInvalidArgument)
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("%w: url must have a host", ErrInvalidArgument)
	}
	return parsed.String(), nil
}
