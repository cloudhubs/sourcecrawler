package test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sourcecrawler/app/cfg"
	cfg2 "sourcecrawler/app/cfg"
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
				three(b)
			}
			func three(c *int) {
				c++	
			}
			`
			return pointerTest{
				Name: "Nested Pointer",
				Src:  src,
				Vars: []string{"main.i"},
			}
		},
		func() pointerTest {
			src := `
			package main
			func main() {
				a := 3
				foo(a)
			}
			func foo(x int) {
				// do something with x
				x++
			}
			`
			return pointerTest{
				Name: "Pass by value",
				Src:  src,
				Vars: []string{"foo.x", "main.a"},
			}
		},
		func() pointerTest {
			src := `
			package main
			type Foo struct {
				Prop int
			}
			func main() {
				a := Foo{3}
				a.Prop = 10 // included
				a.bar()     // call not included
			}
			func (f *Foo) bar() {
				fmt.Println(f.Prop)
			}
			`
			return pointerTest{
				Name: "Struct Attribute",
				Src:  src,
				Vars: []string{"main.a.Prop", "main.a"},
			}
		},
		func() pointerTest {
			src := `
			package main
			func main() {
				a := func(){fmt.Println()}
				foo(a)
			}
			func foo(b func()){
				b()
			}
			`
			return pointerTest{
				Name: "Local Function Arg",
				Src:  src,
				Vars: []string{"main.a"},
			}
		},
		func() pointerTest {
			src := `
			package main
			func main() {
				a := func(){fmt.Println()}
				foo(func(){fmt.Println()})
			}
			func foo(b func()){
				b()
			}
			`
			return pointerTest{
				Name: "Function Literal Arg",
				Src:  src,
				Vars: []string{},
			}
		},
		func() pointerTest {
			src := `
			package main
			func main() {
				b := func(){fmt.Println()}
				foo(bar)
			}
			func foo(b func()){
				b()
			}
			func bar(){
				fmt.Println()
			}
			`
			return pointerTest{
				Name: "Package Function Arg",
				Src:  src,
				Vars: []string{"main.b"},
			}
		},
		func() pointerTest {
			src := `
			package main
			func main() {
				a := func(){fmt.Println()}
				foo(a)
			}
			func foo(b func()){
				bar(b)
			}
			func bar(c func()){
				c()
			}
			`
			return pointerTest{
				Name: "Nested Function Arg",
				Src:  src,
				Vars: []string{"main.a"},
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
				cfg.ExpandCFGRecur(w, make([]*cfg.FnWrapper, 0))
			}

			// condStmts := make(map[ast.Node]cfg.ExecutionLabel)
			// stmts := make([]ast.Node, 0)

			path := cfg2.CreateNewPath()
			leaves := cfg.GetLeafNodes(w)
			for _, leaf := range leaves {
				path.TraverseCFG(leaf, w)
			}
			// if len(leaves) > 0 {
			// 	cfg.TraverseCFG(leaves[0], condStmts, vars, w, make(map[string]ast.Node))
			// } else {
			// 	t.Error("Not enough leaves")
			// }

			// cfg.DebugPrint(w, "", make(map[cfg.Wrapper]struct{}))

			// cfg.TraverseCFG(w, condStmts, vars, w)

			// traverse(w)
			t.Log(path)
			// for _, p := range path.GetExecPath() {
			// 	if len(p.Variables) != len(test.Vars) {
			// 		t.Error("expected # of vars", len(test.Vars), "found", len(p.Variables))
			// 	} else {
			// 		for i, x := range p.Variables {
			// 			t.Log(x, reflect.TypeOf(x))
			// 			// if x, ok := x.(*ast.ExprStmt); ok {
			// 			// 	t.Log("   ", x.X, reflect.TypeOf(x.X))
			// 			// }
			// 			name := ""
			// 			if v, ok := x.(*ast.SelectorExpr); ok {
			// 				name = fmt.Sprintf("%v.%v", v.X, v.Sel)
			// 			} else {
			// 				name = fmt.Sprint(x)
			// 			}
			// 			if fmt.Sprint(x) != test.Vars[i] {
			// 				t.Error("expected var", test.Vars[i], "found", name)
			// 			}
			// 		}
			// 	}
			// }
		})
	}
}

// func traverse(w cfg.Wrapper) {
// 	fmt.Println(w.GetChildren(), w.GetParents(), reflect.TypeOf(w))
// 	for _, succ := range w.GetChildren() {
// 		traverse(succ)
// 	}
// }
