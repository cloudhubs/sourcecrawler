/**
This is just a test file for the rewrite functions
*/
package test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	cfg2 "sourcecrawler/app/cfg"
	"testing"

	"golang.org/x/tools/go/cfg"
)

func TestExpressions(t *testing.T) {
	cases := []func() testCase{
		func() testCase {

			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "testLit.go", nil, 0)
			if err != nil {
				fmt.Println("Error parsing file")
			}

			var cfgList []*cfg.CFG

			//Get the cfg for testLit.go
			ast.Inspect(file, func(node ast.Node) bool {
				if blockNode, ok := node.(*ast.BlockStmt); ok {
					if len(cfgList) < 1 {
						cfgList = append(cfgList, cfg.New(blockNode, func(expr *ast.CallExpr) bool { return true }))
					}
				}
				return true
			})

			//Print the blocks
			for _, currCFG := range cfgList {
				fmt.Println(currCFG.Format(fset))
			}

			// if len(cfgList) >= 2 {
			// fmt.Println(cfgList[0].Format(fset))
			// fmt.Println(cfgList[1].Format(fset))
			// }

			//Create a sample tree (separate from the actual cfg)
			root := &cfg2.BlockWrapper{Block: cfgList[0].Blocks[0]} //PathList: cfg2.PathList{Paths: make([]cfg2.Path, 0)},

			tchild := &cfg2.BlockWrapper{Block: cfgList[0].Blocks[1]}
			fchild := &cfg2.BlockWrapper{Block: cfgList[0].Blocks[3]}
			end := &cfg2.BlockWrapper{Block: cfgList[0].Blocks[2]}

			tsucc1 := &cfg2.BlockWrapper{Block: cfgList[0].Blocks[4]}
			tsucc2 := &cfg2.BlockWrapper{Block: cfgList[0].Blocks[6]}

			root.AddChild(tchild)
			root.AddChild(fchild)
			root.SetOuterWrapper(root)

			tchild.AddParent(root)
			tchild.AddChild(tsucc1)
			tchild.AddChild(tsucc2)
			tchild.SetOuterWrapper(root)
			fchild.AddParent(root)
			fchild.AddChild(end)
			fchild.SetOuterWrapper(root)

			end.AddParent(tchild)
			end.AddParent(fchild)
			end.SetOuterWrapper(root)

			tsucc1.AddParent(tchild)
			tsucc2.AddParent(tchild)
			// tsucc1.AddChild(end)
			// tsucc2.AddChild(end)

			//Start at end node and label
			cfg2.LabelCFG(end, nil, root)

			//stmts := make(map[string]cfg2.ExecutionLabel)
			//vars := make(map[ast.Node]cfg2.ExecutionLabel)
			//vars := make(map[ast.Node]string)
			stmts := make(map[string]cfg2.ExecutionLabel)
			vars := []ast.Node{}

			//Start at end node
			//var pathList cfg2.PathList
			cfg2.TraverseCFG(end, stmts, vars, root)

			//Print created execution path
			//filter := make(map[string]string)
			fmt.Println("\n========================")
			cfg2.PathInstance.PrintExecPath()
			cfg2.PathInstance.PrintExpressions()

			return testCase{
				Name: "Test Expression to Z3",
			}
		},
	}

	//Run test cases
	for _, testCase := range cases {

		test := testCase()
		t.Run(test.Name, func(t *testing.T) {
			_, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
