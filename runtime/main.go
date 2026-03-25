package runtime

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
)

// descibes the type and nature of Task
type Meta int

type Task struct {
	// Name of the function or any uuid
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

func (rt *Runtime) Start(source <-chan *Task, done chan any) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// need to make sure that other channles have started and are ready
	ready := make(chan bool)
	go rt.startEnvironments(ctx, source, ready)

	<-ready
	fmt.Println("recvd start opt")

	stack := rt.drainQueue(rt.stack)

	fmt.Println("running from stack")
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
	done <- true
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
// these are synchronous tasks
func (rt *Runtime) startEnvironments(ctx context.Context, src <-chan *Task, ready chan<- bool) {
	fmt.Println("environment started")

	prefix := "env: "
	ready <- true
	for {
		select {

		case <-ctx.Done():
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


// Starts at the IO Queue, then to the MicroQueues and then follows sequential
// order described in the docs. It breaks out by checking that the number of inflight go-routines
// is 0, otherwise, it continues running
func (rt *Runtime) eventLoop(ctx context.Context) {
	_ = ctx
	pref := "evt_loop"
	for {
		if rt.inflight.Load() == 0 {
			fmt.Printf("%s total_inflight routines: %d\n", pref, rt.inflight.Load())
			break
		}
	}
}

// drainQueue locks the queue for reading and writing and unlocks it when done
// to create a copy of the queue, it's caller can use.
// When done, it sets the queue to nil to empty it.
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

// Results and error to be returned from async based goroutines
type result struct {
	success any
	err     error
}

// runIO simulates libuv's C++ os capabilities. After executing fn
// it's result and error are both passed into the done channel.
// Callers should run this in a goroutine to avoid blocking
func runIO(fn fn, done chan<- *result) {
	res, err := fn()
	done <- &result{success: res, err: err}
}

// This is to simulate Node & C++ bindings itself. When a task is marked as async
// or it's nature involves async/await, the libuv (for io, etc) executes the task
// using a thread from its pool. When done, it returns the result, or error,
// to Node to then wrap the result into a `Promise`. This `Promise` is then
// sent to promise queue for v8 to resolve/reject, which is what we do here
// The async bound task is ran in a routine, it's result is wrapped, and then
// added into the 'promise' queue. This fuction is none blocking. When the result
// has been wrapped, the goroutine reduces the inflight field in rt *Runtime
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
