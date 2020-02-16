package pool

import (
	"fmt"
	"github.com/thompsonlabs/taskmaster/models"
	"sync"
	"time"
)

//ElasticTaskPool - An Elastic TaskPool dynamically scales up AND down to accomadate the submital of tasks for execution. On instantiation a min and max Worker pool value is specified and the pool will then expand and contract to these values respectively in line with the load its required to handle. Idle Workers in the pool over and above the specified minimum will be automatically evicted in due course as part of the pools aforementioned contraction process.
type ElasticTaskPool struct {
	running              bool
	maxWorkerCount       int
	maxQueueCount        int
	taskQueue            []models.Executable
	taskQueueLock        sync.Mutex
	workers              *[]Worker
	workersLock          sync.Mutex
	waitGroup            sync.WaitGroup
	maxCachePeriodMillis int64
	minWorkerCount       int
	customErrorFunction  func(interface{})
}

//NewElasticTaskPool Creates new DefaultTaskPool instance.
func NewElasticTaskPool(maxCachePeriodInMillis int64, minWorkerCount int) *ElasticTaskPool {

	wkrs := make([]Worker, 0)

	return &ElasticTaskPool{
		running:              false,
		maxWorkerCount:       10,
		maxQueueCount:        100,
		taskQueue:            make([]models.Executable, 0),
		workers:              &wkrs,
		maxCachePeriodMillis: maxCachePeriodInMillis,
		minWorkerCount:       minWorkerCount}
}

//StartUp - Starts the threadpool up and places it in a "ready-state" to begin accepting tasks.
func (etp *ElasticTaskPool) StartUp() {

	//we add addtional val to the waitGroup to ensure
	//that the pool waitGroup DOES NOT exit should all
	//go routines return. This is necessary as we would
	//like to dynamically add workers according to the
	//submitted tasks in this TaskPool implementation.
	//NOTE: The shutdown method of this taskpool implementation WILL ALWAYS
	//      decrement the waitGroup count by 1 to account for this.
	etp.waitGroup.Add(1)

	etp.running = true

}

//ShutDown - ShutsDown the entire pool. All currently executing tasks will be permitted to complete their respective operations before the pool is gracefully shutdown additionally no further tasks will be accepted for execution following a call to this routine.
func (etp *ElasticTaskPool) ShutDown() {

	etp.waitGroup.Add(-1)
	etp.workersLock.Lock()
	defer etp.workersLock.Unlock()

	for _, curWorker := range *etp.workers {

		curWorker.stop()
	}

	etp.taskQueue = etp.taskQueue[:0]
	workerVar := (*etp.workers)[:0]
	*(etp.workers) = workerVar

	etp.running = false

}

func (etp *ElasticTaskPool) getTaskQueue() *[]models.Executable {

	return &etp.taskQueue
}

//IsRunning - Returns a boolean flag which indicates whether or not this TaskPool instance is currently running, will return TRUE where this is the case and FALSE otherwise.
func (etp *ElasticTaskPool) IsRunning() bool {

	return etp.running
}

//SubmitTask - Queues a task for asynchronous execution in an alternate thread/go-routine. A bool is returned to indicate whether or not the task was successfully submitted to the pool.
func (etp *ElasticTaskPool) SubmitTask(task models.Executable) bool {

	appended := false

	etp.taskQueueLock.Lock()

	if len(etp.taskQueue) < etp.maxQueueCount {

		etp.taskQueue = append(etp.taskQueue, task)

		appended = true

		if (len(etp.taskQueue) > etp.IdleWorkersCount()) && len(*etp.workers)+1 <= etp.maxWorkerCount {

			etp.AppendWorkers(1)

		}
	}
	etp.taskQueueLock.Unlock()

	return appended

}

//GetNextTask - Removes and returns the next task in the queue, if there are no tasks in the queue nil is returned.
func (etp *ElasticTaskPool) GetNextTask() models.Executable {

	etp.taskQueueLock.Lock()
	defer etp.taskQueueLock.Unlock()

	if len(etp.taskQueue) == 0 {

		return nil
	}

	//we always remove from then the zero-most index to
	//insure tasks are removed on FIFO basis
	return etp.removeFromTaskQueue(0)

}

//GetMaxWorkerCount - Gets the max number of tasks this TaskPool is configured to run simultaneously.
func (etp *ElasticTaskPool) GetMaxWorkerCount() int {

	return etp.maxWorkerCount
}

