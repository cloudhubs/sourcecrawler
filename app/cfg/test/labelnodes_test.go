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
	"sourcecrawler/app/model"
	"testing"
)

//Test labeling with log matching + stack trace
func testLabel(t *testing.T, fileName string) {
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

	//Expand the CFG for multiple functions
	if w != nil {
		w.Fset = fset
		w.ASTs = []*ast.File{f}
		cfg.ExpandCFG(w, make([]*cfg.FnWrapper, 0))
	}

	//Test log type
	logTypes := []model.LogType{
		{
			LineNumber: 13,
			FilePath:   "simple.go",
			Regex:      "Logging msg",
		},
		{
			LineNumber: 50,
			FilePath: "test.go",
			Regex: "Number %d here",
		},
	}


	//Gather expressions for paths
	paths := cfg.CreateNewPath()
	leaves := cfg.GetLeafNodes(w)
	for _, leaf := range leaves {
		paths.LabelCFG(leaf, logTypes, w) //Label each block with executionLabel (TraverseCFG can be updated to map each stmt to a label)
		paths.TraverseCFG(leaf, w)
	}

	//Test print labels
	PrintLabels(w)

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

func TestLabelFile(t *testing.T) {
	testLabel(t, "simple.go")
}

func PrintLabels(curr cfg.Wrapper){

	if curr == nil{
		return
	}

	switch wrap := curr.(type) {
	case *cfg.FnWrapper:
		fmt.Println(wrap, " | ", wrap.Label)
	case *cfg.BlockWrapper:
		fmt.Println(wrap, " | ", wrap.Label)
	}

	if len(curr.GetChildren()) == 0 {
		return
	}

	for _, child := range curr.GetChildren() {
		PrintLabels(child)
	}
}
