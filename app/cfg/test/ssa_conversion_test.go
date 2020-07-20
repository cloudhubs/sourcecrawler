package test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"sourcecrawler/app/cfg"
	"testing"
	_"fmt"
)

func TestExFile(t *testing.T) { //NOTE: missing some numbers before var names, missing displaying assignStmts/IncDecStmts?
	fileName := "example.go"
	// file2 := "testunsafe.go"
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, fileName, nil, parser.ParseComments)
	if err != nil {
		t.Error(err)
		return
	}

	var w *cfg.FnWrapper
	ast.Inspect(f, func(node ast.Node) bool {
		if fn, ok := node.(*ast.FuncDecl); ok {
			if w == nil {
				w = cfg.NewFnWrapper(fn, make([]ast.Expr, 0))
			}
		}
		return true
	})

	if w != nil {
		w.Fset = fset
		w.ASTs = []*ast.File{f}
		cfg.ExpandCFG(w, make([]*cfg.FnWrapper, 0))
	}

	// condStmts := make(map[ast.Node]cfg.ExecutionLabel)
	// vars := make([]ast.Node, 0)
	exprs := make([]ast.Node, 0)

	paths := cfg.CreateNewPath()
	leaves := cfg.GetLeafNodes(w)
	for _, leaf := range leaves {
		paths.TraverseCFG(leaf, exprs, w, make(map[string]ast.Node))
	}

	cnt := 1
	for _, path := range paths.Paths {
		fmt.Println("---------- PATH", cnt, " -------------")
		cnt++
		for _, expr := range path.Expressions {
			printer.Fprint(os.Stdout, fset, expr)
			fmt.Println()
		}
		fmt.Println()
		// t.Log(expr)
	}

}
