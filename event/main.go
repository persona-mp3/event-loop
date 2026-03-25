package event

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"
)

type ProcessMeta int

const (
	NextTickMeta ProcessMeta = iota
	AsyncMeta
	TimerMeta
	IOMeta
	NoMeta
)

// NodeJS' runtime has 6 queues.
// It first runs synchronous code before executing code from
// any of the six queues. After executing all the code
// in a particular queue it then checks the MicroQueue  everytime
// This is the order in priority the RuntimeEnvironment executes
// 1. MicroTasks which contains two queues, a. nextTick and b. promise queues

// IMPLEMENTATION
// After executing tasks that are synchronous execute code in the MicroTasks Queue
// Problem statement
// We need a way to context-switch between each queue and the main stack
// Although Node doesnt 'context-switch', everything is turn based
// select {
//  case <-fromMainStack:
// }
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

type Process struct {
	Id       string
	Meta     ProcessMeta
	Execute  func() (any, error)
	result   any
	err      error
	Duration *time.Duration
}

type queue struct {
	mu        sync.RWMutex
	processes []*Process
}

type Runtime struct {
	Stack                  []*Process
	inboundNextTickerCh    chan *Process
	inboundPromiseTickerCh chan *Process
}

func NewRunTime() *Runtime {
	return &Runtime{
		Stack:                  make([]*Process, 0),
		inboundNextTickerCh:    make(chan *Process, 100),
		inboundPromiseTickerCh: make(chan *Process, 100),
	}
}

func newQueue() *queue {
	return &queue{
		mu:        sync.RWMutex{},
		processes: make([]*Process, 0),
	}
}

func (rt *Runtime) Run() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nextTickerCh := startNextTickQueue(ctx, rt.inboundNextTickerCh)
	inboundPromiseTickerCh := startPromiseQueue(ctx, rt.inboundPromiseTickerCh)

	// start filling up queues
	// var promiseQueue []*Process
	// var nextTickerQueue []*Process
	promiseQueue := newQueue()
	nextTickerQueue := newQueue()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case p := <-inboundPromiseTickerCh:
				promiseQueue.mu.Lock()
				promiseQueue.processes = append(promiseQueue.processes, p)
				promiseQueue.mu.Unlock()
			case p := <-nextTickerCh:
				nextTickerQueue.mu.Lock()
				nextTickerQueue.processes = append(nextTickerQueue.processes, p)
				nextTickerQueue.mu.Unlock()
			}
		}
	}()

	// Execute all non-async code first
	allTasks := len(rt.Stack)
	totalExecuted := 0

	for _, process := range rt.Stack {
		switch process.Meta {
		case AsyncMeta:
			rt.inboundPromiseTickerCh <- process
			continue
		case NextTickMeta:
			rt.inboundNextTickerCh <- process
			continue
		case IOMeta:
			fmt.Println("ignoring io process")
			continue
		case TimerMeta:
			fmt.Println("ignoring timer process")
			continue
		}

		totalExecuted++
		result, err := process.Execute()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error executing fn <%s>\n", process.Id)
			fmt.Fprintf(os.Stderr, "%4s\n", err)
			return
		}
		fmt.Printf(" ========= function <%s> result\n", process.Id)
		fmt.Printf("%2s\n", result)
		fmt.Println(" ==========")
	}

	rt.Stack = nil

	// Now we need to read from each of these queues
	callbacks := drainQueue(nextTickerQueue)
	for _, process := range callbacks {
		totalExecuted++
		result, err := process.Execute()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error executing fn <%s>\n", process.Id)
			fmt.Fprintf(os.Stderr, "%s\n", err)
			return
		}
		fmt.Printf(" ========= function <%s> result\n", process.Id)
		fmt.Printf("%4s\n", result)
		fmt.Println(" ==========")

	}
	// The promise queue contains functions that are native promises in js ie async/await
	// Since these functions need to be executed in the background, their
	// results and error are provided in the process' fields, and now Run can just
	// read from them instead of calling a wrapped-around function
	promiseCallbacks := drainQueue(promiseQueue)
	for _, process := range promiseCallbacks {
		totalExecuted++
		if process.err != nil {
			fmt.Fprintf(os.Stderr, "error executing fn <%s>\n", process.Id)
			fmt.Fprintf(os.Stderr, "%s\n", process.err)
			return
		}

		fmt.Printf(" ========= function <%s> result\n", process.Id)
		fmt.Printf("%4s\n", process.result)
		fmt.Println(" ==========")

	}

	fmt.Printf("total processes executed: %d vs total processes provided: %d\n", totalExecuted, allTasks)

	// TODO(daniel)  We'll need to check the normal stack
	// if there are more Processes to execute, so it think
	// the Run() would need to take an in channel that it spins from
	// and when no more functions to leaves it and listens from the others
	// it's going to be a bit tricky, but we'll get there
}

// reads from a queue using the mutex stored
// assigned to them, and returns a copy of all
// processes in the queue at the time.
// After that, the queue is cleared.
func drainQueue(q *queue) []*Process {
	var snapshot []*Process
	q.mu.Lock()
	for _, p := range q.processes {
		snapshot = append(snapshot, p)
	}
	// clear the queue
	q.processes = nil
	q.mu.Unlock()

	return snapshot
}

func startNextTickQueue(ctx context.Context, in <-chan *Process) <-chan *Process {
	out := make(chan *Process)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case p := <-in:
				go func() { out <- p }()
			}
		}
	}()
	return out
}

func startPromiseQueue(ctx context.Context, in <-chan *Process) <-chan *Process {
	out := make(chan *Process)
	results := make(chan *Process)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return

			case p := <-in:
				go p.executePromise(ctx, results)

			case doneProcess := <-results:
				// to avoid getting this queue blocked we send into a goroutine
				// it would make more these timeout based and ctx based, because
				// if an async function never resolves, that's a problem
				go func() {
					select {
					case <-ctx.Done():
						return
					case out <- doneProcess:
					}
				}()
			}
		}
	}()
	return out
}

func (p *Process) executePromise(ctx context.Context, out chan<- *Process) {
	var result any
	var err error

	select {
	case <-ctx.Done():
		return
	default:
		result, err = p.Execute()
		p.result = result
		p.err = err
	}

	out <- p
}
