package test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"sourcecrawler/app/cfg"
	"testing"

	"github.com/mitchellh/go-z3"
)

// Run test with -v flag to see Log prints

type z3Test struct {
	Name       string
	Conditions []string
}

func TestZ3Conditions(t *testing.T) {
	cases := []func() z3Test{
		func() z3Test {
			return z3Test{
				Name: "Example Case",
				Conditions: []string{
					"x + y + z > 4",
					"x + y < 2",
					"z > 0",
					"x != y != z",
					"x != 0",
					"y != 0",
					"z != 0",
					"x + y = -3",
				},
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
			exprs := make([]ast.Expr, len(test.Conditions))
			for i, cond := range test.Conditions {
				if expr, err := parser.ParseExpr(cond); err == nil && expr != nil {
					exprs[i] = expr
				}
			}

			s := ctx.NewSolver()
			defer s.Close()

			for _, expr := range exprs {
				if expr == nil {
					t.Error("expr was nil")
					continue
				}
				condition := cfg.ConvertExprToZ3(ctx, expr)
				if condition != nil {
					s.Assert(condition)
				} else {
					t.Error("condition was nil")
				}
			}

			if v := s.Check(); v != z3.True {
				t.Error("Unsolvable")
				return
			}

			m := s.Model()
			defer m.Close()
			assignments := m.Assignments()
			for name, val := range assignments {
				t.Logf("%s = %s\n", name, val)
				fmt.Printf("%s = %s\n", name, val)
			}

		})
	}
}
