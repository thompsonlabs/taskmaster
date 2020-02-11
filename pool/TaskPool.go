package pool

import (
	"taskmaster/models"
)

//TaskPool - Top level TaskPool, allows for tasks to be queued and then subsequently asynchronously processed
//in parrallel using 2 or more worker threads.
type TaskPool interface {
	StartUp()
	ShutDown()
	IsRunning() bool
	SubmitTask(task models.Executable)
	GetNextTask() models.Executable
	GetMaxWorkerCount() int
	GetMaxQueueCount() int
	SetMaxWorkerCount(maxWorkerCount int)
	SetMaxQueueCount(maxQueueCount int)
	IdleWorkersCount() int
	ActiveWorkersCount() int
	AppendWorkers(amount int) int //increments the worker count by the specified amount and returns the total number of workers in the pool.
	Wait(waitTimeInMillis int64)
	SetCustomErrorFunction(func(interface{}))
	GetQueueCount() int
}
