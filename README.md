# taskmaster
A robust, flexible (goroutine based) Thread Pool for Go based on the worker-thread pattern. 

## Overview

TaskMaster is a flexible, managed concurrent ThreadPool implementation for Go. *Three distinct pool implementation variants* are currently
made available via the bundled TaskMaster PoolBuilder, each with their own respective benefits and tradeoffs which may be breifly sumarised as follows:

#### FixedTaskPool

```
taskmaster.Builder().NewFixedTaskPool()
```
Creates a new Fixed TaskPool. A FixedTaskPool starts up with a fixed Worker count (as specified at the time the pool is built) and from that point forward maintains Its Worker count at this constant value until such time that the pool is explitly shutdown or the period specified to the Wait() method elapses (which ever occurs first).

*    FixedTaskPools are perfect for use cases where there is some esitimate of the load the
     pool is likely to be expected to accomadate and where fast execution is required since
     there will always be fully initialised Workers ready to service submitted requests and thus
     zero overhead related to the dynamic spwaning of new Workers on-demand.

*    Conversely, where minimising resource consumption is a key goal then one of the alternative
     pool implementations may be a better fit for your needs.








