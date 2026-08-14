package server

import (
	"testing"
	"time"
)

// ページトークンは外に出ない実装の詳細なので、パッケージ内からテストする。
func Testトークンは往復しても同じ時刻になる(t *testing.T) {
	tests := []struct {
		name string
		in   time.Time
	}{
		{name: "UTC", in: time.Date(2026, 4, 1, 9, 0, 0, 0, time.UTC)},
		{name: "ナノ秒まで保つ", in: time.Date(2026, 4, 1, 9, 0, 0, 123456789, time.UTC)},
		{name: "JST は UTC に正規化される", in: time.Date(2026, 4, 1, 18, 0, 0, 0, time.FixedZone("JST", 9*60*60))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeToken(encodeToken(tt.in))
			if err != nil {
				t.Fatalf("復号に失敗しました: %v", err)
			}
			if !got.Equal(tt.in) {
				t.Errorf("時刻が変わりました: got=%v want=%v", got, tt.in)
			}
		})
	}
}

func Test空のトークンはゼロ値になる(t *testing.T) {
	got, err := decodeToken("")
	if err != nil {
		t.Fatalf("エラーになりました: %v", err)
	}
	if !got.IsZero() {
		t.Errorf("ゼロ値ではありません: %v", got)
	}
}

func Test壊れたトークンはエラーになる(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{name: "base64 ではない", token: "!!!"},
		{name: "base64 だが時刻ではない", token: "aGVsbG8"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := decodeToken(tt.token); err == nil {
				t.Error("エラーになりませんでした")
			}
		})
	}
}

func Testトークンに中身が透けて見えない(t *testing.T) {
	token := encodeToken(time.Date(2026, 4, 1, 9, 0, 0, 0, time.UTC))

	// 見た目から日付を推測できると、利用者が中身に依存し始める
	for _, fragment := range []string{"2026", "04-01", ":"} {
		if contains(token, fragment) {
			t.Errorf("トークンに %q がそのまま出ています: %s", fragment, token)
		}
	}
}

func contains(s, sub string) bool {
	return len(sub) > 0 && len(s) >= len(sub) && indexOf(s, sub) >= 0
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
