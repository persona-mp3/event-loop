package runtime

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	// "time"
)

type Meta int

type Task struct {
	Id      string
	Execute func() (any, error)
	// Kind of function
	Meta
	// for promises.resolve and promises.reject
	resolve any
	reject  error
}

type queue struct {
	mu    sync.RWMutex
	tasks []*Task
}

type Runtime struct {
	inflight    *atomic.Int64
	stack       *queue
	promiseQ    *queue
	nextTickerQ *queue
}

func newQueue() *queue {
	return &queue{
		mu:    sync.RWMutex{},
		tasks: make([]*Task, 0),
	}
}
func NewRuntime() *Runtime {
	return &Runtime{
		inflight:    &atomic.Int64{},
		stack:       newQueue(),
		promiseQ:    newQueue(),
		nextTickerQ: newQueue(),
	}
}

// the source could mean a file
func (rt *Runtime) Start(source <-chan *Task) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ready := make(chan bool)
	go rt.startEnvironments(ctx, source, ready)

	<-ready
	fmt.Println("recvd start opt")

	stack := rt.drainQueue(rt.stack)

	for _, task := range stack {
		result, err := task.Execute()
		if err != nil {
			fmt.Fprintf(os.Stderr, "err executing func: %s\n", task.Id)
			fmt.Fprintf(os.Stderr, "%3s\n", err)
			return
		}

		fmt.Println("result from func: ", task.Id)
		fmt.Printf("%s\n\n", result)
	}

	fmt.Println("executing next_tickers")
	nextTickerQ := rt.drainQueue(rt.nextTickerQ)
	for _, task := range nextTickerQ {
		result, err := task.Execute()
		if err != nil {
			fmt.Fprintf(os.Stderr, "err executing func: %s\n", task.Id)
			fmt.Fprintf(os.Stderr, "%3s\n", err)
			return
		}

		fmt.Println("result from func: ", task.Id)
		fmt.Printf("%s\n\n", result)
	}

	fmt.Println("execuing promise queue")
	promises := rt.drainQueue(rt.promiseQ)
	for _, task := range promises {
		result, err := task.resolve, task.reject
		if err != nil {
			fmt.Fprintf(os.Stderr, "reject: while executing func: %s\n", task.Id)
			fmt.Fprintf(os.Stderr, "%3s\n", err)
			return
		}

		fmt.Println("promise resolved for func: ", task.Id)
		fmt.Printf("%s\n\n", result)
	}

	rt.eventLoop(ctx)
}

const (
	// Synchronous task
	NoMeta Meta = iota

	NextTickerMeta

	// Async/Await task
	PromiseMeta
)

// All tasks that are not syncrhonous ie NoMeta have
// their code executed in a seperate goroutine. When done,
// they append their results to their respective Queues.
//
// NoMeta tasks are appended directly to the stack, because
// these are asynchronous code
func (rt *Runtime) startEnvironments(ctx context.Context, src <-chan *Task, ready chan<- bool) {
	fmt.Println("environment started")

	prefix := "env: "
	close(ready)
	for {
		select {

		case <-ctx.Done():
			fmt.Printf("%s parent canceled: %s\n", prefix, ctx.Err())
			return
		case t, open := <-src:
			if !open {
				fmt.Printf("%s source closed\n", prefix)
				return
			}

			switch t.Meta {
			case NoMeta:
				// might be better to just execute here?
				appendToQueue(rt.stack, t)
			case NextTickerMeta: // no speical operations get ran in the nextTickerQ, its just like appending to the stack
				appendToQueue(rt.nextTickerQ, t)
			case PromiseMeta:
				go rt.execPromise(t)

			}

		}
	}
}

// Starts at the IO Queue, then to the MicroQueues
// and follow the sequential order described in the docs
func (rt *Runtime) eventLoop(ctx context.Context) {
	_ = ctx
	pref := "evt_loop"
	for {
		println("spininn", rt.inflight.Load())
		if rt.inflight.Load() == 0 {
			fmt.Printf("%s no more funcs to run, total_inflight: %d\n", pref, rt.inflight.Load())
			break
		}
	}
}

func (rt *Runtime) drainQueue(q *queue) []*Task {
	var snapshot []*Task
	q.mu.Lock()
	for _, p := range q.tasks {
		snapshot = append(snapshot, p)
	}
	// clear the queue
	q.tasks = nil
	q.mu.Unlock()

	return snapshot
}

func appendToQueue(q *queue, t *Task) {
	q.mu.Lock()
	q.tasks = append(q.tasks, t)
	q.mu.Unlock()
}

type fn func() (any, error)
type result struct {
	success any
	err     error
}

func runIO(fn fn, done chan<- *result) {
	res, err := fn()

	done <- &result{success: res, err: err}
}

func (rt *Runtime) wrapPromise(t *Task) {
	done := make(chan *result)
	go runIO(t.Execute, done)
	go func() {
		res := <-done
		t.reject = res.err
		t.resolve = res.success
		appendToQueue(rt.promiseQ, t)
		rt.inflight.Add(-1)
	}()
}

func (rt *Runtime) execPromise(t *Task) {
	rt.inflight.Add(1)
	rt.wrapPromise(t)
	// pref := "promise_q"
	// fmt.Printf("%s executing %s\n", pref, t.Id)
	// rt.inflight.Add(1)
	// result, err := t.Execute()
	// t.reject = err
	// t.resolve = result
	// appendToQueue(rt.promiseQ, t)
	// rt.inflight.Add(-1)
}
