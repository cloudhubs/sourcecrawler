package test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sourcecrawler/app/cfg"
	"testing"
)

// Run test with -v flag to see Log prints

type pointerTest struct {
	Name string
	Src  string
	Vars []string
}

func TestPointerArgs(t *testing.T) {
	cases := []func() pointerTest{
		func() pointerTest {
			src := `
			package main
			func main() {
				i := 0
				foo(i)
			}
			func foo(a *int) {
				bar(a)
			}
			func bar(b *int) {
				b++
			}
			`
			return pointerTest{
				Name: "Nested Pointer",
				Src:  src,
				Vars: []string{"i"},
			}
		},
	}

	for _, testCase := range cases {
		test := testCase()
		t.Run(test.Name, func(t *testing.T) {
			fset := token.NewFileSet()
			f, _ := parser.ParseFile(fset, "", test.Src, 0)

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

			condStmts := make([]string, 0)
			vars := make([]ast.Node, 0)

			leaves := cfg.GetLeafNodes(w)
			if len(leaves) > 0 {
				cfg.TraverseCFG(leaves[0], condStmts, vars, w)
			} else {
				t.Error("Not enough leaves")
			}

			// cfg.DebugPrint(w, "", make(map[cfg.Wrapper]struct{}))

			// cfg.TraverseCFG(w, condStmts, vars, w)

			// traverse(w)

			t.Log(vars)
		})
	}
}

// func traverse(w cfg.Wrapper) {
// 	fmt.Println(w.GetChildren(), w.GetParents(), reflect.TypeOf(w))
// 	for _, succ := range w.GetChildren() {
// 		traverse(succ)
// 	}
// }
