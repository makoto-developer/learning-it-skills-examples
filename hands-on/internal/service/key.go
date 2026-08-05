package service

import (
	"crypto/rand"
	"math/big"
)

const (
	keyAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	keyLength   = 7
)

// RandomKey は推測されにくい短縮キーを作る。
// 連番だと他人のリンクを総当たりで列挙できてしまうため、暗号乱数を使う。
func RandomKey() (string, error) {
	limit := big.NewInt(int64(len(keyAlphabet)))
	key := make([]byte, keyLength)
	for i := range key {
		n, err := rand.Int(rand.Reader, limit)
		if err != nil {
			return "", err
		}
		key[i] = keyAlphabet[n.Int64()]
	}
	return string(key), nil
}
