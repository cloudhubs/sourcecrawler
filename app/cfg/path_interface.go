package cfg

import (
	"fmt"
	"go/ast"
)

// ---- Represents a possible execution path --------
type Path struct {
	Stmts     []string
	Variables []ast.Node //*ast.AssignStmt or *ast.ValueSpec

}

//List of paths
type PathList struct {
	Paths []Path
}
var pathInstance *PathList = nil

//Adds a path to the list
func (p *PathList) AddNewPath(path Path){
	p.Paths = append(p.Paths, path)
	//pathList.Paths = append(pathList.Paths, path)
}

//Instantiates a new instance of a path list
func CreateNewPath() *PathList {
	if pathInstance == nil{
		pathInstance = new(PathList)
	}
	return pathInstance
}

//Resets and clears out paths in the list
func (p *PathList) ClearPath(){
	p.Paths = make([]Path, 0)
}

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
		for _, value := range path.Stmts{
			fmt.Printf("Stmt: %s ", value)
			for _, vars := range path.Variables{
				fmt.Printf("| Vars: (%v)", vars)
			}
			fmt.Println()
		}

	}
}
