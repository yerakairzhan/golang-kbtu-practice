package main

import (
	"fmt"
	"sync"
	"sync/atomic"
)

func withMutex() {
	var counter int
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mu.Lock()
			counter++
			mu.Unlock()
		}()
	}

	wg.Wait()
	fmt.Printf("[sync.Mutex] Counter: %d\n", counter)
}

func withAtomic() {
	var counter int64
	var wg sync.WaitGroup

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			atomic.AddInt64(&counter, 1)
		}()
	}

	wg.Wait()
	fmt.Printf("[atomic] Counter: %d\n", counter)
}

func main() {
	withMutex()
	withAtomic()
}
