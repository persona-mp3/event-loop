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
	content, err := os.ReadFile("main.go.txt")
	if err != nil {
		return nil, err
	}

	_ = content
	return "read file successfully", nil
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
	// fmt.Printf("response from example.com")
	// fmt.Println(string(content))
	t := time.Since(n)
	fmt.Printf("tt for whole_request: %+v\n", t)
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
		Meta:    runtime.PromiseMeta,
	}

	t2 := &runtime.Task{
		Id:      "read_input",
		Execute: readInput,
		Meta:    runtime.NoMeta,
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
		res, open := <-done
		if !open {
			fmt.Println("the runtime closed the done channel!")
			return
		}

		fmt.Printf("recvd: <%+v> from runtime\n", res)
	}()

	wg.Wait()
}
