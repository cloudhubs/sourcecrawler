package test

import (
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"sourcecrawler/app/cfg"
	"testing"
)

func TestExFile(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "example.go", nil, parser.ParseComments)
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

	condStmts := make(map[ast.Node]cfg.ExecutionLabel)
	vars := make([]ast.Node, 0)

	path := cfg.CreateNewPath()
	leaves := cfg.GetLeafNodes(w)
	for _, leaf := range leaves {
		path.TraverseCFG(leaf, condStmts, vars, w, make(map[string]ast.Node))
	}

	for _, expr := range path.Expressions {
		printer.Fprint(os.Stdout, fset, expr)
		t.Log(expr)
	}

}
