package server

import (
	"encoding/base64"
	"fmt"
	"time"
)

// ページトークンは「次にどこから読むか」を表すだけの不透明な文字列。
// 中身を約束すると変更できなくなるので、base64 にして外からは読めない体裁にしておく。
func encodeToken(t time.Time) string {
	return base64.RawURLEncoding.EncodeToString([]byte(t.UTC().Format(time.RFC3339Nano)))
}

func decodeToken(token string) (time.Time, error) {
	if token == "" {
		return time.Time{}, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return time.Time{}, fmt.Errorf("token をデコードできません: %w", err)
	}
	t, err := time.Parse(time.RFC3339Nano, string(raw))
	if err != nil {
		return time.Time{}, fmt.Errorf("token の中身が時刻ではありません: %w", err)
	}
	return t, nil
}
