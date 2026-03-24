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

// type Meta struct {
// 	Duration *time.Duration
//
// 	Async bool // mark functions as promises
// 	IO    bool
// }

type Process struct {
	Id       string
	Meta     ProcessMeta
	Execute  func() (any, error)
	result   any
	err      error
	Duration *time.Duration
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

func (rt *Runtime) Run() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nextTickerCh := startNextTickQueue(ctx, rt.inboundNextTickerCh)
	inboundPromiseTickerCh := startPromiseQueue(ctx, rt.inboundPromiseTickerCh)

	// start filling up queues
	var promiseQueue []*Process
	var nextTickerQueue []*Process
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case p := <-inboundPromiseTickerCh:
				promiseQueue = append(promiseQueue, p)
			case p := <-nextTickerCh:
				nextTickerQueue = append(nextTickerQueue, p)
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

	// nexttickerqueue simply places functions next on the stack, with methods like
	// `setimmediate` in js, so we just leave it's execution to the main run
	// So regarding this race condition
	// we could do something like this:
	// func drainQueue(q *Queue, send chan<- *Process) {
	//   m.RLock()
	//   for _, p := q {
	//       send <- p
	//    }
	//   m.RUnLock()
	// }
	// We need to actually drain all the current Processes in here
	// but we also need to stop all writing to q. So either we 
	// just consume them through a channel, or just copy it over 
	// But when you look at perf, that's probably the last thing that we'd need to do 
	// unless go can handle it properly? then it would be rt.Snapshot(q) -> *[]Task
	for _, process := range nextTickerQueue {
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

	// TODO(daniel) The actual stack pops fn calls as soon as they're done
	// but for saftey, we could just clear it out, then later
	// on if need be, we can clear upon each iteration
	nextTickerQueue = nil

	// The promise queue are basically functions that
	// are native promises in js ie async/await
	// Since these functions need to be executed in the background, their
	// results and error are provided in the process, and now Run can just
	// read from them instead of wrapping it in another function
	for _, process := range promiseQueue {
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

	promiseQueue = nil

	fmt.Printf("total processes executed: %d vs total processes provided: %d\n", totalExecuted, allTasks)

	// TODO(daniel)  We'll need to check the normal stack
	// if there are more Processes to execute, so it think
	// the Run() would need to take an in channel that it spins from
	// and when no more functions to leaves it and listens from the others
	// it's going to be a bit tricky, but we'll get there
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
