package cfg

import (
	"fmt"
	"go/ast"
)

// ---- Represents a possible execution path --------
type Path struct {
	Stmts       map[ast.Node]ExecutionLabel
	Expressions []ast.Node
}

//List of paths
type PathList struct {
	Paths   []Path
	SsaInts map[string]int
}

//Singleton instance
// var PathInstance *PathList = nil

//Adds a path to the list
func (p *PathList) AddNewPath(path Path) {
	p.Paths = append(p.Paths, path)
}

//Instantiates a new instance of a path list
func CreateNewPath() *PathList {
	return &PathList{
		Paths:   make([]Path, 0),
		SsaInts: make(map[string]int),
	}
}

//Resets and clears out paths in the list
func (p *PathList) ClearPath() {
	p.Paths = make([]Path, 0)
}

//Gets the list of paths inside the pathlist
func (p *PathList) GetExecPath() []Path {
	return p.Paths
}

//Debug print
func (p *PathList) PrintExecPath() {

	if len(p.Paths) == 0 {
		fmt.Println("Empty path list")
		return
	}

	for _, path := range p.Paths {
		for key, value := range path.Stmts {
			fmt.Printf("Stmt: %v - %s ", key, value)
			fmt.Println()
		}

	}
}
