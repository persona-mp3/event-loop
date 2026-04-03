package runtime

import "log"

// Where possible, incase of early returns we'll need
// to clear all other queues, but while this doesn't seem
// neccessary at the moment, it's good to have in mind
func cleanUp(rt *Runtime) {
	logger.Println("cleaning up DBUEGGG")
	// I think this the problem for the third time
	// When we error out, we just head straight out for 
	// the function...but this is something I'd need to diagnose

}

func execTasks(tasks []*Task) error {
	if len(tasks) == 0 {
		return nil
	}
	for _, t := range tasks {
		result, err := t.Execute()
		if err != nil {
			errLogger.Printf("err executing func: %s\n", t.Id)
			errLogger.Printf("%3s\n", err)
			return err
		}

		// logger.Println("result from func: ", t.Id)
		logger.Println()
		logger.Printf("\n=======================%s=======================\n", t.Id)
		logger.Printf("%s\n==============================================\n", result)
	}
	return nil
}

func execPromises(promises []*Task) error {
	for _, task := range promises {
		result, err := task.resolve, task.reject
		if err != nil {
			errLogger.Printf("reject: while executing func: %s\n", task.Id)
			errLogger.Printf("%3s\n", err)
			return err
		}

		logger.Println("promise resolved for func: ", task.Id)
		logger.Printf("%s\n\n", result)
	}
	return nil
}

func appendToQueue(q *queue, t *Task) {
	q.mu.Lock()
	q.tasks = append(q.tasks, t)
	q.mu.Unlock()
}

func (rt *Runtime) debugInfo() {
	log.SetFlags(log.Lshortfile)

	log.Printf(`
	[ STACK - INFO ]
  inflight_routines: %d
  stack_queue_len: %d
  next_ticker_queue_len: %d
  promise_queue_len: %d
  
  
  ================== QUEUES ========================= 
  1. Stack Queue: %+v
  
  2. Next Ticker Queue:  %+v
  
  3. Promise Queue: %+v

  4. Timer Queue: %+v
  
	`,
		rt.inflight.Load(), len(rt.stack.tasks), len(rt.nextTickerQ.tasks),
		len(rt.promiseQ.tasks), rt.stack.tasks, rt.nextTickerQ.tasks, rt.promiseQ.tasks, rt.timerQ.tasks,
	)
}
