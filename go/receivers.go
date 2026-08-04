package main

import "fmt"

type counter struct {
	count int
}

// 値レシーバはコピーを受け取る。呼び出し元の値は変わらない
func (c counter) IncrementByValue() {
	c.count++
}

// ポインタレシーバは実体を指す。呼び出し元の値が変わる
func (c *counter) IncrementByPointer() {
	c.count++
}

func runReceivers() {
	section("03. 値レシーバとポインタレシーバ")

	c := counter{}

	for range 3 {
		c.IncrementByValue()
	}
	fmt.Println("  値レシーバで3回呼んだあと    :", c.count, "<- 増えていない")

	for range 3 {
		c.IncrementByPointer()
	}
	fmt.Println("  ポインタレシーバで3回呼んだあと:", c.count)

	fmt.Println()
	fmt.Println("コンパイルは通り、テストがなければ気づけない。")
	fmt.Println("状態を変えるメソッドはポインタレシーバにする。")
	fmt.Println("同じ型のメソッドは、どちらかに揃えるのが慣習。")
}
