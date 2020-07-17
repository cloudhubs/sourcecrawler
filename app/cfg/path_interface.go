package cfg

import (
	"fmt"
	"go/ast"
)

// ---- Represents a possible execution path --------
type Path struct {
	Stmts     map[ast.Node]ExecutionLabel
	Variables []ast.Node
	//Variables map[ast.Node]string //*ast.AssignStmt or *ast.ValueSpec
	//Stmts []string
}

//List of paths
type PathList struct {
	Paths       []Path
	Expressions []ast.Node
	// Expressions map[ast.Node]string //Temporary, may be subject to change
	SsaInts		map[*ast.Object]int
}

//Singleton instance
// var PathInstance *PathList = nil

//Adds a path to the list
func (p *PathList) AddNewPath(path Path) {
	p.Paths = append(p.Paths, path)
	//pathList.Paths = append(pathList.Paths, path)
}

//Instantiates a new instance of a path list
func CreateNewPath() *PathList {
	return &PathList{
		Paths: make([]Path,0),
		Expressions: make([]ast.Node,0), 
		SsaInts: make(map[*ast.Object]int),
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

//Debug print expressions
func (p *PathList) PrintExpressions() {
	if len(p.Expressions) == 0 {
		fmt.Println("Empty list of expressions")
		return
	}

	fmt.Print("PathList Expressions: ")
	for _, val := range p.Expressions {
		fmt.Print(val, " | ")
	}
	fmt.Println()
}
