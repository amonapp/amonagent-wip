package main

import (
	"fmt"
	"sync"
	"time"
)

func main() {
	wg := sync.WaitGroup{}
	wg.Add(1)
	c := make(chan struct{})
	go func() {
		wg.Wait()
		c <- struct{}{}
	}()
	timeout := time.Duration(1) * time.Second
	fmt.Printf("Wait for waitgroup (up to %s)\n", timeout)
	select {
	case <-c:
		fmt.Printf("Wait group finished\n")
	case <-time.After(timeout):
		fmt.Printf("Timed out waiting for wait group\n")
	}
	fmt.Printf("Free at last\n")
}
