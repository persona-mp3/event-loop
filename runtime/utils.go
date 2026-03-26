package runtime

// Where possible, incase of early returns we'll need
// to clear all other queues, but while this doesn't seem
// neccessary at the moment, it's good to have in mind
func cleanUp() {
	logger.Println("cleaning up")
}

func execTasks(tasks []*Task) error {
	for _, t := range tasks {
		result, err := t.Execute()
		if err != nil {
			errLogger.Printf("err executing func: %s\n", t.Id)
			errLogger.Printf("%3s\n", err)
			return err
		}

		logger.Println("what are you doing?")

		logger.Println("result from func: ", t.Id)
		logger.Printf("%s\n\n", result)
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
