package main

import "fmt"

func main() {
	runErrors()
	runNilTraps()
	runReceivers()
	runRace()
	runContext()

	fmt.Println()
	fmt.Println("--- 以上 5 本 ---")
	fmt.Println("データ競合を検出するには: go run -race .")
}

func section(title string) {
	fmt.Println()
	fmt.Println("===", title, "===")
}
