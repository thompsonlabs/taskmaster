package models

//Executable - Models a single executable that may be passed into a task queue.
type Executable interface {

	//Execute - Execute interface method, this will be overriden by implementers of this interface
	//          to include logic that is required to be ran on the in a TaskQueue thread.
	Execute()
}
