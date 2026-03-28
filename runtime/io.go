package runtime

// NOTE the IO queue in node doesn't execute the whole
// function, only the IO Portion, and it hands it back to node
// to execute
/*
	import fs from "node:fs/promises"
		async function reader(){
		1>	const content = await fs.open()
			// normal tasks
			const encode = encoder(content)
		}

		line 1 runs in the ioQueue and the function execution
		is halted
*/
func (rt *Runtime) execIO(t *Task) {
	done := make(chan *result)
	runIO(rt.ctx, t.Execute, done)
	go func() {
		select {
		case <-rt.ctx.Done():
			return
		case res := <-done:
			_ = res
			// appendToQueue(rt.ioQ, t)

		}
	}()
}
