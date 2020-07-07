package test

import (
	"fmt"
	"go/token"
	"os"
	"sourcecrawler/app/cfg"
	"sourcecrawler/app/db"
	"sourcecrawler/app/helper"
	"testing"
)

type addingTestCase struct {
	Name string
	Root db.Node
}

func addThisFunc(a int) {
	addThisFunc(a)
	fmt.Println("hi")
}
func addThisIndirect2(b int) {
	addThisIndirect1()
}

func addThisIndirect1() {
	a := 2
	addThisIndirect2(a)
}

func TestAddingFuncs(t *testing.T) {
	cases := []func() addingTestCase{
		func() addingTestCase {
			//construct a graph to represent a cfg

			leaf := &db.StatementNode{}
			var fnCall = &db.FunctionNode{
				Filename:     "",
				LineNumber:   0,
				FunctionName: "addThisFunc",
				Args: []db.VariableNode{
					{
						Filename:        "",
						LineNumber:      0,
						ScopeId:         "addThisFunc",
						VarName:         "a",
						Value:           "",
						Parent:          nil,
						Child:           nil,
						ValueFromParent: false,
					},},
				Child:  leaf,
				Parent: nil,
				Label:  0,
			}
			leaf.SetParents(fnCall)
			root := &db.StatementNode{Child: fnCall}
			fnCall.SetParents(root)

			return addingTestCase{
				Name: "simple recursion",
				Root: root,
			}

		},
		func() addingTestCase {
			leaf := &db.StatementNode{}
			fnCall2 := &db.FunctionNode{FunctionName: "addThisIndirect1", Child: leaf}
			leaf.SetParents(fnCall2)
			fnCall1 := &db.FunctionNode{FunctionName: "addThisIndirect1", Child: fnCall2}
			fnCall2.SetParents(fnCall1)
			root := &db.StatementNode{Child: fnCall1}
			fnCall1.SetParents(root)

			return addingTestCase{
				Name: "indirect recursion",
				Root: root,
			}
		},
	}

	for _, testCase := range cases {
		test := testCase()
		t.Run(test.Name, func(t *testing.T) {
			wd, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			//run function to add in additional graphs
			sources := helper.GatherGoFiles(wd)
			fnCfg := cfg.NewFnCfgCreator("",wd, token.NewFileSet())
			fnCfg.ConnectExternalFunctions(test.Root, []*db.FunctionNode{}, sources)
			cfg.PrintCfg(test.Root, "")

		})
	}
}
