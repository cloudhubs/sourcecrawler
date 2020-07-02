package cfg

import (
	"fmt"
	"os"
	"sourcecrawler/app/db"
	"sourcecrawler/app/helper"
	"testing"

	"github.com/rs/zerolog/log"
)

type addingTestCase struct {
	Name string
	Root db.Node
}

func addThisFunc() {
	addThisFunc()
	fmt.Println("hi")
}
func addThisIndirect2() {
	addThisIndirect1()
	fmt.Println("two")
	log.Info().Msg("a log")
}

func addThisIndirect1() {
	addThisIndirect2()
	fmt.Println("one")
}

func TestAddingFuncs(t *testing.T) {
	cases := []func() addingTestCase{
		func() addingTestCase {
			//construct a graph to represent a cfg

			leaf := &db.StatementNode{}
			fnCall := &db.FunctionNode{FunctionName: "addThisFunc", Child: leaf}
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
			ConnectExternalFunctions(test.Root, []*db.FunctionNode{}, sources, wd)
			PrintCfg(test.Root, "")

		})
	}
}
