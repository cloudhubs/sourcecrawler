package test

import (
	"fmt"
	_ "fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"sourcecrawler/app/cfg"
	"testing"
)

func testUtil(t *testing.T, fileName string) {
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
	// var exprs []ast.Node

	paths := cfg.CreateNewPath()
	leaves := cfg.GetLeafNodes(w)
	for _, leaf := range leaves {
		paths.TraverseCFG(leaf, w)
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

func TestExFile(t *testing.T) { 
	testUtil(t, "example.go")
}

func TestReassignment(t *testing.T) {
	testUtil(t, "example_reassignment.go")
}
