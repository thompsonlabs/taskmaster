package pool

import (
	"fmt"
	"sync"
	"taskmaster/models"
	"time"
)

//FixedTaskPool Default implementation of a TaskPool
type FixedTaskPool struct {
	running             bool
	maxWorkerCount      int
	maxQueueCount       int
	taskQueue           []models.Executable
	taskQueueLock       sync.Mutex
	workers             []Worker
	workersLock         sync.Mutex
	waitGroup           sync.WaitGroup
	customErrorFunction func(interface{})
}

//NewFixedTaskPool Creates new DefaultTaskPool instance.
func NewFixedTaskPool() *FixedTaskPool {

	return &FixedTaskPool{
		running:        false,
		maxWorkerCount: 10,
		maxQueueCount:  100,
		taskQueue:      make([]models.Executable, 0),
		workers:        make([]Worker, 0)}
}

//StartUp - Starts the threadpool up and places it in a "ready-state" to begin accepting tasks.
func (ftp *FixedTaskPool) StartUp() {

	ftp.taskQueueLock.Lock()

	for i := 0; i < ftp.maxWorkerCount; i++ {

		//create new worker
		worker := newFixedTaskPoolWorker(ftp, &ftp.waitGroup)

		//append it to our collection
		ftp.workers = append(ftp.workers, worker)

		//increment the wait group count.
		ftp.waitGroup.Add(1)

		//start the worker in alternate thread
		go worker.start()

	}

	ftp.taskQueueLock.Unlock()

	ftp.running = true

}

//ShutDown - ShutsDown the entire pool. All currently executing tasks will be permitted to complete
//           their respective operations before the pool is gracefully shutdown additionally
//           no further tasks will be accepted for execution following a call to this routine.
func (ftp *FixedTaskPool) ShutDown() {

	ftp.workersLock.Lock()
	defer ftp.workersLock.Unlock()

	for _, curWorker := range ftp.workers {

		curWorker.stop()
	}

	ftp.taskQueue = ftp.taskQueue[:0]
	ftp.workers = ftp.workers[:0]

	ftp.running = false

}

func (ftp *FixedTaskPool) getTaskQueue() *[]models.Executable {

	return &ftp.taskQueue
}

//IsRunning - Returns a boolean flag which indicates whether or not this TaskPool instance is
//            currently running, will return TRUE where this is the case and FALSE otherwise.
func (ftp *FixedTaskPool) IsRunning() bool {

	return ftp.running
}

//SubmitTask - Queues a task for asynchronous execution in an alternate thread/go-routine.
func (ftp *FixedTaskPool) SubmitTask(task models.Executable) {

	ftp.taskQueueLock.Lock()
	ftp.taskQueue = append(ftp.taskQueue, task)
	ftp.taskQueueLock.Unlock()

}

//GetNextTask - Removes and returns the next task in the queue, if there are no
//              tasks in the queue nil is returned.
func (ftp *FixedTaskPool) GetNextTask() models.Executable {

	ftp.taskQueueLock.Lock()
	defer ftp.taskQueueLock.Unlock()

	if len(ftp.taskQueue) == 0 {

		return nil
	}

	//we always remove from then the zero-most index to
	//insure tasks are removed on FIFO basis
	return ftp.removeFromTaskQueue(0)

}

//GetMaxWorkerCount - Gets the max number of tasks this TaskPool is configured to run simultaneously.
func (ftp *FixedTaskPool) GetMaxWorkerCount() int {

	return ftp.maxWorkerCount
}

//GetMaxQueueCount - Gets the maximum amount of tasks this TaskPool is configured to queue.
func (ftp *FixedTaskPool) GetMaxQueueCount() int {

	return ftp.maxQueueCount
}

//SetMaxWorkerCount - Sets the max number of tasks this TaskPool is permitted to run simultaneously.
func (ftp *FixedTaskPool) SetMaxWorkerCount(maxWorkerCount int) {

	ftp.maxWorkerCount = maxWorkerCount
}

//SetMaxQueueCount - Sets the maximum amount of tasks this TaskPool is permitted to queue.
func (ftp *FixedTaskPool) SetMaxQueueCount(maxQueueCount int) {

	ftp.maxQueueCount = maxQueueCount
}

//IdleWorkersCount - Returns the number of idle workers in the queue (i,e workers waiting for a task to execute.)
func (ftp *FixedTaskPool) IdleWorkersCount() int {

	ftp.workersLock.Lock()
	defer ftp.workersLock.Unlock()

	idleCount := 0
	for _, curWorker := range ftp.workers {

		if curWorker.isIdle() {

			idleCount++
		}
	}

	return idleCount
}

