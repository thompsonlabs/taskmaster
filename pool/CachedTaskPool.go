package pool

import (
	"fmt"
	"github.com/thompsonlabs/taskmaster/models"
	"sync"
	"time"
)

//CachedTaskPool - A CachedTaskPool initially starts up with a Worker pool count of zero and
//                 then proceeds to dynamically scales up its Workers to accomadate the submission
//                 of new tasks as necessary. Where possible the pool will ALWAYS seek to use existing
//                 (cached) Workers to service newly submitted tasks; Workers ONLY remain
//                 in the pool for as long as absoutely necessary and are promptly evicted after
//                 "maxCachePeriodInMillis" duration specified to the constructor elapses, which may
//                 well result in the pool count returning to zero following periods of prolonged inactivity.
//
//                 CachedTaskPool is the perfect choice for use cases where minimisation of system resource
//                 consumption is paramount as Worker Threads are ONLY created and indeed retained in the pool for as
//                 long as they are required and not a moment longer.
type CachedTaskPool struct {
	running              bool
	maxWorkerCount       int
	maxQueueCount        int
	taskQueue            []models.Executable
	taskQueueLock        sync.Mutex
	workers              *[]Worker
	workersLock          sync.Mutex
	waitGroup            sync.WaitGroup
	maxCachePeriodMillis int64
	customErrorFunction  func(interface{})
}

//NewCachedTaskPool Creates new DefaultTaskPool instance.
func NewCachedTaskPool(maxCachePeriodInMillis int64) *CachedTaskPool {

	wkrs := make([]Worker, 0)

	return &CachedTaskPool{
		running:              false,
		maxWorkerCount:       10,
		maxQueueCount:        100,
		taskQueue:            make([]models.Executable, 0),
		workers:              &wkrs,
		maxCachePeriodMillis: maxCachePeriodInMillis}
}

//StartUp - Starts the threadpool up and places it in a "ready-state" to begin accepting tasks.
func (ctp *CachedTaskPool) StartUp() {

	//we add addtional val to the waitGroup to ensure
	//that the pool waitGroup DOES NOT exit should all
	//go routines return. This is necessary as we would
	//like to dynamically add workers according to the
	//submitted tasks in this TaskPool implementation.
	//NOTE: The shutdown method of this taskpool implementation WILL ALWAYS
	//      decrement the waitGroup count by 1 to account for this.
	ctp.waitGroup.Add(1)

	ctp.running = true

}

//ShutDown - ShutsDown the entire pool. All currently executing tasks will be permitted to complete
//           their respective operations before the pool is gracefully shutdown additionally
//           no further tasks will be accepted for execution following a call to this routine.
func (ctp *CachedTaskPool) ShutDown() {

	ctp.waitGroup.Add(-1)
	ctp.workersLock.Lock()
	defer ctp.workersLock.Unlock()

	for _, curWorker := range *ctp.workers {

		curWorker.stop()
	}

	ctp.taskQueue = ctp.taskQueue[:0]
	workerVar := (*ctp.workers)[:0]
	*(ctp.workers) = workerVar

	ctp.running = false

}

func (ctp *CachedTaskPool) getTaskQueue() *[]models.Executable {

	return &ctp.taskQueue
}

//IsRunning - Returns a boolean flag which indicates whether or not this TaskPool instance is
//            currently running, will return TRUE where this is the case and FALSE otherwise.
func (ctp *CachedTaskPool) IsRunning() bool {

	return ctp.running
}

//SubmitTask - Queues a task for asynchronous execution in an alternate thread/go-routine.
func (ctp *CachedTaskPool) SubmitTask(task models.Executable) {

	ctp.taskQueueLock.Lock()

	ctp.taskQueue = append(ctp.taskQueue, task)

	if (len(ctp.taskQueue) > ctp.IdleWorkersCount()) && len(*ctp.workers)+1 <= ctp.maxWorkerCount {

		ctp.AppendWorkers(1)

	}

	ctp.taskQueueLock.Unlock()

}

//GetNextTask - Removes and returns the next task in the queue, if there are no
//              tasks in the queue nil is returned.
func (ctp *CachedTaskPool) GetNextTask() models.Executable {

	ctp.taskQueueLock.Lock()
	defer ctp.taskQueueLock.Unlock()

	if len(ctp.taskQueue) == 0 {

		return nil
	}

	//we always remove from then the zero-most index to
	//insure tasks are removed on FIFO basis
	return ctp.removeFromTaskQueue(0)

}

