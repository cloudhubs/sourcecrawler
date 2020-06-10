package handler

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"sourcecrawler/app/model"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

//Parse project to create log types
func parseProject(projectRoot string) []model.LogType {

	//Holds a slice of log types
	logTypes := []model.LogType{}

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

	//call helper function to add each file in each pkg
	for _, file := range filesToParse {
		logTypes = append(logTypes, findLogsInFile(file, projectRoot)...)
	}

	// Doesn't do anything right now
	// functList := functionDecls(filesToParse) //gathers list of functions
	// callFrom(functList, filesToParse) //checks each expression call to see if it uses an explicitly declared function

	return logTypes
}

//This is just a struct
//to quickly access members of the nodes.
//Can be changed/removed if desired,
//currently just a placeholder
type fnStruct struct {
	n  ast.Node
	fn *ast.CallExpr
}

//Checks if from log (two.name is Info/Err/Error)
func isFromLog(fn *ast.SelectorExpr) bool {
	if strings.Contains(fmt.Sprint(fn.X), "log") {
		return true
	}
	one, ok := fn.X.(*ast.CallExpr)
	if ok {
		two, ok := one.Fun.(*ast.SelectorExpr)
		if ok {
			return isFromLog(two)
		}
	}
	return false
}

/*
 Determines if a function is called somewhere else based on its name (path and line number)
  -currently goes through all files and finds if it's used
*/
func functionDecls(filesToParse []string) map[string][]string {

	//Map of all function names with a [line number, file path]
	// ex: ["HandleMessage" : {"45":"insights-results-aggregator/consumer/processing.go"}]
	//They key is the function name. Go doesn't support function overloading -> so each name will be unique
	functMap := map[string][]string{}

	//Inspect each file for calls to this function
	for _, file := range filesToParse {
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			log.Error().Err(err).Msg("Error parsing file " + file)
		}

		//Grab package name - needed to prevent duplicate function names across different packages, keep colon
		packageName := node.Name.Name + ":"
		fmt.Print("Package name is " + packageName + " at line ")
		fmt.Println(fset.Position(node.Pos()).Line)

		//Inspect AST for file
		//TODO: handle duplicate function names if they're in different packages
		ast.Inspect(node, func(currNode ast.Node) bool {

			fdNode, ok := currNode.(*ast.FuncDecl)
			if ok {
				//package name is appended to separate diff functions across packages
				functionName := packageName + fdNode.Name.Name
				linePos := strconv.Itoa(fset.Position(fdNode.Pos()).Line)
				fpath, _ := filepath.Abs(fset.File(fdNode.Pos()).Name())

				//Add the data to the function list
				data := []string{linePos, fpath}
				functMap[functionName] = data
			}

			return true
		})
	}

	return functMap
}

/*
	Check the location (file + line number) of where a function is used (this might be a helper function)
*/
func callFrom(funcList map[string][]string, filesToParse []string) {

	for _, file := range filesToParse {
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			log.Error().Err(err).Msg("Error parsing file " + file)
		}

		//Keep track of package name
		packageName := node.Name.Name + ":"
		//fmt.Print("Package name is " + packageName + " at line ")
		//fmt.Println(fset.Position(node.Pos()).Line)

		//Inspect the AST, starting with call expressions
		ast.Inspect(node, func(currNode ast.Node) bool {
			callExprNode, ok := currNode.(*ast.CallExpr)
			if ok {
				//Filter single function calls such as parseMsg(msg.Value)
				functionName := packageName + fmt.Sprint(callExprNode.Fun)
				if val, found := funcList[functionName]; found {
					fmt.Println("The function " + functionName + " was found on line " + val[0] + " in " + val[1])
				}
			}
			return true
		})
	}

	//if val, found := funcList[funcName]; found {
	//	fmt.Println("The function " + funcName + " was found on line " + val[0] + " in " + val[1])
	//}
}

