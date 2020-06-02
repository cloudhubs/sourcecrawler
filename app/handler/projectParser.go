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
		logTypes = append(logTypes, findLogsInFile(file)...)
	}

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

func findLogsInFile(path string) []model.LogType {
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

				//add the log information to the result array
				currentLog.FilePath, _ = filepath.Abs(fset.File(l.n.Pos()).Name())
				currentLog.LineNumber = fset.Position(l.n.Pos()).Line

				//Regex value currently
				reg := v.Value

				//Converting current regex strings to regex format (parenthesis, %d,%s,%v,',%+v)
				if strings.Contains(reg, "("){
					reg = strings.ReplaceAll(reg,"(", "\\(")
				}
				if strings.Contains(reg, ")"){
					reg = strings.ReplaceAll(reg, ")", "\\)")
				}

				//Converting %d, %s, %v to regex num, removing single quotes
				if strings.Contains(reg, "%d"){
					reg = strings.ReplaceAll(reg, "%d", "\\d")
				}
				if strings.Contains(reg, "%s"){
					reg = strings.ReplaceAll(reg, "%s", ".*")
				}
				if strings.Contains(reg, "%v"){
					reg = strings.ReplaceAll(reg, "%v", ".*")
				}
				if strings.Contains(reg, "'"){
					reg = strings.ReplaceAll(reg, "'", "")
				}
				if strings.Contains(reg, "%+v"){
					reg = strings.ReplaceAll(reg, "%+v", ".+")
				}

				currentLog.Regex = reg
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

	//Modify Regex
	//Create regex based on specific log type
	//for index := range logInfo {
	//
	//	lnNumber := logInfo[index].LineNumber
	//	fPath := logInfo[index].FilePath
	//	//TODO: If log type is Msgf - need to build regex for variable string
	//	//For msgf in aggregator.go line 221
	//	if strings.Contains(fPath, "aggregator.go"){
	//		if lnNumber == 221{
	//			logInfo[index].Regex = "database preparation exited with error code \\d+ \\d+"
	//		}
	//	}
	//
	//	if strings.Contains(logInfo[index].FilePath, "aggregator.go") && logInfo[index].LineNumber == 125 {
	//		logInfo[index].Regex = "old DB migration version (current: \\d, latest: \\d)"
	//	}
	//}

	return logInfo
}
