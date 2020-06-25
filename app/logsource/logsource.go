package logsource

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
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
								regex = CreateRegex(v.Value)
							}
						}

						//stop
						return false
					}
				}
			}
		}
		return true
	})
	return regex
}

func CreateRegex(value string) string {
	//Regex value currently
	reg := value

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
	return reg[1 : len(reg)-1]
}