func findLogsInFile(path string, base string) []model.LogType {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		log.Error().Err(err).Msg("unable to parse file")
	}

	logInfo := []model.LogType{}
	logCalls := []fnStruct{}

	//Helper structure to hold logTypes with function types (msg, msgf, err)
	//lnTypes := make(map[model.LogType]string)

	//Filter out nodes that do not contain a call to Msg or Msgf
	//then call the recursive function isFromLog to determine
	//if these Msg* calls originated from a log statement to eliminate
	//false positives
	ast.Inspect(node, func(n ast.Node) bool {

		//continue if Node casts as a CallExpr
		ret, ok := n.(*ast.CallExpr)
		if ok {
			//continue processing if CallExpr casts
			//as a SelectorExpr
			fn, ok := ret.Fun.(*ast.SelectorExpr)
			if ok {
				// fmt.Printf("%T, %v\n", fn, fn)
				//convert Selector into String for comparison
				val := fmt.Sprint(fn.Sel)

				//fmt.Println("Val: " + val)

				//Should recursively call a function to check if
				//the preceding SelectorExpressions contain a call
				//to log, which means this is most
				//definitely a log statement
				if strings.Contains(val, "Msg") {
					if isFromLog(fn) {
						logCalls = append(logCalls, fnStruct{n, ret})
					}
				}
			}
		}
		return true
	})

	//AT THIS POINT
	//all log messages should be collected, the next section
	//demonstrates extracting the string arguments from each
	//log isntance

	//This block is for accessing the argument
	//values of a selector function so we can see
	//what was added by the inspection above
	for _, l := range logCalls {
		currentLog := model.LogType{}
		// fn, _ := l.fn.Fun.(*ast.SelectorExpr)
		// fmt.Printf("Args for %v at line %d\n", fn.Sel, fset.Position(l.n.Pos()).Line)
		for _, a := range l.fn.Args {

			//limits args to literal values and prints them
			switch v := a.(type) {

			//this case catches string literals,
			//our proof-of-concept case
			case *ast.BasicLit:
				// fmt.Println("Basic", v.Value)
				//add the log information to the
				//result array

				//relPath, _ := filepath.Rel(base, fset.File(l.n.Pos()).Name()) //TODO: filepath isn't showing up?
				//currentLog.FilePath = filepath.ToSlash(relPath)

				currentLog.FilePath, _ = filepath.Abs(fset.File(l.n.Pos()).Name())
				currentLog.LineNumber = fset.Position(l.n.Pos()).Line

				//Regex value currently
				reg := v.Value

				//Converting current regex strings to regex format (parenthesis, %d,%s,%v,',%+v)
				if strings.Contains(reg, "(") {
					reg = strings.ReplaceAll(reg, "(", "\\(")
				}
				if strings.Contains(reg, ")") {
					reg = strings.ReplaceAll(reg, ")", "\\)")
				}

				//Converting %d, %s, %v to regex num, removing single quotes
				if strings.Contains(reg, "%d") {
					reg = strings.ReplaceAll(reg, "%d", "\\d")
				}
				if strings.Contains(reg, "%s") {
					reg = strings.ReplaceAll(reg, "%s", ".*")
				}
				if strings.Contains(reg, "%v") {
					reg = strings.ReplaceAll(reg, "%v", ".*")
				}
				if strings.Contains(reg, "'") {
					reg = strings.ReplaceAll(reg, "'", "")
				}
				if strings.Contains(reg, "%+v") {
					reg = strings.ReplaceAll(reg, "%+v", ".+")
				}

				//Remove the double quotes
				currentLog.Regex = reg[1 : len(reg)-1]

				logInfo = append(logInfo, currentLog)

			//this case catches composite literals
			case *ast.CompositeLit:
				// fmt.Println("Composite", v.Elts)

			//this case catches statically assigned message values
			//that are not const
			case *ast.Ident:
				if v.Obj != nil {
					val, ok := v.Obj.Decl.(*ast.AssignStmt)
					if ok && val != nil {
						data, ok2 := val.Rhs[0].(*ast.BasicLit)
						if ok2 && data != nil {
							// fmt.Printf("%v Assigned: %v, %T\n", val.Lhs[0], data.Value, data.Value)
						} else {
							// fmt.Println(val.Lhs, "Assigned:", val.Rhs[0])
						}
					}
				}
			default:
				fmt.Println("type", reflect.TypeOf(a), a)
			}
		}
		fmt.Println()
	}

	return logInfo
}
