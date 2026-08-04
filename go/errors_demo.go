package main

import (
	"errors"
	"fmt"
)

var errUserNotFound = errors.New("ユーザーが見つかりません")

// エラーに文脈を持たせたい時は型にする。errors.As で中身を取り出せる
type validationError struct {
	Field  string
	Reason string
}

func (e *validationError) Error() string {
	return fmt.Sprintf("%s が不正です: %s", e.Field, e.Reason)
}

func fetchUser(id string) error {
	if id == "" {
		return &validationError{Field: "id", Reason: "空文字"}
	}
	if id == "999" {
		return errUserNotFound
	}
	return nil
}

// %v は文字列に潰す。呼び出し元は「元が何のエラーか」を判定できなくなる
func loadProfileWithV(id string) error {
	if err := fetchUser(id); err != nil {
		return fmt.Errorf("プロフィールの取得に失敗: %v", err)
	}
	return nil
}

// %w は元のエラーを保持する。errors.Is / errors.As で辿れる
func loadProfileWithW(id string) error {
	if err := fetchUser(id); err != nil {
		return fmt.Errorf("プロフィールの取得に失敗: %w", err)
	}
	return nil
}

func runErrors() {
	section("01. error のラップと errors.Is / errors.As")

	// go vet が Println 内の書式指定子を誤検知するため Printf でエスケープする
	fmt.Printf("■ %%v でラップした場合\n")
	err := loadProfileWithV("999")
	fmt.Println("  メッセージ:", err)
	fmt.Println("  errors.Is(err, errUserNotFound) =", errors.Is(err, errUserNotFound), "<- 判定できない")

	fmt.Println()
	fmt.Printf("■ %%w でラップした場合\n")
	err = loadProfileWithW("999")
	fmt.Println("  メッセージ:", err)
	fmt.Println("  errors.Is(err, errUserNotFound) =", errors.Is(err, errUserNotFound), "<- 判定できる")

	fmt.Println()
	fmt.Println("■ errors.As で中身を取り出す")
	err = loadProfileWithW("")
	var target *validationError
	if errors.As(err, &target) {
		fmt.Printf("  フィールド=%s 理由=%s\n", target.Field, target.Reason)
	}

	fmt.Println()
	fmt.Println("メッセージが同じでも、判定できるかどうかが違う。")
	fmt.Println("「404 を返すか 500 を返すか」はこの判定で決まる。")
}
