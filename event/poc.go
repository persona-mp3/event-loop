package main

import (
	"context"
	"sync"
)

// Problem statement
// We need a way to context-switch between each queue and the main stack
// Although Node doesnt 'context-switch', everything is turn based
// 					mainStack microQueue ioQueue timerQueue
// 						^          *
// 						*          ^
// 						^          _        *
// 						*          _        ^
// 						^          _        _        *
//
//  ^ :  current_execution_s
//  _ :  already_emptied_s
//  * :  next_executino_s

type queue struct {
	mu *sync.RWMutex
	tasks []string
}

func main() {
	stack := &queue{}
	next_ticker := &queue{}
	promise := &queue{}
	// can we get something like a signal?
	for {
		readqueue(stack)

		readqueue(next_ticker)

		readqueue(promise)
	}
}

type signal int

const (
	Stack signal = iota
	Micro
	Promise
	Done
)

type env struct {
	stack   *queue
	micro   *queue
	promise *queue
	src     chan string
}

// so this is just the 'runtime' executing functions continously, theh
// stack will be fed from an outside source.
// we first check the stack inside the while loop and block till we receive a done
// then switch to the micro queue, and then back to the stack and so forth
func (e *env) iterator() {
	for task := range e.src {
		// filter tasks to stack or micro or promise queues
		// so only this one is writing
		_ = task
	}
}

func runner() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sig := make(chan signal, 50)
	done := make(chan signal, 50)

	env := &env{
		stack:   &queue{},
		micro:   &queue{},
		promise: &queue{},
	}

	go env.loop(ctx, sig, done)
	for {
		if env.done() {
			return
		}

		if len(env.stack.tasks) > 0 {
			sig <- Stack
			<-done
		}

		if len(env.micro.tasks) > 0 {
			sig <- Micro
			<-done
		}

		if len(env.stack.tasks) > 0 {
			sig <- Stack
			<-done
		}

		if len(env.promise.tasks) > 0 {
			sig <- Promise
			<-done
		}

		if len(env.stack.tasks) > 0 {
			sig <- Stack
			<-done
		}

	}
}
func (q *env) loop(ctx context.Context, sig chan signal, done chan signal) {
	for {
		select {
		case <-ctx.Done():
			return
		case s := <-sig:
			switch s {
			case Stack:
				exec_stack(q.stack)
				done <- Done
			case Micro:
				exec_micro(q.micro)
				done <- Done
			case Promise:
				exec_promise(q.promise)
				done <- Done
			case Done:
				return
			}
		}

	}
}

func exec_stack(q *queue)   {}
func exec_micro(q *queue)   {}
func exec_promise(q *queue) {}

func readqueue(q *queue) {}

func (e *env) done() bool {
	// need to lock everything here
	return len(e.stack.tasks) == 0 && len(e.micro.tasks) == 0 && len(e.promise.tasks) == 0
}
