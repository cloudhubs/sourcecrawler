package test

import (
	"fmt"
	"golang.org/x/tools/go/cfg"
	cfg2 "sourcecrawler/app/cfg"
	//"github.com/rs/zerolog/log"
	"go/ast"
	"go/parser"
	"go/token"

	"testing"
)

type varTestCase struct {
	Name string
}

func TestInterface(t *testing.T) {
	cases := []func() varTestCase{
		func() varTestCase {
			//construct a graph to represent a cfg

			one := cfg2.FnVariableWrapper{
				Name: "one",
			}
			two := cfg2.FnVariableWrapper{
				Name: "two",
			}
			fail := cfg2.FnVariableWrapper{
				Name: "fail",
			}

			one.SetValue("test")
			two.SetValue(&cfg2.FnWrapper{})
			fail.SetValue(2)
			return varTestCase{
				Name: "setValue",
			}

		},
		func() varTestCase {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "testLit.go", nil, 0)
			if err != nil{
				fmt.Println("Error parsing file")
			}

			//var testBlockStmt *ast.BlockStmt
			//var condNode *ast.IfStmt

			//var testCFG1 *cfg.CFG
			//var testCFG2 *cfg.CFG

			var cfgList []*cfg.CFG

			//Get the cfg for testLit.go
			ast.Inspect(file, func(node ast.Node) bool {
				if blockNode, ok := node.(*ast.BlockStmt); ok{
					if len(cfgList) < 2 {
						cfgList = append(cfgList, cfg.New(blockNode, func(expr *ast.CallExpr) bool { return true }))
					}
				}
				return true
			})

			if len(cfgList) >= 2 {
				fmt.Println(cfgList[0].Format(fset))
				fmt.Println(cfgList[1].Format(fset))
			}

			//Make wrappers --- pass in root node
			fnw1 := cfg2.NewFnWrapper(cfgList[0].Blocks[0].Nodes[0])
			//fnw2 := cfg2.NewFnWrapper(cfgList[1].Blocks[0].Nodes[0])

			//Search function literals / assignment for each node in each block
			for _, block := range cfgList[0].Blocks{
				for _, blockNode := range block.Nodes{
					varList :=  cfg2.SearchFuncLits(blockNode)
					if len(varList) > 0{
						for _, v := range varList{
							fmt.Printf("Var: (%s) == (%v)\n", v.GetName(), v.GetValue())
						}
						fnw1.Vars = append(fnw1.Vars, varList...)
					}
					//fmt.Println("Variables list:", varList)
				}
			}
			fmt.Println("FnWrapper for func1", fnw1.Vars)

			//Search variables
			//for _, block := range cfgList[1].Blocks{
			//	for _, blockNode := range block.Nodes{
			//		varList :=  cfg2.SearchFuncLits(blockNode)
			//		if len(varList) > 0{
			//			for _, v := range varList{
			//				fmt.Println("Var name/value", v.GetName(), v.GetValue())
			//			}
			//		}
			//		//fmt.Println("Variables list:", varList)
			//	}
			//}



			return varTestCase{
				Name: "Test Create VarWrapper",
			}
		},
	}

	for _, testCase := range cases {
		test := testCase()
		t.Run(test.Name, func(t *testing.T) {

		})
	}
}