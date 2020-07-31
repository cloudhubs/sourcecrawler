package cfg

import (
	"fmt"
	"go/ast"
	"reflect"
)

// ---- Represents a possible execution path --------
type Path struct {
	Expressions []ast.Node                  //List of all expressions (includes conditions and vars) (slice used b/c of ordering issue with maps)
	ExecStatus	[]ExecutionLabel			//Parallel array with Expressions
	Stmts       map[ast.Node]ExecutionLabel 
	DidExecute	ExecutionLabel  			//Determine if an entire branch has not executed (based on absence of log stmts)
	CopyExpressions []ast.Node
	CopyExecStatus []ExecutionLabel
}

//List of paths
type PathList struct {
	Paths   []Path
	SsaInts map[string]int
	Regexes []string //List of all regex strings in the paths
	//StkTrcInfo	[]handler.StackTraceStruct
}

//Adds a path to the list
func (p *PathList) AddNewPath(path Path) {
	do := true
	for _, aPath := range p.Paths {
		if reflect.DeepEqual(aPath, path) {
			do = false
			break
		}
	}
	if do {
		p.Paths = append(p.Paths, path)
	}
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

	// for _, path := range paths.Paths {
	// 	//fmt.Println("Path is", path)
	// 	//Print statements
	// 	for key, value := range path.Stmts {
	// 		fmt.Printf("Stmt: %v - %s ", key, value)
	// 		for _, varNode := range path.Variables {
	// 			fmt.Printf("| Vars: (%v)", varNode)
	// 		}
	// 		fmt.Println()
	// 	}

	// }
}
