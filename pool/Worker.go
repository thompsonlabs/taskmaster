package pool

//Worker - Responsible for processing tasks in the queue.
type Worker interface {
	isIdle() bool
	start()
	stop()
	isStarted() bool
}
