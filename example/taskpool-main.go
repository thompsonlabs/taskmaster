package main

import (
	"bufio"
	"fmt"
	"github.com/thompsonlabs/taskmaster"
	"github.com/thompsonlabs/taskmaster/pool"
	"os"
	"strconv"
	"time"
)

func main() {

	var taskpool = buildTaskPool()

	go launchTaskPool(30, &taskpool)

	readUserInput(taskpool)
}

func buildTaskPool() pool.TaskPool {

	/**m
	   Here we build a task pool according to our requirements
	   for the purposes of this example we elect to build an
	   ElasticTaskPool which allows us to specify as parameters:

	   1)An inactivity timeout value (in milliseconds) for the worker threads in the pool.
	   2)The minimum number of core threads that must be present in the pool these threads
	     WILL NOT be removed even if their respective inactivity timeouts are reached.

		The Builder will return the pool as an abstract TaskPool type irrespetive of the specific
	    TaskPool implementation you choose to build. (see the docs for more info)
	*/

	var taskpool = taskmaster.
		Builder().
		NewElasticTaskPool(60000, 5).
		SetMaxQueueCount(50).
		SetMaxWorkerCount(20).
		SetCustomErrorFunction(TestCustomErrorFunction).
		Build()

	return taskpool
}

func launchTaskPool(numberOfTestTasksToLaunch int, taskpool *pool.TaskPool) {

	fmt.Println("Launching " + strconv.Itoa(numberOfTestTasksToLaunch) + " Test Tasks in the Taskpool... ")

	(*taskpool).StartUp()

	for i := 0; i < numberOfTestTasksToLaunch; i++ {

		(*taskpool).SubmitTask(NewTestTask(i))
	}

	//wait the specified time on the pool; a wait value of 0 may be entered to wait indefinitely
	(*taskpool).Wait(30000)

	fmt.Println("Pool has been successfully shutdown.")

}

func readUserInput(pool pool.TaskPool) {

	/**

	  Here, for the purposes of the test we just read in commands from the
	  standard input (i.e command line) and execute them against the launched
	  pool instance. The supported commands are as follows:
	  1)quit - quit the example program which implicitly shutsdown the taskpool.
	  2)new - submits a new test task to the taskpool.
	  3)stats - displays the current pool stats: queued tasks,idle workers and active workers.
	  4)shutdown - shuts down the launched pool. "quit" will still need to be typed to exit.
	  5)new10 - Adds 10 new test tasks to the taskpool.

	*/

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {

		readText := scanner.Text()

		if readText == "quit" {

			break
		} else if readText == "new" {

			pool.SubmitTask(NewTestTask(110))

		} else if readText == "stats" {

			fmt.Println("Queue count: " + strconv.Itoa(pool.GetQueueCount()))

			fmt.Println("Idle worker count: " + strconv.Itoa(pool.IdleWorkersCount()))

			fmt.Println("Active worker count: " + strconv.Itoa(pool.ActiveWorkersCount()))

		} else if readText == "shutdown" {

			pool.ShutDown()

		} else if readText == "new10" {

			for i := 0; i < 10; i++ {

				pool.SubmitTask(NewTestTask(110))

			}

		}

		fmt.Println(scanner.Text())
	}
}

//TestTask - A test task created for the purposes of this example. The task
//           implements the requisite "Executable" interface (containing a single Execute() function)
//           and simply print a string to the console on execution.
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

//TestCustomErrorFunction - A test custom error function
func TestCustomErrorFunction(panicError interface{}) {

	fmt.Println("Error log from TestCustomErrorFunction: ", panicError)
}
