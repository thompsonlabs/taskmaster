package taskmaster

import (
	"github.com/thompsonlabs/taskmaster/pool"
)

var poolBuilderInstance *PoolBuilder

//PoolBuilder - Builds a new TaskMaster TaskPool
type PoolBuilder struct {
	maxWorkerCount         int
	maxQueueCount          int
	poolType               PoolType
	customErrorFunction    func(interface{})
	maxCachePeriodInMillis int64
	minWorkerCount         int
}

//NewFixedTaskPool - Create a new Fixed TaskPool
func (tmpb *PoolBuilder) NewFixedTaskPool() *PoolBuilder {

	tmpb.resetValues()
	tmpb.poolType = FIXED

	return tmpb
}

//NewCachedTaskPool - Create a new Cached TaskPool
func (tmpb *PoolBuilder) NewCachedTaskPool(maxCachePeriodInMillis int64) *PoolBuilder {

	tmpb.resetValues()
	tmpb.poolType = CACHED
	tmpb.maxCachePeriodInMillis = maxCachePeriodInMillis
	return tmpb
}

//NewElasticTaskPool - Creates a new Elastic TaskPool
func (tmpb *PoolBuilder) NewElasticTaskPool(maxCachePeriodInMillis int64, minWorkerCount int) *PoolBuilder {

	tmpb.resetValues()
	tmpb.poolType = ELASTIC
	tmpb.maxCachePeriodInMillis = maxCachePeriodInMillis
	tmpb.minWorkerCount = minWorkerCount
	return tmpb
}

//SetMaxWorkerCount  - Set the resulting TaskPool's Max Worker Count; defaults to 10
//                     and each worker occupies its own Go Routine.
func (tmpb *PoolBuilder) SetMaxWorkerCount(maxWorkerCount int) *PoolBuilder {

	tmpb.maxWorkerCount = maxWorkerCount
	return tmpb
}

//SetMaxQueueCount  - Set the resulting TaskPool's Max Queue Count; this is the max number of tasks that may be
//                    queued for execution and defaults to 100.
func (tmpb *PoolBuilder) SetMaxQueueCount(maxQueueCount int) *PoolBuilder {

	tmpb.maxQueueCount = maxQueueCount
	return tmpb
}

//SetCustomErrorFunction - Allows a custom, developer-defined error function to be associated
//                         with the pool.When specified a call will be made to this function
//                         each time a Pool worker (or more specifically the go routine its associated with)
//                         encounter an unrecoverable error (i.e a panic)
func (tmpb *PoolBuilder) SetCustomErrorFunction(errorFunction func(interface{})) *PoolBuilder {

	tmpb.customErrorFunction = errorFunction
	return tmpb
}

//Build - Builds a new TaskPool using the settings supplied to the builder.
func (tmpb *PoolBuilder) Build() pool.TaskPool {

	var aTaskPool pool.TaskPool

	if tmpb.poolType == FIXED {

		aTaskPool = pool.NewFixedTaskPool()

	} else if tmpb.poolType == CACHED {

		aTaskPool = pool.NewCachedTaskPool(tmpb.maxCachePeriodInMillis)

	} else {

		aTaskPool = pool.NewElasticTaskPool(tmpb.maxCachePeriodInMillis, tmpb.minWorkerCount)

	}

	aTaskPool.SetMaxQueueCount(tmpb.maxQueueCount)
	aTaskPool.SetMaxWorkerCount(tmpb.maxWorkerCount)
	aTaskPool.SetCustomErrorFunction(tmpb.customErrorFunction)

	return aTaskPool
}

func (tmpb *PoolBuilder) resetValues() {

	tmpb.maxQueueCount = 100
	tmpb.maxWorkerCount = 10
	tmpb.maxCachePeriodInMillis = 0
	tmpb.minWorkerCount = 0
	tmpb.customErrorFunction = nil
}

//Builder - Returns a singular reference to the TaskMasterBuilder.
func Builder() *PoolBuilder {

	if poolBuilderInstance == nil {

		poolBuilderInstance = new(PoolBuilder)
	}

	return poolBuilderInstance
}

//PoolType - A group of constants.
type PoolType int

const (
	//FIXED -A FIXED POOL
	FIXED PoolType = iota
	//CACHED - A CACHE POOL
	CACHED
	//ELASTIC - AN ELASTIC POOL
	ELASTIC
)

func (poolType PoolType) String() string {
	return [...]string{"FIXED", "CACHED", "ELASTIC"}[poolType]
}
