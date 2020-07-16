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

	"github.com/mitchellh/go-z3"
)

// Run test with -v flag to see Log prints

type z3Test struct {
	Name string
	Src  string
}

func TestZ3Conditions(t *testing.T) {
	cases := []func() z3Test{
		func() z3Test {
			return z3Test{
				Name: "Example Case",
				Src: `
				package main
				func main() {
					x, y, z := -1, -2, 8
					x + y + z > 4
					x + y < 2
					z > 0
					x != y
					x != z
					y != z
					x != 0
					y != 0
					z != 0
					x + y == -3
				}
				`,
			}
		},
		func() z3Test {
			return z3Test{
				Name: "Contradictions",
				Src: `
				package main
				func main() {
					x, y := 0, 0
					x < 2
					x == 2
					x > 2
				}
				`,
			}
		},
	}

	for _, testCase := range cases {
		config := z3.NewConfig()
		ctx := z3.NewContext(config)
		config.Close()
		defer ctx.Close()

		test := testCase()
		t.Run(test.Name, func(t *testing.T) {
			exprs := make([]ast.Expr, 0)
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "", test.Src, parser.ParseComments)
			if err != nil {
				t.Error("could not parse file")
				return
			}

			ast.Inspect(f, func(n ast.Node) bool {
				if fn, ok := n.(*ast.FuncDecl); ok && fn != nil {
					if body := fn.Body; body != nil {
						for _, stmt := range body.List {
							if stmt == nil {
								continue
							}
							if expr, ok := stmt.(*ast.ExprStmt); ok {
								if _, ok := expr.X.(*ast.CallExpr); ok {
									continue
								}
								printer.Fprint(os.Stdout, fset, expr.X)
								fmt.Println()
								exprs = append(exprs, expr.X)
							}
						}
					}
				}

				return true
			})

			s := ctx.NewSolver()
			defer s.Close()

			for _, expr := range exprs {
				if expr == nil {
					t.Error("expr was nil")
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
			for name, val := range assignments {
				t.Logf("%s = %s\n", name, val)
				fmt.Printf("%s = %s\n", name, val)
			}
			defer m.Close()

		})
	}
}
