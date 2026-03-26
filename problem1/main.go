package main

import (
	"fmt"
	"sync"
)

func withSyncMap() {
	var m sync.Map
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(key int) {
			defer wg.Done()
			m.Store("key", key)
		}(i)
	}

	wg.Wait()

	val, _ := m.Load("key")
	fmt.Printf("[sync.Map] Value: %v\n", val)
}

func withRWMutex() {
	safeMap := make(map[string]int)
	var mu sync.RWMutex
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(key int) {
			defer wg.Done()
			mu.Lock()
			safeMap["key"] = key
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	mu.RLock()
	value := safeMap["key"]
	mu.RUnlock()
	fmt.Printf("[sync.RWMutex] Value: %d\n", value)
}

func main() {
	withSyncMap()
	withRWMutex()
}
