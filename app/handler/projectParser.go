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

func parseProject(projectRoot string) []model.LogType {
	// TODO: parse project and create log types
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

	//call helper function for each file in each pkg
	for _, file := range filesToParse {
		logTypes = append(logTypes, findLogsInFile(file, projectRoot)...)
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

func findLogsInFile(path string, base string) []model.LogType {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		log.Error().Err(err).Msg("unable to parse file")
	}

	logInfo := []model.LogType{}
	logCalls := []fnStruct{}

	//Filter out nodes that do not contain a call to Msg or Msgf
	//then call the recursive function isFromLog to determine
	//if these Msg* calls originated from a log statement to eliminate
	//false positives
	ast.Inspect(node, func(n ast.Node) bool {

		//The following two blocks are related to finding variables
		//and their values

		//These follow pattern "name :=/= value"
		asn, ok := n.(*ast.AssignStmt)
		if ok {
			fmt.Println("Assigned", asn.Lhs, asn.Rhs)
		}

		//filter nodes that represent variable assignments,
		//collect this information for reference later
		//These nodes follow pattern "var/const name = value"
		expr, ok := n.(*ast.GenDecl)
		if ok {
			spec, ok := expr.Specs[0].(*ast.ValueSpec)
			if ok {
				for _, a := range spec.Values {
					switch v := a.(type) {

					//this case catches string literals
					case *ast.BasicLit:
						fmt.Println("Declared", spec.Names, v.Value)

					default:
						fmt.Println("type", reflect.TypeOf(a), a)
					}
					// fmt.Println("Value: ",  v)
				}
			}
		}

		//The following block is for finding log statements and the
		//values passed to them as args

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

				//Should recursively call a function to check if
				//the preceding SelectorExpressions contain a call
				//to log, which means this is most
				//definitely a log statement
				if strings.Contains(val, "Msg") || val == "Err" {
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
		fn, _ := l.fn.Fun.(*ast.SelectorExpr)
		fmt.Printf("Args for %v at line %d\n", fn.Sel, fset.Position(l.n.Pos()).Line)
		for _, a := range l.fn.Args {
			//limits args to literal values and prints them
			switch v := a.(type) {

			//this case catches string literals,
			//our proof-of-concept case
			case *ast.BasicLit:
				// fmt.Println("Basic", v.Value)
				//add the log information to the
				//result array
				relPath, _ := filepath.Rel(base, fset.File(l.n.Pos()).Name())
				currentLog.FilePath = filepath.ToSlash(relPath)
				currentLog.LineNumber = fset.Position(l.n.Pos()).Line
				currentLog.Regex = v.Value[1 : len(v.Value)-1]
				fmt.Printf("%v, ", v.Value)
				logInfo = append(logInfo, currentLog)

			//this case catches composite literals
			case *ast.CompositeLit:
				// fmt.Println("Composite", v.Elts)

			//this case catches statically assigned message values
			//that are declared in same file and not const
			case *ast.Ident:
				if v.Obj != nil {
					val, ok := v.Obj.Decl.(*ast.AssignStmt)
					if ok && val != nil {
						data, ok2 := val.Rhs[0].(*ast.BasicLit)
						if ok2 && data != nil {
							fmt.Printf("%v Assigned: %v, %T\n", val.Lhs[0], data.Value, data.Value)
						} else {
							fmt.Println(val.Lhs, "Assigned:", val.Rhs[0])
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
