package store

import (
	"context"
	"sort"
	"sync"
	"time"
)

// Memory はテストと手元での試し用の保存先。
// 本番では Spanner を使うが、同じインターフェースを満たすので差し替えられる。
type Memory struct {
	mu    sync.RWMutex
	links map[string]Link
	// 時刻を固定できるようにしておく。テストの期待値に現在時刻が混ざらない
	now func() time.Time
}

func NewMemory(now func() time.Time) *Memory {
	if now == nil {
		now = time.Now
	}
	return &Memory{links: make(map[string]Link), now: now}
}

func (m *Memory) Create(_ context.Context, link Link) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.links[link.Key]; exists {
		return ErrAlreadyExists
	}
	link.CreateTime = m.now()
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

func (m *Memory) List(_ context.Context, limit int, after time.Time) ([]Link, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	all := make([]Link, 0, len(m.links))
	for _, link := range m.links {
		if !after.IsZero() && !link.CreateTime.Before(after) {
			continue
		}
		all = append(all, link)
	}

	// Spanner 側と同じ「作成日時の降順」に揃える。
	// 同時刻はキーで並べる。順序が決まらないとテストが不安定になる
	sort.Slice(all, func(i, j int) bool {
		if all[i].CreateTime.Equal(all[j].CreateTime) {
			return all[i].Key < all[j].Key
		}
		return all[i].CreateTime.After(all[j].CreateTime)
	})

	if len(all) > limit {
		all = all[:limit]
	}
	return all, nil
}
