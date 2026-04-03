package runtime

import (
	"context"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// describes the type and nature of Task
type Meta int

// Now IOMeta and NoMeta supposedly mean the same thing
// to run this function synchornously. But v8 can't do syscalls
// so it has to rely on libuv hence, the seperation

const (
	// Synchronous task
	SyncMeta Meta = iota

	NextTickerMeta

	// Async/Await task
	PromiseMeta

	IOMeta
	AsyncIOMeta

	// So anytime we recv a task with a timer, we also expect
	// a timer dutation
	TimerMeta
)

type Task struct {
	// Name of the function or any uuid
	Id      string
	Execute func() (any, error)

	// Kind of function
	Meta
	// for promises.resolve and promises.reject
	resolve any
	reject  error

	Duration *time.Duration
}

type queue struct {
	mu    sync.RWMutex
	tasks []*Task
}

type Runtime struct {
	// Current number io-bound goroutine workers
	inflight     *atomic.Int64
	stack        *queue
	promiseQ     *queue
	nextTickerQ  *queue
	asyncIOQueue *queue
	timerQ       *queue
	exitStatus   int
	ctx          context.Context
}

func newQueue() *queue {
	return &queue{
		mu:    sync.RWMutex{},
		tasks: make([]*Task, 0),
	}
}
func NewRuntime() *Runtime {
	return &Runtime{
		inflight:     &atomic.Int64{},
		stack:        newQueue(),
		promiseQ:     newQueue(),
		asyncIOQueue: newQueue(),
		timerQ:       newQueue(),
		nextTickerQ:  newQueue(),
	}
}

var errLogger = log.New(os.Stderr, "[error] ", log.LstdFlags)

// success logger
var logger = log.New(os.Stdout, "[success] ", log.LstdFlags)

func (rt *Runtime) Start(source <-chan *Task, done chan any) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// All synchronous code from the source will be executed first
	// until it closes
	stackCh := make(chan *Task, 100)
	go rt.startEnvironments(ctx, source, stackCh)

	// Once the source channel has been closed, this channel will also
	// The receiver will always be waiting. This is to make sure
	// that the caller doesn't exit before we finish executing all tasks
	defer func() {
		done <- rt.exitStatus
		rt.debugInfo()
	}()

	// if the stack is closed, synchronous code to run, and we can
	// proceed into looking to execute other queues
	log.Printf("executing all sync code\n\n")
	for task := range stackCh {
		result, err := task.Execute()
		if err != nil {
			errLogger.Printf("err executing func: %s\n", task.Id)
			errLogger.Printf("%3s\n", err)
			rt.exitStatus = 1
			return
		}

		logger.Println()
		logger.Printf("\n=======================%s=======================\n", task.Id)
		logger.Printf("%s\n==============================================\n", result)
		// logger.Println("result from func: ", task.Id)
		// logger.Printf("%s\n\n", result)
	}

	rt.eventLoop(ctx)

}

// All tasks that are not syncrhonous ie NoMeta have
// their code executed in a seperate goroutine. When done,

// they append their results to their respective Queues.
//
// NoMeta tasks are appended directly to the stack, because
// these are synchronous tasks
func (rt *Runtime) startEnvironments(ctx context.Context, src <-chan *Task, stackCh chan<- *Task) {

	prefix := "env: "
	defer close(stackCh)
	for {
		select {

		case <-ctx.Done():
			return
		case t, open := <-src:
			if !open {
				logger.Printf("%s source closed\n", prefix)
				return
			}

			switch t.Meta {
			case SyncMeta:
				stackCh <- t
			case IOMeta:
				stackCh <- t
			case NextTickerMeta: // no speical operations get ran in the nextTickerQ, its just like appending to the stack
				appendToQueue(rt.nextTickerQ, t)
			case PromiseMeta:
				go rt.nodeExecPromise(t)
			case AsyncIOMeta:
				rt.nodeWrapPromise(ctx, t)
			case TimerMeta:
				// ha, noope
				go rt.nodeExecTimer(ctx, t)

			}

		}
	}
}

// Starts at the IO Queue, then to the MicroQueues and then follows sequential
// order described in the docs. It breaks out by checking that the number of inflight go-routines
// is 0, otherwise, it continues running
func (rt *Runtime) eventLoop(ctx context.Context) {
	defer cleanUp(rt)

	for {
		// time.Sleep(20 * time.Millisecond) // avoid overruning
		if ctx.Err() != nil {
			errLogger.Println("evtloop leaving, error-> ", ctx.Err())
			return
		}

		if rt.queuesAreEmpty() {
			break
		}

		nextTickerQ := rt.drainQueue(rt.nextTickerQ)
		if err := execTasks(nextTickerQ); err != nil {
			rt.exitStatus = 1
			return
		}

		promises := rt.drainQueue(rt.promiseQ)
		if err := execPromises(promises); err != nil {
			rt.exitStatus = 1
			return
		}

		timerQ := rt.drainQueue(rt.timerQ)
		if err := execTasks(timerQ); err != nil {
			rt.exitStatus = 1
			return
		}

		// TODO: or is it because of the timer opts
		asyncIOQ := rt.drainQueue(rt.asyncIOQueue)
		if err := execTasks(asyncIOQ); err != nil {
			rt.exitStatus = 1
			return
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

type fn func() (any, error)

// Results and error to be returned from async based goroutines
type result struct {
	success any
	err     error
}

func (rt *Runtime) queuesAreEmpty() bool {
	return (rt.inflight.Load() == 0 &&
		len(rt.stack.tasks) == 0 &&
		len(rt.nextTickerQ.tasks) == 0 &&
		len(rt.timerQ.tasks) == 0 &&
		len(rt.promiseQ.tasks) == 0)
}

// runIO simulates libuv's C++ os capabilities. After executing fn
// it's result and error are both passed into the done channel.
// Callers should run this in a goroutine to avoid blocking
// func runIO(ctx context.Context, fn fn, done chan<- *result) {
// 	res, err := fn()
// 	select {
// 	case <-ctx.Done():
// 		return
// 	case done <- &result{success: res, err: err}:
// 	}
// }

// This is to simulate Node & C++ bindings itself. When a task is marked as async
// or it's nature involves async/await, the libuv (for io, etc) executes the task
// using a thread from its pool. When done, it returns the result, or error,
// to Node to then wrap the result into a `Promise`. This `Promise` is then
// sent to promise queue for v8 to resolve/reject, which is what we do here
// The async bound task is ran in a routine, it's result is wrapped, and then
// added into the 'promise' queue. This fuction is none blocking. When the result
// has been wrapped, the goroutine reduces the inflight field in rt *Runtime

// func (rt *Runtime) _wrapPromise(t *Task) {
// 	done := make(chan *result)
// 	go runIO(rt.ctx, t.Execute, done)
// 	go func() {
// 		select {
// 		case <-rt.ctx.Done():
// 			return
// 		case res := <-done:
// 			t.reject = res.err
// 			t.resolve = res.success
// 			appendToQueue(rt.promiseQ, t)
// 			rt.inflight.Add(-1)
// 		}
//
// 	}()
// }
