package helper

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"

	"github.com/rs/zerolog/log"
)

//Struct for quick access to the function declaration nodes
type fdeclStruct struct {
	node     ast.Node
	fd       *ast.FuncDecl
	filePath string
	lineNum  string
	Name     string
}

//Gathers all go files to parse
func GatherGoFiles(projectRoot string) []string {
	filesToParse := []string{}
	//gather all go files in project
	filepath.Walk(projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("prevent panic by handling failure accessing a path %q: %v\n", path, err)
			return err
		}
		if filepath.Ext(path) == ".go" {
			fullpath, err := filepath.Abs(path)
			if err != nil {
				fmt.Println(err)
			}
			filesToParse = append(filesToParse, fullpath)
		}
		return nil
	})

	return filesToParse
}

//Struct for quick access to the function declaration nodes
type FdeclStruct struct {
	Node     ast.Node
	Fd       *ast.FuncDecl
	FilePath string
	LineNum  string
	Name     string
}

/*
 Determines if a function is called somewhere else based on its name (path and line number)
  -currently goes through all files and finds if it's used
*/
func FunctionDeclsMap(filesToParse []string) map[string][]string {

	//Map of all function names with a [line number, file path]
	// ex: ["HandleMessage" : {"45":"insights-results-aggregator/consumer/processing.go"}]
	//They key is the function name. Go doesn't support function overloading -> so each name will be unique
	functMap := map[string][]string{}
	functCalls := []FdeclStruct{}

	//Inspect each file for calls to this function
	for _, file := range filesToParse {
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			log.Error().Err(err).Msg("Error parsing file " + file)
		}

		//Grab package name - needed to prevent duplicate function names across different packages, keep colon
		//packageName := node.Name.Name + ":"

		//Inspect AST for explicit function declarations
		ast.Inspect(node, func(currNode ast.Node) bool {
			fdNode, ok := currNode.(*ast.FuncDecl)
			if ok {
				//package name is appended to separate diff functions across packages
				functionName := fdNode.Name.Name
				linePos := strconv.Itoa(fset.Position(fdNode.Pos()).Line)
				fpath, _ := filepath.Abs(fset.File(fdNode.Pos()).Name())

				//Add the data to the function list
				data := []string{linePos, fpath}
				functMap[functionName] = data

				//Add astNode and the FuncDecl node to the function calls
				functCalls = append(functCalls, FdeclStruct{
					currNode,
					fdNode,
					fpath,
					linePos,
					functionName,
				})
			}
			return true
		})

		//Inspect the AST Call Expressions (where they call a function)
		//ast.Inspect(node, func(currNode ast.Node) bool {
		//	callExprNode, ok := currNode.(*ast.CallExpr)
		//	if ok {
		//		//Filter single function calls such as parseMsg(msg.Value)
		//		//functionName := packageName + fmt.Sprint(callExprNode.Fun)
		//		functionName := packageName
		//		if _, found := functMap[functionName]; found {
		//			//fmt.Println("The function " + functionName + " was found on line " + val[0] + " in " + val[1])
		//			//fmt.Println("")
		//		}
		//	}
		//	return true
		//})
	}

	return functMap
}
