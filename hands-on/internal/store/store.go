// Package store は短縮リンクの保存先を表す。
package store

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"
)

var (
	ErrNotFound = errors.New("store: link not found")
	ErrConflict = errors.New("store: key already exists")
)

// Link は保存される1件分のデータ。
type Link struct {
	Key       string
	URL       string
	CreatedAt time.Time
}

// Store は保存先の口。呼び出し側はこの口だけに依存するので、
// インメモリを PostgreSQL に差し替えても service は変わらない。
type Store interface {
	Create(ctx context.Context, link Link) error
	Get(ctx context.Context, key string) (Link, error)
	// List は after より後ろのキーを昇順で最大 limit 件返す。
	List(ctx context.Context, limit int, after string) ([]Link, error)
}

// Memory はプロセス内のみで完結する実装。再起動すると全部消える。
type Memory struct {
	mu    sync.RWMutex
	links map[string]Link
}

func NewMemory() *Memory {
	return &Memory{links: make(map[string]Link)}
}

func (m *Memory) Create(_ context.Context, link Link) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.links[link.Key]; exists {
		return ErrConflict
	}
	m.links[link.Key] = link
	return nil
}

func (m *Memory) Get(_ context.Context, key string) (Link, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	link, ok := m.links[key]
	if !ok {
		return Link{}, ErrNotFound
	}
	return link, nil
}

func (m *Memory) List(_ context.Context, limit int, after string) ([]Link, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// キーの全順序で区切ることで、途中で件数が増減してもページ境界がずれない
	keys := make([]string, 0, len(m.links))
	for key := range m.links {
		if key > after {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	if len(keys) > limit {
		keys = keys[:limit]
	}

	links := make([]Link, 0, len(keys))
	for _, key := range keys {
		links = append(links, m.links[key])
	}
	return links, nil
}
