package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"persona/runtime"
	"sync"
	"time"
)

func readFile() (any, error) {
	content, err := os.ReadFile("gitlogs.txt")
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
	fmt.Printf("  total time to make request: %+v\n", t)
	return "http_request successfull", nil
}

func readInput() (any, error) {
	scanner := bufio.NewScanner(os.Stdin)
	var username string
	var bio string
	fmt.Print("Please provide username: ")
	scanner.Scan()
	username = scanner.Text()
	fmt.Print("Please provide bio: ")
	scanner.Scan()
	bio = scanner.Text()

	result := fmt.Sprintf("%s has a bio that says: %s\n", username, bio)
	return result, nil
}

func mockFns() []*runtime.Task {
	t1 := &runtime.Task{
		Id:      "read_file",
		Execute: readFile,
		// import fs from "fs/promises"
		// fs.open() -> IOQueue -> C++, io-operation -> result -> Node -> result, error -> promise (resolve, reject)
		Meta: runtime.AsyncIOMeta,
	}

	t2 := &runtime.Task{
		Id:      "read_input",
		Execute: readInput,
		Meta:    runtime.SyncMeta,
	}

	t3 := &runtime.Task{
		Id:      "make-request",
		Execute: makeHttpRequest,
		Meta:    runtime.NextTickerMeta,
	}

	return []*runtime.Task{t1, t2, t3}
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
