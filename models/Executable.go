package models

//Executable - Developer-defined tasks MUST implment this interface in order to
//             to be executed via a TaskPool thread.
type Executable interface {

	//Execute - Execute interface method, this will be overriden by implementers of this interface
	//          to include logic that is required to be ran in a TaskPool thread.
	Execute()
}
