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
	"strings"
	"testing"

	"github.com/rs/zerolog/log"
	"golang.org/x/tools/go/cfg"
)

type addingTestCase struct {
	Name string
	Root interface{}
}

func test(arg1 int) {
	test := "string"
	fmt.Println(test)
}

func getNum() int {
	return 5
}

func TestRegexFromBlock(t *testing.T) {
	cases := []func() addingTestCase{
		func() addingTestCase {

			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "rewrite_test.go", nil, 0)
			if err != nil {
				fmt.Println("Error parsing file")
			}

			var testBlockStmt *ast.BlockStmt
			var testFunc *ast.FuncDecl

			ast.Inspect(file, func(node ast.Node) bool {
				if blockNode, ok := node.(*ast.BlockStmt); ok {
					for _, value := range blockNode.List {
						if strings.Contains(fmt.Sprint(value), "test") {
							testBlockStmt = blockNode
							return true
						}
					}
					//return true //end
				}

				if funcNode, ok := node.(*ast.FuncDecl); ok {
					if funcNode.Name.Name == "testLog" {
						testFunc = funcNode
					}
				}
				return true
			})

			fmt.Println("Test func is", testFunc)
			//for _, val := range testBlockStmt.List{
			//	fmt.Println("block stmt",val)
			//}

			//create test CFG
			testCFG := cfg.New(testBlockStmt, func(expr *ast.CallExpr) bool {
				return true
			})

			//formatted := testCFG.Format(fset)
			//fmt.Println(formatted)

			//Test log regex
			regexes := cfg2.ExtractLogRegex(testCFG.Blocks[0])
			for _, msg := range regexes {
				fmt.Println("Regex:", msg)
			}

			if len(regexes) >= 2 {
				if regexes[0] != "Testing log message \\d" || regexes[1] != "\\(log msg 2\\)" {
					t.Errorf("Expected: %s and %s  | but got %s and %s\n",
						"Testing log message \\d", "\\(log msg 2\\)", regexes[0], regexes[1])
				}
			}

			return addingTestCase{
				Name: "Extract regex from Block",
				Root: nil,
			}

		},
		func() addingTestCase {

			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "rewrite_test.go", nil, 0)
			if err != nil {
				fmt.Println("Error parsing file")
			}

			var testBlockStmt *ast.BlockStmt
			//var condNode *ast.IfStmt

			ast.Inspect(file, func(node ast.Node) bool {
				if blockNode, ok := node.(*ast.BlockStmt); ok {
					for _, value := range blockNode.List {
						if strings.Contains(fmt.Sprint(value), "test") {
							testBlockStmt = blockNode
							return true
						}
					}
					//return true //end
				}

				//if ifNode, ok := node.(*ast.IfStmt); ok{
				//	condNode = ifNode
				//}

				return true
			})

			//create test CFG
			testCFG := cfg.New(testBlockStmt, func(expr *ast.CallExpr) bool {
				return true
			})

			//Make test block wrapper
			succsList := []cfg2.Wrapper{}
			succsList = append(succsList, &cfg2.BlockWrapper{
				Block:   testCFG.Blocks[1],
				Parents: nil,
				Succs:   nil,
				Outer:   nil,
			})
			succsList = append(succsList, &cfg2.BlockWrapper{
				Block:   testCFG.Blocks[2],
				Parents: nil,
				Succs:   nil,
				Outer:   nil,
			})
			blockW := &cfg2.BlockWrapper{
				Block:   testCFG.Blocks[0],
				Parents: nil,
				Succs:   succsList,
				Outer:   nil,
			}

			fmt.Println("Condition is:", blockW.GetCondition())
			return addingTestCase{
				Name: "Test GetCondition",
				Root: nil,
			}
		},
		func() addingTestCase {

			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "rewrite_test.go", nil, 0)
			if err != nil {
				fmt.Println("Error parsing file")
			}

			var testBlockStmt *ast.BlockStmt
			//var condNode *ast.IfStmt

			ast.Inspect(file, func(node ast.Node) bool {
				if blockNode, ok := node.(*ast.BlockStmt); ok {
					for _, value := range blockNode.List {
						if strings.Contains(fmt.Sprint(value), "test") {
							testBlockStmt = blockNode
							return true
						}
					}
					//return true //end
				}

				//if ifNode, ok := node.(*ast.IfStmt); ok{
				//	condNode = ifNode
				//}

				return true
			})

			//create test CFG
			testCFG := cfg.New(testBlockStmt, func(expr *ast.CallExpr) bool {
				return true
			})

			fmt.Println(testCFG.Format(fset))

			//for _, block := range testCFG.Blocks {
			//	varList := cfg2.GetVariables(block.Nodes)
			//
			//	for _, variable := range varList {
			//		switch varType := variable.(type) {
			//		case *ast.AssignStmt:
			//			//fmt.Println("is assignment", varType.Lhs, varType.Tok.String(), varType.Rhs)
			//		case *ast.ValueSpec:
			//			//fmt.Println("is value spec", varType)
			//		default:
			//			fmt.Println(varType)
			//		}
			//	}
			//}

			return addingTestCase{
				Name: "Get Variables",
				Root: nil,
			}
		},
		func() addingTestCase {

			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "rewrite_test.go", nil, 0)
			if err != nil {
				fmt.Println("Error parsing file")
			}

			var testBlockStmt *ast.BlockStmt
			//var condNode *ast.IfStmt
			var funcNode *ast.FuncDecl

			ast.Inspect(file, func(node ast.Node) bool {
				if blockNode, ok := node.(*ast.BlockStmt); ok {
					for _, value := range blockNode.List {
						if strings.Contains(fmt.Sprint(value), "test") {
							testBlockStmt = blockNode
							return true
						}
					}
					//return true //end
				}

				if declNode, ok := node.(*ast.FuncDecl); ok {
					funcNode = declNode
				}

				return true
			})

			//create test CFG
			testCFG := cfg.New(testBlockStmt, func(expr *ast.CallExpr) bool {
				return true
			})

			fmt.Println(testCFG.Format(fset))

			//condStmts := []string{}
			condStmts := make(map[string]cfg2.ExecutionLabel)
			//varNodes := make(map[ast.Node]string)
			varNodes := []ast.Node{}

			rootWrapper := &cfg2.BlockWrapper{ //block 0
				Block:   testCFG.Blocks[0],
				Parents: nil,
				Succs:   nil,
				Outer:   nil,
			}

			succ1 := &cfg2.BlockWrapper{ //block 1
				Block:   testCFG.Blocks[1],
				Parents: nil,
				Succs:   nil,
				Outer:   nil,
			}

			exceptionWrapper := &cfg2.BlockWrapper{ //block 3
				Block:   testCFG.Blocks[3],
				Parents: nil,
				Succs:   nil,
				Outer:   nil,
			}

			//Set parents
			exceptionWrapper.AddParent(rootWrapper)
			succ1.AddParent(rootWrapper)

			//Set root children
			rootWrapper.AddChild(succ1)
			rootWrapper.AddChild(exceptionWrapper)

			//FnWrapper
			funcWrapper := &cfg2.FnWrapper{
				Fn:         funcNode,
				FirstBlock: rootWrapper,
				Parents:    nil,
				Outer:      nil,
			}

			//Test on simple case
			cfg2.TraverseCFG(exceptionWrapper, condStmts, varNodes, rootWrapper, make(map[string]ast.Node))
			fmt.Println("Execution path after", cfg2.PathInstance.GetExecPath())

			//for _, value := range cfg2.GetExecPath(){
			//	fmt.Println(value.Variables)
			//}

			//Test on function wrapper
			cfg2.TraverseCFG(funcWrapper, condStmts, varNodes, rootWrapper, make(map[string]ast.Node))

			return addingTestCase{
				Name: "TraverseCFG",
				Root: nil,
			}
		},
	}

	//Run test cases
	for _, testCase := range cases {
		log.Log().Msg("Testing log message %d")
		log.Logger.Info().Msg("(log msg 2)")

		//ValueSpecs
		var decl1 string
		var decl2 int = 5

		//Assign stmts
		assignStr := "test if"
		num := 10
		sum := num
		sum2 := num + num
		sum3 := getNum()

		if 5 >= 10 {
			fmt.Println("test if")
			fmt.Println(assignStr)
			fmt.Println(sum)
			fmt.Println(sum2)
			fmt.Println(sum3)
			fmt.Println(decl1)
			fmt.Println(decl2)
			panic("bad")
		} else {
			fmt.Println("else")
		}

		test := testCase()
		t.Run(test.Name, func(t *testing.T) {
			_, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func testRewrite(a string, b int) {
	//testVar := ""
	////
	//fmt.Println(testVar)
}
