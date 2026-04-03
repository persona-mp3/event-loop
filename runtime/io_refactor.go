package runtime

import (
	"context"
	"time"
)

// NOTE: v8 cannot make syscalls or access the filesystem,
// it relies on libuv to do that, and then the result or error, is wrapped
// via Node for v8 to handle. Also depending on the api used
// ie `promises`, it doesn't go to the promise queue, and to reduce
// abstraction, we'll execute the function as a NoMeta task leaving
// only execAsyncIO as the promise version

func execAsyncIO(ctx context.Context, callback fn, done chan *result) {
	res, err := callback()
	select {
	case <-ctx.Done():
		return
	case done <- &result{success: res, err: err}:
	}
}

// General execution of promises. Results and errors are
// attached to the result and reject props, and appended to the promise queue
func (rt *Runtime) nodeExecPromise(t *Task) {
	result, err := t.Execute()
	t.resolve = result
	t.reject = err
	appendToQueue(rt.promiseQ, t)
}

// wrapPromise executes an AsyncIO bound task ie whose meta type
// is AsyncIO. The results are appended to the IOQueue to be
// resolved by the mainCaller
func (rt *Runtime) nodeWrapPromise(ctx context.Context, t *Task) {
	logger.Println("DEBUG -> nodeWrapPromise: before: ", rt.inflight)
	logger.Println("DEBUG -> after -> ", rt.inflight)
	rt.inflight.Add(1)
	done := make(chan *result)
	childCtx, cancel := context.WithCancel(ctx)
	go execAsyncIO(childCtx, t.Execute, done)
	go func() {
		defer cancel()
		defer func() {
			logger.Println("DEBUG -> NWP removing::", rt.inflight)
			logger.Println("DEBUG -> NWP removed::", rt.inflight)
		}()

		select {
		case <-ctx.Done():
			rt.inflight.Add(-1)
			return
		case res := <-done:
			t.reject = res.err
			t.resolve = res.success
			appendToQueue(rt.asyncIOQueue, t)
			rt.inflight.Add(-1)
			return
		}
	}()
}

func (rt *Runtime) nodeExecTimer(ctx context.Context, t *Task) {
	if t.Duration == nil {
		logger.Println("Timer is Nil, assuming 0-seconds -> ", t.Duration)
		appendToQueue(rt.timerQ, t)
		return
	}

	// I think the problem is that we are adding a timer before/during the error
	rt.inflight.Add(1)
	defer func() {
		logger.Println("DEBUG ] reducing inflightRoutines for execTimer", rt.inflight)
		logger.Println("DEBUG ] after for execTimer", rt.inflight)
	}()

	timer := time.NewTimer(*t.Duration)

	select {
	case <-ctx.Done():
		rt.inflight.Add(-1)
		return
	case <-timer.C:
		rt.inflight.Add(-1)
		appendToQueue(rt.timerQ, t)
	}
}
