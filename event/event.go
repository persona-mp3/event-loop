package event

import (
	"context"
	"fmt"
	"os"
	"time"
)

type Task struct {
	Id        string
	Fn        func() error
	Duration  *time.Duration
	Awaitable bool
}

type Environment struct {
	// All expected functions or tasks to be ran
	Stack []*Task
	// All functions with timeouts, are resolved here first
	timerQueue []*Task
}

func NewEnvironment() *Environment {
	return &Environment{
		Stack: make([]*Task, 0),
	}
}

// To emulate the Node environment, the Stack member
// represents javascripts V8 Engine, that executes js code, sychronoulsy
//
// The callbackQueue are for functions that can be executed by the Stack
// but were previoulsy in a timeout queue, await queue, or io queue
func (e *Environment) Run() {
	if len(e.Stack) == 0 {
		return
	}

	// This is were timeout functions will be sent into
	// to waitout their task.Duration time
	timeoutCh := make(chan *Task)

	// All apis will write to this channel
	// and the main execution thread, Start
	// will be the one executing all tasks.Fn from callbackCh
	callbackCh := make(chan *Task, 100)
	// defer close(timeoutCh)
	// defer close(callbackCh)

	allTasks := len(e.Stack)
	executedTasks := 0

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go runMicroQueue(ctx, timeoutCh, callbackCh)

	for _, task := range e.Stack {
		if task.Duration != nil {
			timeoutCh <- task
			continue
		}

		executedTasks++
		if err := task.Fn(); err != nil {
			fmt.Fprintf(os.Stderr, "uncaught exception: err: %s\n", err)
			return
		}
	}

	fmt.Println("executed_tasks:", executedTasks)
	fmt.Println("total_tasks:", allTasks)
	for {
		if executedTasks == allTasks {
			fmt.Fprintf(os.Stdout, "Executed all tasks <%d> successfully\n", executedTasks)
			return
		}
		task := <-callbackCh

		if err := task.Fn(); err != nil {
			fmt.Fprintf(os.Stderr, "error occured: %s\n", err)
			return
		}
		executedTasks++
	}
}

// At the moment, the runMicroQueue is used to run timeout functions
// but in the Node Environment the timeout queue is responsible for this
// This queue will later be used to run await based code
func runMicroQueue(ctx context.Context, in <-chan *Task, cbCh chan<- *Task) {
	for {
		select {
		case <-ctx.Done():
			fmt.Printf("closing microQueue routine: %s\n", ctx.Err())
			return
		case task, open := <-in:
			if !open {
				return
			}

			fmt.Printf("recvd task with timeout-duration %s\n", task.Duration.String())
			go func() {
				timer := time.NewTimer(*task.Duration)
				<-timer.C
				fmt.Printf("timeout passed for task <%s>, passing to cb \n", task.Id)
				task.Duration = nil
				cbCh <- task
			}()

		}
	}
}
