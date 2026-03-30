package runtime

import "context"

// NOTE: v8 cannot make syscalls or access the filesystem,
// it relies on libuv to do that, and then the result or error 
// is propagated via Node to v8. Also depending on the api used
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
	result, err  := t.Execute()
	t.resolve = result
	t.reject = err
	appendToQueue(rt.promiseQ, t)
}

// wrapPromise executes an AsyncIO bound task ie whose meta type 
// is AsyncIO. The results are appended to the IOQueue to be 
// resolved by the mainCaller
func (rt *Runtime) nodeWrapPromise(ctx context.Context, t *Task) {
	// keep track of goroutines doing io-bound tasks
	// although im not sure yet, do the correspoding resolves, do we 
	// need to keep track of them?
	rt.inflight.Add(1)
	done := make(chan *result)
	childCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go execAsyncIO(childCtx, t.Execute, done)
	go func() {
		select {
		case <-ctx.Done():
			return
		case res := <-done:
			t.reject = res.err
			t.resolve = res.success
			appendToQueue(rt.promiseQ, t)
			rt.inflight.Add(-1)
		}
	}()
}
