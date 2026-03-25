package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"persona/event"
	"time"
)

func readFile() (any, error) {
	content, err := os.ReadFile("main.go.txt")
	if err != nil {
		return nil, err
	}

	_ = content
	fmt.Println("read file successfully")
	return "read file successfully", nil
}

func makeHttpRequest() (any, error) {
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

func newTasks() []*event.Process {
	t1 := &event.Process{
		Id:      "read_file",
		Execute: readFile,
		Meta:    event.IOMeta,
	}

	t2 := &event.Process{
		Id:      "read_input",
		Execute: readInput,
		Meta: event.NoMeta,
	}

	d2 := time.Second * 2
	t3 := &event.Process{
		Duration: &d2,
		Id:       "make_request",
		Execute:  makeHttpRequest,
		Meta:     event.TimerMeta,
	}

	return []*event.Process{t1, t2, t3}
}

func main() {
	// tasks := newTasks()
	// node := event.NewEnvironment()
	// node.Stack = tasks
	// node.Run()

	rt := event.NewRunTime()
	rt.Stack = newTasks()
	rt.Run()
}
