package main

import (
	"fmt"
	"sync"
)

const workers = 100
const perWorker = 1000

// 複数の goroutine が同じ変数を保護なしで触っている。値がずれる
func countUnsafe() int {
	count := 0
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range perWorker {
				count++
			}
		}()
	}
	wg.Wait()
	return count
}

func countWithMutex() int {
	count := 0
	var mu sync.Mutex
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range perWorker {
				mu.Lock()
				count++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return count
}

// 共有変数を触らせず、結果を channel で集める。Go が本来推したい形
func countWithChannel() int {
	results := make(chan int, workers)
	for range workers {
		go func() {
			sum := 0
			for range perWorker {
				sum++
			}
			results <- sum
		}()
	}

	total := 0
	for range workers {
		total += <-results
	}
	return total
}

func runRace() {
	section("04. データ競合")

	expected := workers * perWorker
	fmt.Println("  期待値              :", expected)
	fmt.Println("  保護なし            :", countUnsafe(), "<- 実行するたびに変わる")
	fmt.Println("  sync.Mutex で保護   :", countWithMutex())
	fmt.Println("  channel で集約      :", countWithChannel())

	fmt.Println()
	fmt.Println("保護なしの行が期待値と一致することもある。だから怖い。")
	fmt.Println("go run -race . で実行すると WARNING: DATA RACE が出る。")
	fmt.Println("CI で -race を回すのは、この「たまたま動く」を潰すため。")
}
