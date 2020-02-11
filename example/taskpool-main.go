package main

import (
	"bufio"
	"fmt"
	"github.com/thompsonlabs/taskmaster/pool"
	"os"
	"strconv"
	"time"
)

var taskpool pool.TaskPool

func main() {

	//taskpool = pool.NewFixedTaskPool()
	//taskpool = pool.NewCachedTaskPool(60000)
	//taskpool = pool.NewElasticTaskPool(60000, 5)
	taskpool = taskmaster.
		Builder().
		NewElasticTaskPool(60000, 5).
		SetMaxQueueCount(50).
		SetMaxWorkerCount(20).
		SetCustomErrorFunction(MyCustomErrorFunction)
	Build()

	go launchTaskPool(30)

	readUserInput(taskpool)
}

func launchTaskPool(initialTaskCount int) {

	fmt.Println("Launching taskpool... ")

	taskpool.StartUp()

	for i := 0; i < initialTaskCount; i++ {

		taskpool.SubmitTask(newMyExecutable(i))
	}

	//wait the specified time on the pool 0 may be entered to wait indefinitely
	//taskpool.Wait()
	taskpool.Wait(30000)

	fmt.Println("Pool has been successfully shutdown.")

}

func readUserInput(pool pool.TaskPool) {

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {

		readText := scanner.Text()

		if readText == "quit" {

			break
		} else if readText == "new" {

			pool.SubmitTask(newMyExecutable(110))

		} else if readText == "stats" {

			fmt.Println("Queue count: " + strconv.Itoa(pool.GetQueueCount()))

			fmt.Println("Idle worker count: " + strconv.Itoa(pool.IdleWorkersCount()))

			fmt.Println("Active worker count: " + strconv.Itoa(pool.ActiveWorkersCount()))

		} else if readText == "shutdown" {

			pool.ShutDown()

		} else if readText == "new7" {

			for i := 0; i < 7; i++ {

				pool.SubmitTask(newMyExecutable(110))

			}

		}

		fmt.Println(scanner.Text())
	}
}

type myExecutable struct {
	exeIdx int
}

func newMyExecutable(anIndex int) *myExecutable {

	return &myExecutable{

		exeIdx: anIndex}
}

func (me *myExecutable) Execute() {

	//to test the pools panic recovery
	if me.exeIdx == 7 {

		panic("7 Index is not allowed.")
	}

	time.Sleep(time.Millisecond * 3000)
	fmt.Println("Task: " + strconv.Itoa(me.exeIdx) + " Successfully executed")
}

func MyCustomErrorFunction(panicError interface{}) {

	fmt.Println("Error log from MyCustomErrorFunction: ", panicError)
}
