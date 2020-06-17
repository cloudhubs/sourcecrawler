package logsource

import (
	"fmt"
	"go/ast"
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
