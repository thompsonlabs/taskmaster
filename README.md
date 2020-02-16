# taskmaster
[![GoDoc](https://godoc.org/github.com/thompsonlabs/taskmaster?status.png)](https://godoc.org/github.com/thompsonlabs/taskmaster)
[![MIT License](https://img.shields.io/apm/l/atomic-design-ui.svg?)](https://godoc.org/github.com/thompsonlabs/taskmaster)
[![Donate](https://img.shields.io/badge/$-support-ff69b4.svg?style=flat)](https://paypal.me/ThompsonLabsUK?locale.x=en_GB)  


A robust, flexible (goroutine based) Thread Pool for Go based on the worker-thread pattern. 

## Overview

TaskMaster is a flexible, managed concurrent ThreadPool implementation for Go. **Three distinct pool implementation variants** are currently
made available via the bundled TaskMaster PoolBuilder, each with their own respective benefits and trade-offs which may be breifly sumarised as follows:

#### FixedTaskPool

```go
taskmaster.Builder().NewFixedTaskPool()
```
Creates a new Fixed TaskPool. A FixedTaskPool starts up with a fixed Worker count (as specified at the time the pool is built) and from that point forward maintains Its Worker count at this constant value until such time that the pool is explitly shutdown or the period specified to the Wait() method elapses (which ever occurs first).

*    FixedTaskPools are suitable for use cases where there is some esitimate of the load the
     pool is likely to be expected to accomadate and where fast execution is required since
     there will always be fully initialised Workers ready to service submitted requests and thus
     zero overhead related to the dynamic spawning of new Workers on-demand.

*    Conversely, where minimising resource consumption is a key goal then one of the alternative
     pool implementations in this package may be a better fit.


#### CachedTaskPool

```go
taskmaster.Builder().NewCachedTaskPool(maxCachePeriodInMillis int64)
```
Creates a new Cached TaskPool. A CachedTaskPool initially starts up with a Worker pool count of zero and then proceeds to dynamically scale up its Workers to accomadate the submission of new tasks as necessary. Where possible the pool will **always** seek to use existing (cached) Workers to service newly submitted tasks; Workers ONLY remain in the pool for as long as absoutely necessary and are promptly evicted after **maxCachePeriodInMillis** duration specified to the constructor elapses, which may well result in the pool count returning to zero following periods of prolonged inactivity.

*    CachedTaskPool is a suitable choice for use cases where minimisation of system resource
     consumption is paramount as Worker Threads are ONLY created and indeed retained in the pool for as
     long as they are required.

*    Where maximum performance is a key factor, one of the alternative pool implementations in this package may be a better
     fit as there will generally be some degree of overhead associated with the spawning of new Workers on-demand,
     this will be especially true in cases where the **maxCachePeriodInMillis** parameter is incorrectly set resulting in
     Cached Workers being evicted from the pool prematurely.


#### ElasticTaskPool

```go
taskmaster.Builder(),NewElasticTaskPool(maxCachePeriodInMillis int64, minWorkerCount int)
```
Creates a new Elastic TaskPool. An Elastic TaskPool dynamically scales up **and** down to accomadate the submittal of tasks for execution. On instantiation a min and max Worker pool value is specified and the pool will then expand and contract between these values respectively in line with the load its required to handle. Idle Workers in the pool over and above the specified minimum will be automatically evicted in due course as part of the pools aforementioned contraction process.

*    ElasticTaskPool is a happy medium between the other TaskPool varients as it allows for there to **always** be some amount 
     of dedicated pool Worker resources on-hand to immediately service tasks and provides the ability to scale up (and back down) from there as necessary.

*    Where there is an expectation that there will be really low or really high throughput demands (especially where the delivery will be sporadic
     in nature) then one of the alternative pool implementations in this package may be a better fit.

Please see the full package [Documentation](https://godoc.org/github.com/thompsonlabs/taskmaster) for further information.


## Installation

Use the `go` command

```
$ go get github.com/thompsonlabs/taskmaster
```

## Example

Here a basic (and hopefully intuitive) example is provided to quickly get users started using TaskMaster. Please note a more comprehensive **interactive** example
may be found in the **example package** please do also view the project [Documentation](https://godoc.org/github.com/thompsonlabs/taskmaster) for further details.

```go

package main

import(

"fmt"
"github.com/thompsonlabs/taskmaster"
"time"
"strconv"

)


func main(){

/** 
   utilise the PoolBuilder to build and configure a new TaskPool instance
   here we have elected to use the ElasticTaskPool varient but the process
   is exactly the same for other pool implementations as they are all returned
   from builder as the abstract TaskPool type.
 */

//build a taskpool according to your requirements
var taskpool = taskmaster.
                Builder().
                NewElasticTaskPool(60000, 5).
                SetMaxQueueCount(50).
                SetMaxWorkerCount(20).
                Build()

//startup taskpool
taskpool.StartUp()

//submit a 30 test tasks to the pool for execution (the TestTask struct is defined below)
for i := 0; i < 30; i++ {

  taskpool.SubmitTask(NewTestTask(i))

}


//wait on the pool for the 15 seconds to allow time for all tasks to execute a wait
//value of 0 may be specified to wait indefinetely. Where 0 IS specified the pool
//should be started from ANOTHER Go Routine to avoid the Main Thread blocking indefinitely
//"taskPool.ShutDown()" may then subsequently be explictly called, from the Main thread, to
//shutdown the pool at some point in the future
taskpool.Wait(10000)


fmt.Println("Wait period elapsed; the Pool has been successfully, automatically shutdown.")
}





//TestTask - A test task created for the purposes of this example. The task
//           implements the requisite "Executable" interface (containing a single Execute() function)
//           and simply prints a string to the console on execution.
type TestTask struct {
        exeIdx int
}

//NewTestTask Creates a and returns a new TestTask instance.
func NewTestTask(anIndex int) *TestTask {

        return &TestTask{

                exeIdx: anIndex}
}


//Execute - Overrriden from the Executable interface; here is where the operation
//          the TaskPool is required to run should be defined.
func (tt *TestTask) Execute() {

        //to test the pools panic recovery
        if tt.exeIdx == 7 {

                panic("7 Index is not allowed.")
        }

        //sleep to simulate some time taken to complete the task
        time.Sleep(time.Millisecond * 3000)

        //print success status to the console.
        fmt.Println("Task: " + strconv.Itoa(tt.exeIdx) + " Successfully executed")
} 

```

## Documentation

[Documentation](https://godoc.org/github.com/thompsonlabs/taskmaster) is hosted at the GoDoc project.


## Copyright

Copyright (C) 2019-2020 by ThompsonLabs | [thompsonlabs@outlook.com](mailto:thompsonlabs@outlook.com)

TaskMaster package released under MIT Licence. See [LICENSE](https://github.com/thompsonlabs/taskmaster/blob/master/LICENSE) for further details.





















