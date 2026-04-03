package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"persona/runtime"
	"sync"
	"time"
)

func readFile() (any, error) {
	content, err := os.ReadFile("test-file.txt")
	if err != nil {
		return nil, err
	}

	return string(content), nil
}

func makeHttpRequest() (any, error) {
	n := time.Now()
	res, err := http.Get("https://example.com")
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	content, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	_ = content
	t := time.Since(n)
	return fmt.Sprintf("%v to make request\n", t), nil
}

func readInput() (any, error) {
	return "Read Input", nil
}

func timerFunc() (any, error) {
	return "Timer function completed", nil
}

func mockFns() []*runtime.Task {
	t1 := &runtime.Task{
		Id:      "THIRD_FUNCTION",
		Execute: readFile,
		Meta:    runtime.AsyncIOMeta,
	}

	t2 := &runtime.Task{
		Id:      "FIRST_FUNCTION",
		Execute: readInput,
		Meta:    runtime.SyncMeta,
	}

	t3 := &runtime.Task{
		Id:      "SECOND FUNCTION",
		Execute: makeHttpRequest,
		Meta:    runtime.NextTickerMeta,
	}

	// NOTE: This will run last, because it will be sitting in the timeoutQueue for 4 seconds
	// before then, all other functions could have been executed
	d4 := time.Duration(4 * time.Second)
	t4 := &runtime.Task{
		Id:       "LAST_FUNCTION",
		Execute:  timerFunc,
		Meta:     runtime.TimerMeta,
		Duration: &d4,
	}

	return []*runtime.Task{t1, t2, t3, t4}
}

func main() {
	tasks := mockFns()
	rt := runtime.NewRuntime()
	src := make(chan *runtime.Task)
	go func() {
		defer close(src)
		for _, t := range tasks {
			src <- t
		}
	}()
	done := make(chan any)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		rt.Start(src, done)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		res, open := <-done // could change it to return exit code
		if !open {
			fmt.Println("the runtime closed the done channel!")
			return
		}

		fmt.Println("Done executing all code", res)

	}()

	wg.Wait()
}
