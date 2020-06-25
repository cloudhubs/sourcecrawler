package logsource

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"sourcecrawler/app/handler"
	"strings"
)

//Checks if from log (two.name is Info/Err/Error)
func IsFromLog(fn *ast.SelectorExpr) bool {
	if strings.Contains(fmt.Sprint(fn.X), "log") {
		return true
	}
	one, ok := fn.X.(*ast.CallExpr)
	if ok {
		two, ok := one.Fun.(*ast.SelectorExpr)
		if ok {
			return IsFromLog(two)
		}
	}
	return false
}

func GetLogRegexFromInfo(filename string, lineNumber int) string{
	fset := token.NewFileSet()
	tk, err := parser.ParseFile(fset,filename, nil, parser.ParseComments)
	if err != nil {
		panic(err)
	}
	var regex string
	ast.Inspect(tk, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
				if IsFromLog(sel){
					if fset.Position(n.Pos()).Line == lineNumber {
						//get log from node
						for _, arg := range call.Args{
							switch v := arg.(type){
							case *ast.BasicLit:
								//create regex
								regex = handler.CreateRegex(v.Value)
							}
						}

						//stop
						return false
					}
					return true
				}
			}
		}
	})
	return regex
}
