package test

import (
	"bufio"
	"fmt"
	_ "fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sourcecrawler/app/cfg"
	"sourcecrawler/app/helper"
	"testing"

	"github.com/mitchellh/go-z3"
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

	logs := helper.ParseProject("../../../../sourcecrawler")
	//Sample stack trace
	stkTrcFile, err := os.Open("../../../stackTrace.log")
	if err != nil {
		fmt.Println("Bad stack trace file")
	}
	scanner := bufio.NewScanner(stkTrcFile)
	messageString := ""
	for scanner.Scan() {
		messageString += scanner.Text() + "\n"
	}
	// fmt.Println("Message", messageString)
	stkInfo := helper.ParsePanic("../../../../sourcecrawler", messageString)

	// condStmts := make(map[ast.Node]cfg.ExecutionLabel)
	// vars := make([]ast.Node, 0)
	// var exprs []ast.Node

	paths := cfg.CreateNewPath()
	leaves := cfg.GetLeafNodes(w)
	for _, leaf := range leaves {
		paths.LabelCFG(leaf, logs, w, stkInfo) //output isnt printed, but labeling still occurs
		paths.TraverseCFG(leaf, w)
	}

	config := z3.NewConfig()
	ctx := z3.NewContext(config)
	config.Close()
	defer ctx.Close()

	cnt := 1
	for _, path := range paths.Paths {
		fmt.Println("---------- PATH", cnt, " -------------")
		cnt++
		// for _, expr := range path.Expressions {
		// 	printer.Fprint(os.Stdout, fset, expr)
		// 	fmt.Println()
		// }

		s := ctx.NewSolver()
		defer s.Close()

		for _, expr := range path.Expressions {
			if expr == nil {
				t.Error("expr was nil")
				continue
			}
			if _, ok := expr.(*ast.ReturnStmt); ok {
				t.Error("found a return stmt")
				continue
			}
			if _, ok := expr.(*ast.Ident); ok {
				t.Error("lone id ..? ")
				continue
			}
			condition := cfg.ConvertExprToZ3(ctx, expr, fset)
			if condition != nil {
				fmt.Println(condition.String())
				s.Assert(condition)
			} else {
				t.Errorf("condition %v (%T) was nil", expr, expr)
			}
		}

		if v := s.Check(); v != z3.True {
			t.Error("Unsolvable")
			return
		}
		t.Log("it passed!")
		m := s.Model()
		assignments := m.Assignments()

		cfg.FilterToUserInput(leaves[0], path.Expressions, assignments)
		for name, val := range assignments {
			t.Logf("%s = %s\n", name, val)
			fmt.Printf("%s = %s\n", name, val)
		}
		defer m.Close()
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

func TestZ3AndSSA(t *testing.T) {
	testUtil(t, "example_z3.go")
}