//GetMaxWorkerCount - Gets the max number of tasks this TaskPool is configured to run simultaneously.
func (ctp *CachedTaskPool) GetMaxWorkerCount() int {

	return ctp.maxWorkerCount
}

//GetMaxQueueCount - Gets the maximum amount of tasks this TaskPool is configured to queue.
func (ctp *CachedTaskPool) GetMaxQueueCount() int {

	return ctp.maxQueueCount
}

//SetMaxWorkerCount - Sets the max number of tasks this TaskPool is permitted to run simultaneously.
func (ctp *CachedTaskPool) SetMaxWorkerCount(maxWorkerCount int) {

	ctp.maxWorkerCount = maxWorkerCount
}

//SetMaxQueueCount - Sets the maximum amount of tasks this TaskPool is permitted to queue.
func (ctp *CachedTaskPool) SetMaxQueueCount(maxQueueCount int) {

	ctp.maxQueueCount = maxQueueCount
}

//IdleWorkersCount - Returns the number of idle workers in the queue (i,e workers waiting for a task to execute.)
func (ctp *CachedTaskPool) IdleWorkersCount() int {

	ctp.workersLock.Lock()
	defer ctp.workersLock.Unlock()

	idleCount := 0
	for _, curWorker := range *ctp.workers {

		if curWorker.isIdle() {

			idleCount++
		}
	}

	return idleCount
}

//ActiveWorkersCount - Returns the number of active workers in the queue (i,e workers currently executing a task)
func (ctp *CachedTaskPool) ActiveWorkersCount() int {

	ctp.workersLock.Lock()
	defer ctp.workersLock.Unlock()

	activeCount := 0
	for _, curWorker := range *ctp.workers {

		if !curWorker.isIdle() {

			activeCount++
		}
	}

	return activeCount

}

//AppendWorkers - Appends the specified number of workers to this pool.
func (ctp *CachedTaskPool) AppendWorkers(amount int) int {

	ctp.workersLock.Lock()
	defer ctp.workersLock.Unlock()

	for i := 0; i < amount; i++ {

		newWorker := newCachedTaskPoolWorker(ctp, &ctp.waitGroup)
		*ctp.workers = append(*ctp.workers, newWorker)
		ctp.waitGroup.Add(1)
		go newWorker.start()

	}

	return len(*ctp.workers)

}

//Wait - Causes the current go routine to wait on this taskpool until the specified
//       wait time has elapsed; a wait time value of 0 may be specified to wait indefinitely.
func (ctp *CachedTaskPool) Wait(waitTimeInMillis int64) {

	if waitTimeInMillis > 0 {

		ctp.launchDelayedPoolShutdown(waitTimeInMillis)
	}

	ctp.waitGroup.Wait()
}

//GetQueueCount - Returns the number of outstanding tasks in the queue
func (ctp *CachedTaskPool) GetQueueCount() int {

	return len(ctp.taskQueue)
}

//RemoveWorker - Attemps to remove the specified worker from the internal queue.
func (ctp *CachedTaskPool) RemoveWorker(aWorker Worker) {

	ctp.workersLock.Lock()
	var targetIndex int
	targetIndex = -1

	for idx, curWorker := range *ctp.workers {

		if aWorker == curWorker {

			targetIndex = idx

			break
		}
	}

	if targetIndex > -1 {

		ctp.removeFromWorkerQueueByIndex(targetIndex)
	}
	ctp.workersLock.Unlock()
}

//MaxCachePeriodMillis - The max amount of time in millies that a worker may remain idle in the pool
func (ctp *CachedTaskPool) MaxCachePeriodMillis() int64 {

	return ctp.maxCachePeriodMillis
}

//SetCustomErrorFunction - Allows a developer-defined custom error function to be
//                         associated with the taskpool instance. This function will
//                         be called each time a taskpool worker encounters an error
//                         whilst processing a task.
func (ctp *CachedTaskPool) SetCustomErrorFunction(customErrorFunction func(interface{})) {

	ctp.customErrorFunction = customErrorFunction
}

//private helper functions

