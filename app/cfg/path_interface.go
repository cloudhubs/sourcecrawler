package cfg

import (
	"fmt"
	"go/ast"
)

// ---- Represents a possible execution path --------
type Path struct {
	Stmts       map[ast.Node]ExecutionLabel //Conditional statements
	Expressions []ast.Node		//List of all expressions
	Variables   []ast.Node      //AssignStmt, ValueSpec, IncDecStmt, Ident
	ExecStatus	ExecutionLabel  //Should indicate if path was executed
}

//List of paths
type PathList struct {
	Paths []Path
	SsaInts map[string]int
	Regexes		[]string		//List of all regex strings in the paths
}

//Adds a path to the list
func (paths *PathList) AddNewPath(path Path) {
	paths.Paths = append(paths.Paths, path)
	//pathList.Paths = append(pathList.Paths, path)
}

//Instantiates a new instance of a path list
func CreateNewPath() *PathList {
	return &PathList{
		Paths:   make([]Path, 0),
		SsaInts: make(map[string]int),
	}
}

//Resets and clears out paths in the list
func (paths *PathList) ClearPath() {
	paths.Paths = make([]Path, 0)
}

//Gets the list of paths inside the pathlist
func (paths *PathList) GetExecPath() []Path {
	return paths.Paths
}

//Debug print
func (paths *PathList) PrintExecPath() {

	if len(paths.Paths) == 0 {
		fmt.Println("Empty path list")
		return
	}

	for _, path := range paths.Paths {
		//fmt.Println("Path is", path)
		//Print statements
		for key, value := range path.Stmts {
			fmt.Printf("Stmt: %v - %s ", key, value)
			for _, varNode := range path.Variables {
				fmt.Printf("| Vars: (%v)", varNode)
			}
			fmt.Println()
		}

	}
}
