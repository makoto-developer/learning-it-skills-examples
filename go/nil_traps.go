package main

import "fmt"

type user struct {
	Name string
}

type notFoundError struct{}

func (e *notFoundError) Error() string { return "見つかりません" }

// 戻り値が error 型なのに *notFoundError を返している。これが後の罠になる
func findUserBad(id string) *notFoundError {
	if id == "missing" {
		return &notFoundError{}
	}
	return nil
}

func findUserGood(id string) error {
	if id == "missing" {
		return &notFoundError{}
	}
	return nil
}

// panic はプロセスを落とす。ここでは観察のために recover で受け止める
func observePanic(label string, fn func()) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("  %s -> panic: %v\n", label, r)
		}
	}()
	fn()
	fmt.Printf("  %s -> 落ちなかった\n", label)
}

func runNilTraps() {
	section("02. nil の罠")

	fmt.Println("■ nil ポインタの参照")
	var u *user
	observePanic("u.Name を読む", func() { _ = u.Name })
	observePanic("nil チェックしてから読む", func() {
		if u == nil {
			return
		}
		_ = u.Name
	})

	fmt.Println()
	fmt.Println("■ nil マップ / nil スライス（挙動が違う）")
	var m map[string]int
	var s []int
	fmt.Println("  nil マップの読み取り:", m["key"], "(ゼロ値が返る。落ちない)")
	fmt.Println("  nil マップの len   :", len(m))
	observePanic("nil マップへの書き込み", func() { m["key"] = 1 })
	s = append(s, 1, 2)
	fmt.Println("  nil スライスへの append:", s, "(こちらは落ちない)")

	fmt.Println()
	fmt.Println("■ nil インターフェースの罠（Go で最も有名な事故）")
	var errBad error = findUserBad("found")
	errGood := findUserGood("found")
	fmt.Println("  findUserBad(\"found\")  == nil ?", errBad == nil, "<- nil を返したはずなのに false")
	fmt.Println("  findUserGood(\"found\") == nil ?", errGood == nil)

	fmt.Println()
	fmt.Println("インターフェースは (型, 値) の組。型が入っていれば値が nil でも nil ではない。")
	fmt.Println("だから error を返す関数の戻り値は、具体型ではなく error 型で宣言する。")
}