//GetMaxQueueCount - Gets the maximum amount of tasks this TaskPool is configured to queue.
func (etp *ElasticTaskPool) GetMaxQueueCount() int {

	return etp.maxQueueCount
}

//SetMaxWorkerCount - Sets the max number of tasks this TaskPool is permitted to run simultaneously.
func (etp *ElasticTaskPool) SetMaxWorkerCount(maxWorkerCount int) {

	etp.maxWorkerCount = maxWorkerCount
}

//SetMaxQueueCount - Sets the maximum amount of tasks this TaskPool is permitted to queue.
func (etp *ElasticTaskPool) SetMaxQueueCount(maxQueueCount int) {

	etp.maxQueueCount = maxQueueCount
}

//IdleWorkersCount - Returns the number of idle workers in the queue (i,e workers waiting for a task to execute.)
func (etp *ElasticTaskPool) IdleWorkersCount() int {

	etp.workersLock.Lock()
	defer etp.workersLock.Unlock()

	idleCount := 0
	for _, curWorker := range *etp.workers {

		if curWorker.isIdle() {

			idleCount++
		}
	}

	return idleCount
}

//ActiveWorkersCount - Returns the number of active workers in the queue (i,e workers currently executing a task)
func (etp *ElasticTaskPool) ActiveWorkersCount() int {

	etp.workersLock.Lock()
	defer etp.workersLock.Unlock()

	activeCount := 0
	for _, curWorker := range *etp.workers {

		if !curWorker.isIdle() {

			activeCount++
		}
	}

	return activeCount

}

//AppendWorkers - Appends the specified number of workers to this pool.
func (etp *ElasticTaskPool) AppendWorkers(amount int) int {

	etp.workersLock.Lock()
	defer etp.workersLock.Unlock()

	for i := 0; i < amount; i++ {

		newWorker := newElasticTaskPoolWorker(etp, &etp.waitGroup)
		*etp.workers = append(*etp.workers, newWorker)
		etp.waitGroup.Add(1)
		go newWorker.start()

	}

	return len(*etp.workers)

}

//Wait - Causes the current go routine to wait on this taskpool until the specified wait time has elapsed; a wait time value of 0 may be specified to wait indefinitely.
func (etp *ElasticTaskPool) Wait(waitTimeInMillis int64) {

	if waitTimeInMillis > 0 {

		etp.launchDelayedPoolShutdown(waitTimeInMillis)
	}

	etp.waitGroup.Wait()
}

//GetQueueCount - Returns the number of outstanding tasks in the queue
func (etp *ElasticTaskPool) GetQueueCount() int {

	return len(etp.taskQueue)
}

//RemoveWorker - Attemps to remove the specified worker from the internal queue. Remove will ONLY be permitted to go ahead where doing so will not result in fewer then the specified minimum number of threads left remining in the pool.
func (etp *ElasticTaskPool) RemoveWorker(aWorker Worker) {

	etp.workersLock.Lock()

	var targetIndex int
	targetIndex = -1

	for idx, curWorker := range *etp.workers {

		if aWorker == curWorker {

			targetIndex = idx

			break
		}
	}

	if targetIndex > -1 {

		etp.removeFromWorkerQueueByIndex(targetIndex)
	}
	etp.workersLock.Unlock()
}

//MaxCachePeriodMillis - The max amount of time in millies that a worker may remain idle in the pool
func (etp *ElasticTaskPool) MaxCachePeriodMillis() int64 {

	return etp.maxCachePeriodMillis
}

//GetMinWorkerCount - The minimum amount of workers that should be present in this taskpool instance.
func (etp *ElasticTaskPool) GetMinWorkerCount() int {

	return etp.minWorkerCount
}

//SetCustomErrorFunction - Allows a developer-defined custom error function to be
//                         associated with the taskpool instance. This function will
//                         be called each time a taskpool worker encounters an error
//                         whilst processing a task.
func (etp *ElasticTaskPool) SetCustomErrorFunction(customErrorFunction func(interface{})) {

	etp.customErrorFunction = customErrorFunction
}

//private helper functions

func (etp *ElasticTaskPool) removeFromTaskQueue(indexOfEntry int) models.Executable {

	tq := &etp.taskQueue
	removedEntry := (*tq)[indexOfEntry] //grab reference to the entry before its removed.
	copy((*tq)[indexOfEntry:], (*tq)[indexOfEntry+1:])
	(*tq)[len(*tq)-1] = nil
	(*tq) = (*tq)[:len((*tq))-1]
	return removedEntry

}

