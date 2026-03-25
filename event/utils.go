package event

func appendToQueue(q *queue, p *Process) {
	q.mu.Lock()
	q.processes = append(q.processes, p)
	q.mu.Unlock()
}

// func (rt *Runtime) snapshot(q *queue) []*Process {
// 	var snapshot []*Process
// 	q.mu.Lock()
// 	for _, p := range q.processes {
// 		snapshot = append(snapshot, p)
// 	}
// 	// empty the stack
// 	q.processes = nil
// 	q.mu.Unlock()
// 	return snapshot
// }
