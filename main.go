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

func readFile() error {
	content, err := os.ReadFile("main.go.txt")
	if err != nil {
		return err
	}

	_ = content
	fmt.Println("read file successfully")
	return nil
}

func makeHttpRequest() error {
	res, err := http.Get("https://example.com")
	if err != nil {
		return err
	}

	defer res.Body.Close()
	content, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	_ = content
	// fmt.Printf("response from example.com")
	// fmt.Println(string(content))
	fmt.Println("http_request successfull")
	return nil
}

func readInput() error {
	scanner := bufio.NewScanner(os.Stdin)
	var username string
	var bio string
	fmt.Print("Please provide username: ")
	scanner.Scan()
	username = scanner.Text()
	fmt.Print("Please provide bio: ")
	scanner.Scan()
	bio = scanner.Text()

	fmt.Printf("%s has a bio that says: %s\n", username, bio)
	return nil
}

func newTasks() []*event.Task {
	t1 := &event.Task{
		Id: "read_file",
		Fn: readFile,
	}

	d := time.Second * 3
	t2 := &event.Task{
		Id:       "read_input",
		Fn:       readInput,
		Duration: &d,
	}

	d2 := time.Second * 2
	t3 := &event.Task{
		Id:       "make_request",
		Fn:       makeHttpRequest,
		Duration: &d2,
	}

	return []*event.Task{t1, t2, t3}
}

func main() {
	tasks := newTasks()
	node := event.NewEnvironment()
	node.Stack = tasks
	node.Run()
}