func (etp *ElasticTaskPool) removeFromWorkerQueueByIndex(indexOfEntry int) {

	tq := etp.workers
	copy((*tq)[indexOfEntry:], (*tq)[indexOfEntry+1:])
	(*tq)[len(*tq)-1] = nil
	(*tq) = (*tq)[:len((*tq))-1]

}

func (etp *ElasticTaskPool) launchDelayedPoolShutdown(timeInMillis int64) {

	shutdownTimer := time.NewTimer(time.Millisecond * time.Duration(timeInMillis))
	go func() {
		<-shutdownTimer.C
		etp.ShutDown()
	}()

}

//ElasticTaskPoolWorker implementation
type ElasticTaskPoolWorker struct {
	pool                  *ElasticTaskPool
	idle                  bool
	started               bool
	currentTaskID         string
	keepPooling           bool
	poolWaitGroup         *sync.WaitGroup
	lastTaskExecutionTime int64
}

//factory method.
func newElasticTaskPoolWorker(poolRef *ElasticTaskPool, poolWaitGroup *sync.WaitGroup) *ElasticTaskPoolWorker {

	return &ElasticTaskPoolWorker{
		pool:                  poolRef,
		idle:                  false,
		started:               false,
		keepPooling:           false,
		poolWaitGroup:         poolWaitGroup,
		lastTaskExecutionTime: 0}
}

func (etpw *ElasticTaskPoolWorker) isIdle() bool {

	return etpw.idle
}
func (etpw *ElasticTaskPoolWorker) start() {

	defer etpw.poolWaitGroup.Done()

	etpw.lastTaskExecutionTime = etpw.nowAsUnixMillis()

	etpw.keepPooling = true

	etpw.started = true

	etpw.poolTaskQueue() //block until max cache time is reached or stop is explicitly called.

	etpw.removeSelfFromWorkerQueue()

}

func (etpw *ElasticTaskPoolWorker) removeSelfFromWorkerQueue() {

	etpw.pool.RemoveWorker(etpw)
}

func (etpw *ElasticTaskPoolWorker) nowAsUnixMillis() int64 {

	return time.Now().UnixNano() / 1e6
}

func (etpw *ElasticTaskPoolWorker) stop() {

	etpw.keepPooling = false

	etpw.started = false
}
func (etpw *ElasticTaskPoolWorker) isStarted() bool {

	return etpw.started
}

func (etpw *ElasticTaskPoolWorker) poolTaskQueue() {

	for {

		if !etpw.keepPooling {

			if !etpw.isIdle() {

				etpw.idle = true
			}

			break
		}

		executableTask := etpw.pool.GetNextTask()

		if executableTask == nil {

			if etpw.maxCacheTimeElapsed() {

				if etpw.poolContainsEnoughWorkersToExit() {

					break
				} else {

					etpw.lastTaskExecutionTime = etpw.nowAsUnixMillis()
				}

			}

			if !etpw.isIdle() {

				etpw.idle = true
			}

			time.Sleep(time.Millisecond * 125)

		} else {

			etpw.idle = false
			etpw.lastTaskExecutionTime = etpw.nowAsUnixMillis()
			etpw.executeTask(executableTask)

		}

	}

}

func (etpw *ElasticTaskPoolWorker) poolContainsEnoughWorkersToExit() bool {

	etpw.pool.workersLock.Lock()
	defer etpw.pool.workersLock.Unlock()
	return (len(*etpw.pool.workers) - 1) >= etpw.pool.GetMinWorkerCount()
}

func (etpw *ElasticTaskPoolWorker) maxCacheTimeElapsed() bool {

	return ((etpw.nowAsUnixMillis() - etpw.lastTaskExecutionTime) > etpw.pool.MaxCachePeriodMillis())
}

func (etpw *ElasticTaskPoolWorker) executeTask(executableTask models.Executable) {

	defer etpw.recoverFunc() //where execution of the programmer supplied task fails we would like to recover the GoRoutine
	executableTask.Execute()

}

func (etpw *ElasticTaskPoolWorker) recoverFunc() {
	if r := recover(); r != nil {

		//call custom error function where one has been defined
		if etpw.pool.customErrorFunction != nil {

			etpw.pool.customErrorFunction(r)
		} else {
			fmt.Println("recovered from ", r)
		}
	}
}
