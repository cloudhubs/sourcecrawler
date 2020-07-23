package helper

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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

/*
 Determines if a function is called somewhere else based on its name (path and line number)
  -currently goes through all files and finds if it's used
*/
func functionDeclsMap(filesToParse []string) map[string][]string {

	//Map of all function names with a [line number, file path]
	// ex: ["HandleMessage" : {"45":"insights-results-aggregator/consumer/processing.go"}]
	//They key is the function name. Go doesn't support function overloading -> so each name will be unique
	functMap := map[string][]string{}
	functCalls := []fdeclStruct{}

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
				functCalls = append(functCalls, fdeclStruct{
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

/*
 Determines if a function is called somewhere else based on its name (path and line number)
  -currently goes through all files and finds if it's used
*/
func findFunctionNodes(filesToParse []string) []fdeclStruct {

	//Map of all function names with a [line number, file path]
	// ex: ["HandleMessage" : {"45":"insights-results-aggregator/consumer/processing.go"}]
	//They key is the function name. Go doesn't support function overloading -> so each name will be unique
	functCalls := []fdeclStruct{}

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

				//Add astNode and the FuncDecl node to the function calls
				functCalls = append(functCalls, fdeclStruct{
					currNode,
					fdNode,
					fpath,
					linePos,
					functionName,
				})
			}
			return true
		})
	}

	return functCalls
}

//Helper function to find origin of function (not used but may need later)
func findFuncOrigin(name string, funcDecList []fdeclStruct) {
	for _, value := range funcDecList {
		if name == value.fd.Name.Name {
			fmt.Println(name, value.filePath, value.lineNum)
		}
	}
}

//Takes in a file name + function name -> returns AST FuncDecl node
func getFdASTNode(fileName string, functionName string, stackFuncNodes []fdeclStruct) *ast.FuncDecl {
	for _, val := range stackFuncNodes {
		if strings.Contains(val.filePath, fileName) && functionName == val.Name {
			return val.fd
		}
	}

	return nil
}

