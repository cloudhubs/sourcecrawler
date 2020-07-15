package cfg

import (
	"fmt"
	"go/ast"
)

// ---- Represents a possible execution path --------
type Path struct {
	Stmts     map[string]ExecutionLabel
	//Variables map[ast.Node]string //*ast.AssignStmt or *ast.ValueSpec
	//Stmts []string
	Variables []ast.Node
}

//List of paths
type PathList struct {
	Paths []Path
	Expressions map[ast.Expr]string //Temporary, may be subject to change
}

//Singleton instance
var PathInstance *PathList = nil

//Adds a path to the list
func (p *PathList) AddNewPath(path Path){
	p.Paths = append(p.Paths, path)
	//pathList.Paths = append(pathList.Paths, path)
}

//Instantiates a new instance of a path list
func CreateNewPath() *PathList {
	if PathInstance == nil{
		PathInstance = new(PathList)
		PathInstance.Expressions = make(map[ast.Expr]string)
		PathInstance.Paths = make([]Path, 0)
	}
	return PathInstance
}

//Resets and clears out paths in the list
func (p *PathList) ClearPath(){
	p.Paths = make([]Path, 0)
}

//Gets the list of paths inside the pathlist
func (p *PathList) GetExecPath() []Path {
	return p.Paths
}

//Debug print
func (p *PathList) PrintExecPath(){

	if len(p.Paths) == 0 {
		fmt.Println("Empty path list")
		return
	}

	for _, path := range p.Paths{
		//fmt.Println("Path is", path)
		//Print statements
		for key, value := range path.Stmts{
			fmt.Printf("Stmt: %v - %s ", key, value)
			for _, varNode := range path.Variables{
				fmt.Printf("| Vars: (%v)", varNode)
			}
			fmt.Println()
		}

	}
}
