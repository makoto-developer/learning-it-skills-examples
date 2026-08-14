// Package server は LinkService の gRPC 実装。
package server

import (
	"context"
	"crypto/rand"
	"errors"
	"log/slog"
	"net/url"
	"time"

	linkv1 "github.com/example/microservices/gen/link/v1"
	"github.com/example/microservices/services/link/internal/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	keyLength       = 7
	defaultPageSize = 20
	maxPageSize     = 100
	// 衝突しても諦めずに採番し直す回数
	createRetries = 5
)

// 紛らわしい文字（0/O/1/l）を外した英数字
const keyAlphabet = "abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"

type Store interface {
	Create(ctx context.Context, link store.Link) error
	Get(ctx context.Context, key string) (store.Link, error)
	List(ctx context.Context, limit int, after time.Time) ([]store.Link, error)
}

type Server struct {
	linkv1.UnimplementedLinkServiceServer
	store Store
}

func New(s Store) *Server { return &Server{store: s} }

func (s *Server) CreateLink(ctx context.Context, req *linkv1.CreateLinkRequest) (*linkv1.CreateLinkResponse, error) {
	if err := validateURL(req.GetUrl()); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	for attempt := range createRetries {
		key, err := newKey()
		if err != nil {
			return nil, status.Error(codes.Internal, "キーを採番できませんでした")
		}

		err = s.store.Create(ctx, store.Link{Key: key, URL: req.GetUrl()})
		switch {
		case err == nil:
			// 保存した行を読み直す。create_time はサーバー側で決まるため
			saved, getErr := s.store.Get(ctx, key)
			if getErr != nil {
				return nil, status.Error(codes.Internal, "保存後の読み取りに失敗しました")
			}
			slog.InfoContext(ctx, "link created", "key", key)
			return &linkv1.CreateLinkResponse{Link: toProto(saved)}, nil
		case errors.Is(err, store.ErrAlreadyExists):
			slog.WarnContext(ctx, "key collision", "key", key, "attempt", attempt+1)
			continue
		default:
			slog.ErrorContext(ctx, "create failed", "error", err)
			return nil, status.Error(codes.Internal, "保存に失敗しました")
		}
	}
	return nil, status.Error(codes.ResourceExhausted, "キーを採番できませんでした。時間をおいて再試行してください")
}

func (s *Server) GetLink(ctx context.Context, req *linkv1.GetLinkRequest) (*linkv1.GetLinkResponse, error) {
	if req.GetKey() == "" {
		return nil, status.Error(codes.InvalidArgument, "key は必須です")
	}

	link, err := s.store.Get(ctx, req.GetKey())
	if errors.Is(err, store.ErrNotFound) {
		return nil, status.Error(codes.NotFound, "そのキーは存在しません")
	}
	if err != nil {
		slog.ErrorContext(ctx, "get failed", "error", err)
		return nil, status.Error(codes.Internal, "取得に失敗しました")
	}
	return &linkv1.GetLinkResponse{Link: toProto(link)}, nil
}

func (s *Server) ListLinks(ctx context.Context, req *linkv1.ListLinksRequest) (*linkv1.ListLinksResponse, error) {
	size := int(req.GetPageSize())
	switch {
	case size <= 0:
		size = defaultPageSize
	case size > maxPageSize:
		size = maxPageSize
	}

	after, err := decodeToken(req.GetPageToken())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "page_token が不正です")
	}

	// 次ページの有無を知るために1件多く取る
	links, err := s.store.List(ctx, size+1, after)
	if err != nil {
		slog.ErrorContext(ctx, "list failed", "error", err)
		return nil, status.Error(codes.Internal, "一覧の取得に失敗しました")
	}

	next := ""
	if len(links) > size {
		links = links[:size]
		next = encodeToken(links[len(links)-1].CreateTime)
	}

	out := make([]*linkv1.Link, 0, len(links))
	for _, link := range links {
		out = append(out, toProto(link))
	}
	return &linkv1.ListLinksResponse{Links: out, NextPageToken: next}, nil
}

func validateURL(raw string) error {
	if raw == "" {
		return errors.New("url は必須です")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return errors.New("url の形式が不正です")
	}
	// スキームを絞らないと javascript: を保存できてしまう
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("url は http または https で始めてください")
	}
	if parsed.Host == "" {
		return errors.New("url にホスト名がありません")
	}
	return nil
}

func newKey() (string, error) {
	buf := make([]byte, keyLength)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	key := make([]byte, keyLength)
	for i, b := range buf {
		key[i] = keyAlphabet[int(b)%len(keyAlphabet)]
	}
	return string(key), nil
}

func toProto(link store.Link) *linkv1.Link {
	return &linkv1.Link{
		Key:        link.Key,
		Url:        link.URL,
		CreateTime: timestamppb.New(link.CreateTime),
	}
}