//ActiveWorkersCount - Returns the number of active workers in the queue (i,e workers currently executing a task)
func (ftp *FixedTaskPool) ActiveWorkersCount() int {

	ftp.workersLock.Lock()
	defer ftp.workersLock.Unlock()

	activeCount := 0
	for _, curWorker := range ftp.workers {

		if !curWorker.isIdle() {

			activeCount++
		}
	}

	return activeCount

}

//AppendWorkers - Appends the specified number of workers to this pool.
func (ftp *FixedTaskPool) AppendWorkers(amount int) int {

	ftp.workersLock.Lock()
	defer ftp.workersLock.Unlock()

	for i := 0; i < amount; i++ {

		newWorker := newFixedTaskPoolWorker(ftp, &ftp.waitGroup)
		ftp.workers = append(ftp.workers, newWorker)
		ftp.waitGroup.Add(1)
		go newWorker.start()

	}

	return len(ftp.workers)

}

//Wait - Causes the current go routine to wait on this taskpool until the specified
//       wait time has elapsed; a wait time value of 0 may be specified to wait indefinitely.
func (ftp *FixedTaskPool) Wait(waitTimeInMillis int64) {

	if waitTimeInMillis > 0 {

		ftp.launchDelayedPoolShutdown(waitTimeInMillis)
	}

	ftp.waitGroup.Wait()
}

//GetQueueCount - Returns the number of outstanding tasks in the queue
func (ftp *FixedTaskPool) GetQueueCount() int {

	return len(ftp.taskQueue)
}

//SetCustomErrorFunction - Allows a developer-defined custom error function to be
//                         associated with the taskpool instance. This function will
//                         be called each time a taskpool worker encounters an error
//                         whilst processing a task.
func (ftp *FixedTaskPool) SetCustomErrorFunction(customErrorFunction func(interface{})) {

	ftp.customErrorFunction = customErrorFunction
}

//private helper functions

func (ftp *FixedTaskPool) removeFromTaskQueue(indexOfEntry int) models.Executable {

	tq := &ftp.taskQueue
	removedEntry := (*tq)[indexOfEntry] //grab reference to the entry before its removed.
	copy((*tq)[indexOfEntry:], (*tq)[indexOfEntry+1:])
	(*tq)[len(*tq)-1] = nil
	(*tq) = (*tq)[:len((*tq))-1]
	return removedEntry

}

func (ftp *FixedTaskPool) launchDelayedPoolShutdown(timeInMillis int64) {

	shutdownTimer := time.NewTimer(time.Millisecond * time.Duration(timeInMillis))
	go func() {
		<-shutdownTimer.C
		ftp.ShutDown()
	}()

}

//FixedTaskPoolWorker implementation
type FixedTaskPoolWorker struct {
	pool          *FixedTaskPool
	idle          bool
	started       bool
	currentTaskID string
	keepPooling   bool
	poolWaitGroup *sync.WaitGroup
}

//factory method.
func newFixedTaskPoolWorker(poolRef *FixedTaskPool, poolWaitGroup *sync.WaitGroup) *FixedTaskPoolWorker {

	return &FixedTaskPoolWorker{
		pool:          poolRef,
		idle:          true,
		started:       false,
		keepPooling:   false,
		poolWaitGroup: poolWaitGroup}
}

func (ftpw *FixedTaskPoolWorker) isIdle() bool {

	return ftpw.idle
}
func (ftpw *FixedTaskPoolWorker) start() {

	defer ftpw.poolWaitGroup.Done()

	ftpw.keepPooling = true

	ftpw.poolTaskQueue()

	ftpw.started = true
}
func (ftpw *FixedTaskPoolWorker) stop() {

	ftpw.keepPooling = false

	ftpw.started = false
}
func (ftpw *FixedTaskPoolWorker) isStarted() bool {

	return ftpw.started
}

func (ftpw *FixedTaskPoolWorker) poolTaskQueue() {

	for {

		if !ftpw.keepPooling {

			if !ftpw.isIdle() {

				ftpw.idle = true
			}

			break
		}

		executableTask := ftpw.pool.GetNextTask()

		if executableTask == nil {

			time.Sleep(time.Millisecond * 500)

		} else {

			ftpw.idle = false
			ftpw.executeTask(executableTask)
			ftpw.idle = true

		}

	}

}

func (ftpw *FixedTaskPoolWorker) executeTask(executableTask models.Executable) {

	defer ftpw.recoverFunc() //where execution of the programmer supplied task fails we would like to recover the GoRoutine
	executableTask.Execute()

}

func (ftpw *FixedTaskPoolWorker) recoverFunc() {

	if r := recover(); r != nil {

		//call custom error function where one has been defined
		if ftpw.pool.customErrorFunction != nil {

			ftpw.pool.customErrorFunction(r)

		} else {
			fmt.Println("recovered from ", r)
		}
	}
}
