/**
This is just a test file for the rewrite functions
 */
package test

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"go/ast"
	"go/parser"
	"go/token"
	"golang.org/x/tools/go/cfg"
	"os"
	cfg2 "sourcecrawler/app/cfg"
	"strings"
	"testing"
)

func test(){
	test := "string"
	fmt.Println(test)
}

func TestRegexFromBlock(t *testing.T) {
	cases := []func() addingTestCase{
		func() addingTestCase {

			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "rewrite_test.go", nil, 0)
			if err != nil{
				fmt.Println("Error parsing file")
			}

			var testBlockStmt *ast.BlockStmt
			var testFunc *ast.FuncDecl

			ast.Inspect(file, func(node ast.Node) bool {
				if blockNode, ok := node.(*ast.BlockStmt); ok{
					for _, value := range blockNode.List{
						if strings.Contains(fmt.Sprint(value), "test"){
							testBlockStmt = blockNode
							return true
						}
					}
					//return true //end
				}

				if funcNode, ok := node.(*ast.FuncDecl); ok{
					if funcNode.Name.Name == "testLog"{
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
			testCFG := cfg.New(testBlockStmt, func(expr *ast.CallExpr) bool{
				return true
			})

			formatted := testCFG.Format(fset)
			fmt.Println(formatted)

			//Test log regex
			regexes := cfg2.ExtractLogRegex(testCFG.Blocks[0])
			for _, msg := range regexes{
				fmt.Println("Regex:", msg)
			}

			if regexes[0] != "Testing log message \\d" || regexes[1] != "\\(log msg 2\\)"{
				t.Errorf("Expected: %s and %s  | but got %s and %s\n",
					"Testing log message \\d", "\\(log msg 2\\)", regexes[0], regexes[1])
			}

			return addingTestCase{
				Name: "Extract regex from Block",
				Root: nil,
			}

		},
		func() addingTestCase {

			return addingTestCase{
				Name: "Traverse CFG",
				Root: nil,
			}
		},
	}

	//Run test cases
	for _, testCase := range cases {
		log.Log().Msg("Testing log message %d")
		log.Logger.Info().Msg("(log msg 2)")

		test := testCase()
		t.Run(test.Name, func(t *testing.T) {
			_, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}