func (ctp *CachedTaskPool) removeFromTaskQueue(indexOfEntry int) models.Executable {

	tq := &ctp.taskQueue
	removedEntry := (*tq)[indexOfEntry] //grab reference to the entry before its removed.
	copy((*tq)[indexOfEntry:], (*tq)[indexOfEntry+1:])
	(*tq)[len(*tq)-1] = nil
	(*tq) = (*tq)[:len((*tq))-1]
	return removedEntry

}

func (ctp *CachedTaskPool) removeFromWorkerQueueByIndex(indexOfEntry int) {

	tq := ctp.workers
	copy((*tq)[indexOfEntry:], (*tq)[indexOfEntry+1:])
	(*tq)[len(*tq)-1] = nil
	(*tq) = (*tq)[:len((*tq))-1]

}

func (ctp *CachedTaskPool) launchDelayedPoolShutdown(timeInMillis int64) {

	shutdownTimer := time.NewTimer(time.Millisecond * time.Duration(timeInMillis))
	go func() {
		<-shutdownTimer.C
		ctp.ShutDown()
	}()

}

//CachedTaskPoolWorker implementation
type CachedTaskPoolWorker struct {
	pool                  *CachedTaskPool
	idle                  bool
	started               bool
	currentTaskID         string
	keepPooling           bool
	poolWaitGroup         *sync.WaitGroup
	lastTaskExecutionTime int64
}

//factory method.
func newCachedTaskPoolWorker(poolRef *CachedTaskPool, poolWaitGroup *sync.WaitGroup) *CachedTaskPoolWorker {

	return &CachedTaskPoolWorker{
		pool:                  poolRef,
		idle:                  false,
		started:               false,
		keepPooling:           false,
		poolWaitGroup:         poolWaitGroup,
		lastTaskExecutionTime: 0}
}

func (ctpw *CachedTaskPoolWorker) isIdle() bool {

	return ctpw.idle
}
func (ctpw *CachedTaskPoolWorker) start() {

	defer ctpw.poolWaitGroup.Done()

	ctpw.lastTaskExecutionTime = ctpw.nowAsUnixMillis()

	ctpw.keepPooling = true

	ctpw.started = true

	ctpw.poolTaskQueue() //block until max cache time is reached or stop is explicitly called.

	ctpw.removeSelfFromWorkerQueue()

}

func (ctpw *CachedTaskPoolWorker) removeSelfFromWorkerQueue() {

	ctpw.pool.RemoveWorker(ctpw)
}

func (ctpw *CachedTaskPoolWorker) nowAsUnixMillis() int64 {

	return time.Now().UnixNano() / 1e6
}

func (ctpw *CachedTaskPoolWorker) stop() {

	ctpw.keepPooling = false

	ctpw.started = false
}
func (ctpw *CachedTaskPoolWorker) isStarted() bool {

	return ctpw.started
}

func (ctpw *CachedTaskPoolWorker) poolTaskQueue() {

	for {

		if !ctpw.keepPooling {

			if !ctpw.isIdle() {

				ctpw.idle = true
			}

			break
		}

		executableTask := ctpw.pool.GetNextTask()

		if executableTask == nil {

			if ctpw.maxCacheTimeElapsed() {

				break
			}

			if !ctpw.isIdle() {

				ctpw.idle = true
			}

			time.Sleep(time.Millisecond * 125)

		} else {

			ctpw.idle = false
			ctpw.lastTaskExecutionTime = ctpw.nowAsUnixMillis()
			ctpw.executeTask(executableTask)

		}

	}

}

func (ctpw *CachedTaskPoolWorker) maxCacheTimeElapsed() bool {

	return ((ctpw.nowAsUnixMillis() - ctpw.lastTaskExecutionTime) > ctpw.pool.MaxCachePeriodMillis())
}

func (ctpw *CachedTaskPoolWorker) executeTask(executableTask models.Executable) {

	defer ctpw.recoverFunc() //where execution of the programmer supplied task fails we would like to recover the GoRoutine
	executableTask.Execute()

}

func (ctpw *CachedTaskPoolWorker) recoverFunc() {
	if r := recover(); r != nil {

		//call custom error function where one has been defined
		if ctpw.pool.customErrorFunction != nil {

			ctpw.pool.customErrorFunction(r)
		} else {
			fmt.Println("recovered from ", r)
		}
	}
}
