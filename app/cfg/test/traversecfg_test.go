/**
This is just a test file for the rewrite functions
*/
package test

import (
	"fmt"
	"os"
	cfg2 "sourcecrawler/app/cfg"
	"testing"

	"github.com/rs/zerolog/log"
)

type rewriteTestCase struct {
	Name string
}

func testPrint(curr cfg2.Wrapper) {

	//if len(curr.GetChildren()) == 0 {
	//	fmt.Println("At bottommost child", curr, curr.GetLabel())
	//}

	if len(curr.GetChildren()) > 0 {
		for _, child := range curr.GetChildren() {
			testPrint(child)
		}
	}

	fmt.Println(curr, curr.GetLabel())
}

func printPath(paths []cfg2.Path) {

	for _, path := range paths {

		//fmt.Println("Path is", path)

		//Print statements
		for _, value := range path.Stmts {
			fmt.Printf("Stmt: %s ", value)
			for _, vars := range path.Variables {
				fmt.Printf("| Vars: (%v)", vars)
			}
			fmt.Println()
		}

	}
}

func TestRewrite2(t *testing.T) {
	cases := []func() rewriteTestCase{
		//func() addingTestCase {
		//
		//	fset := token.NewFileSet()
		//	file, err := parser.ParseFile(fset, "rewrite_test.go", nil, 0)
		//	if err != nil{
		//		fmt.Println("Error parsing file")
		//	}
		//
		//	var testBlockStmt *ast.BlockStmt
		//	var testFunc *ast.FuncDecl
		//
		//	ast.Inspect(file, func(node ast.Node) bool {
		//		if blockNode, ok := node.(*ast.BlockStmt); ok{
		//			for _, value := range blockNode.List{
		//				if strings.Contains(fmt.Sprint(value), "test"){
		//					testBlockStmt = blockNode
		//					return true
		//				}
		//			}
		//			//return true //end
		//		}
		//
		//		if funcNode, ok := node.(*ast.FuncDecl); ok{
		//			if funcNode.Name.Name == "testLog"{
		//				testFunc = funcNode
		//			}
		//		}
		//		return true
		//	})
		//
		//	fmt.Println("Test func is", testFunc)
		//	//for _, val := range testBlockStmt.List{
		//	//	fmt.Println("block stmt",val)
		//	//}
		//
		//	//create test CFG
		//	testCFG := cfg.New(testBlockStmt, func(expr *ast.CallExpr) bool{
		//		return true
		//	})
		//
		//	//formatted := testCFG.Format(fset)
		//	//fmt.Println(formatted)
		//
		//	//Test log regex
		//	regexes := cfg2.ExtractLogRegex(testCFG.Blocks[0])
		//	for _, msg := range regexes{
		//		fmt.Println("Regex:", msg)
		//	}
		//
		//	if len(regexes) >= 2 {
		//		if regexes[0] != "Testing log message \\d" || regexes[1] != "\\(log msg 2\\)" {
		//			t.Errorf("Expected: %s and %s  | but got %s and %s\n",
		//				"Testing log message \\d", "\\(log msg 2\\)", regexes[0], regexes[1])
		//		}
		//	}
		//
		//	return addingTestCase{
		//		Name: "Extract regex from Block",
		//		Root: nil,
		//	}
		//
		//},
		//func() addingTestCase {
		//
		//	fset := token.NewFileSet()
		//	file, err := parser.ParseFile(fset, "rewrite_test.go", nil, 0)
		//	if err != nil{
		//		fmt.Println("Error parsing file")
		//	}
		//
		//	var testBlockStmt *ast.BlockStmt
		//	//var condNode *ast.IfStmt
		//
		//	ast.Inspect(file, func(node ast.Node) bool {
		//		if blockNode, ok := node.(*ast.BlockStmt); ok{
		//			for _, value := range blockNode.List{
		//				if strings.Contains(fmt.Sprint(value), "test"){
		//					testBlockStmt = blockNode
		//					return true
		//				}
		//			}
		//			//return true //end
		//		}
		//
		//
		//		//if ifNode, ok := node.(*ast.IfStmt); ok{
		//		//	condNode = ifNode
		//		//}
		//
		//		return true
		//	})
		//
		//	//create test CFG
		//	testCFG := cfg.New(testBlockStmt, func(expr *ast.CallExpr) bool{
		//		return true
		//	})
		//
		//	//Make test block wrapper
		//	succsList := []cfg2.Wrapper{}
		//	succsList = append(succsList, &cfg2.BlockWrapper{
		//		Block:   testCFG.Blocks[1],
		//		Parents: nil,
		//		Succs:   nil,
		//		Outer:   nil,
		//	})
		//	succsList = append(succsList, &cfg2.BlockWrapper{
		//		Block:   testCFG.Blocks[2],
		//		Parents: nil,
		//		Succs:   nil,
		//		Outer:   nil,
		//	})
		//	blockW := &cfg2.BlockWrapper{
		//		Block:   testCFG.Blocks[0],
		//		Parents: nil,
		//		Succs:   succsList,
		//		Outer:   nil,
		//	}
		//
		//
		//	fmt.Println("Condition is:", blockW.GetCondition())
		//	return addingTestCase{
		//		Name: "Test GetCondition",
		//		Root: nil,
		//	}
		//},
		//func() addingTestCase {
		//
		//	fset := token.NewFileSet()
		//	file, err := parser.ParseFile(fset, "../../unsafe/unsafe.go", nil, 0)
		//	if err != nil{
		//		fmt.Println("Error parsing file")
		//	}
		//
		//	var testBlockStmt *ast.BlockStmt
		//	//var condNode *ast.IfStmt
		//
		//	ast.Inspect(file, func(node ast.Node) bool {
		//		if blockNode, ok := node.(*ast.BlockStmt); ok{
		//			for _, value := range blockNode.List{
		//				if strings.Contains(fmt.Sprint(value), "test"){
		//					testBlockStmt = blockNode
		//					return true
		//				}
		//			}
		//			//return true //end
		//		}
		//
		//
		//		//if ifNode, ok := node.(*ast.IfStmt); ok{
		//		//	condNode = ifNode
		//		//}
		//
		//		return true
		//	})
		//
		//	//create test CFG
		//	testCFG := cfg.New(testBlockStmt, func(expr *ast.CallExpr) bool{
		//		return true
		//	})
		//
		//	fmt.Println(testCFG.Format(fset))
		//
		//
		//	for _, block := range testCFG.Blocks{
		//		varList := cfg2.GetVariables(block.Nodes)
		//
		//		for _, variable := range varList{
		//			switch varType := variable.(type){
		//			case *ast.AssignStmt:
		//				//fmt.Println("is assignment", varType.Lhs, varType.Tok.String(), varType.Rhs)
		//			case *ast.ValueSpec:
		//				//fmt.Println("is value spec", varType)
		//			default:
		//				fmt.Println(varType)
		//			}
		//		}
		//	}
		//
		//
		//	return addingTestCase{
		//		Name: "Get Variables",
		//		Root: nil,
		//	}
		//},
		//func() addingTestCase {
		//
		//	fset := token.NewFileSet()
		//	file, err := parser.ParseFile(fset, "rewrite_test.go", nil, 0)
		//	if err != nil{
		//		fmt.Println("Error parsing file")
		//	}
		//
		//	var testBlockStmt *ast.BlockStmt
		//	//var condNode *ast.IfStmt
		//	var funcNode *ast.FuncDecl
		//
		//	ast.Inspect(file, func(node ast.Node) bool {
		//		if blockNode, ok := node.(*ast.BlockStmt); ok{
		//			for _, value := range blockNode.List{
		//				if strings.Contains(fmt.Sprint(value), "test"){
		//					testBlockStmt = blockNode
		//					return true
		//				}
		//			}
		//			//return true //end
		//		}
		//
		//
		//		if declNode, ok := node.(*ast.FuncDecl); ok{
		//			funcNode = declNode
		//		}
		//
		//		return true
		//	})
		//
		//	//create test CFG
		//	testCFG := cfg.New(testBlockStmt, func(expr *ast.CallExpr) bool{
		//		return true
		//	})
		//
		//	fmt.Println(testCFG.Format(fset))
		//
		//	condStmts := []string{}
		//	varNodes := []ast.Node{}
		//
		//	rootWrapper := &cfg2.BlockWrapper{ //block 0
		//		Block:   testCFG.Blocks[0],
		//		Parents: nil,
		//		Succs:   nil,
		//		Outer:   nil,
		//	}
		//
		//	succ1 :=  &cfg2.BlockWrapper{ //block 1
		//		Block:   testCFG.Blocks[1],
		//		Parents: nil,
		//		Succs:   nil,
		//		Outer:   nil,
		//	}
		//
		//	exceptionWrapper := &cfg2.BlockWrapper{ //block 3
		//		Block:   testCFG.Blocks[3],
		//		Parents: nil,
		//		Succs:   nil,
		//		Outer:   nil,
		//	}
		//
		//	//Set parents
		//	exceptionWrapper.AddParent(rootWrapper)
		//	succ1.AddParent(rootWrapper)
		//
		//	//Set root children
		//	rootWrapper.AddChild(succ1)
		//	rootWrapper.AddChild(exceptionWrapper)
		//
		//	//FnWrapper
		//	funcWrapper := &cfg2.FnWrapper{
		//		Fn:         funcNode,
		//		FirstBlock: rootWrapper,
		//		Parents:    nil,
		//		Outer:      nil,
		//	}
		//
		//	//Test on simple case
		//	cfg2.TraverseCFG(exceptionWrapper, condStmts, varNodes, rootWrapper)
		//	fmt.Println("Execution path after", cfg2.GetExecPath())
		//
		//	//for _, value := range cfg2.GetExecPath(){
		//	//	fmt.Println(value.Variables)
		//	//}
		//
		//	//Test on function wrapper
		//	cfg2.TraverseCFG(funcWrapper, condStmts, varNodes, rootWrapper)
		//
		//	return addingTestCase{
		//		Name: "TraverseCFG",
		//		Root: nil,
		//	}
		//},
		func() rewriteTestCase {

			// 	fset := token.NewFileSet()
			// 	file, err := parser.ParseFile(fset, "testLit.go", nil, 0)
			// 	if err != nil{
			// 		fmt.Println("Error parsing file")
			// 	}

			// 	var cfgList []*cfg.CFG

			// 	//Get the cfg for testLit.go
			// 	ast.Inspect(file, func(node ast.Node) bool {
			// 		if blockNode, ok := node.(*ast.BlockStmt); ok{
			// 			if len(cfgList) < 2 {
			// 				cfgList = append(cfgList, cfg.New(blockNode, func(expr *ast.CallExpr) bool { return true }))
			// 			}
			// 		}
			// 		return true
			// 	})

			// 	if len(cfgList) >= 2 {
			// 		//fmt.Println(cfgList[0].Format(fset))
			// 		//fmt.Println(cfgList[1].Format(fset))
			// 	}

			// 	//Create a sample tree
			// 	root := &cfg2.BlockWrapper{Block: cfgList[0].Blocks[0]}
			// 	tchild := &cfg2.BlockWrapper{Block: cfgList[0].Blocks[1]}
			// 	fchild := &cfg2.BlockWrapper{Block: cfgList[0].Blocks[3]}
			// 	end := &cfg2.BlockWrapper{Block: cfgList[0].Blocks[2]}

			// 	root.AddChild(tchild)
			// 	root.AddChild(fchild)
			// 	root.SetOuterWrapper(root)

			// 	tchild.AddParent(root)
			// 	tchild.AddChild(end)
			// 	tchild.SetOuterWrapper(root)
			// 	fchild.AddParent(root)
			// 	fchild.AddChild(end)
			// 	fchild.SetOuterWrapper(root)

			// 	end.AddParent(tchild)
			// 	end.AddParent(fchild)
			// 	end.SetOuterWrapper(root)

			// 	//testPrint(root)
			// 	cfg2.LabelCFG(end, nil, root)

			// 	//fmt.Printf("\nAfter label function: ============\n")
			// 	//testPrint(root)

			// 	return testCase{
			// 		Name: "TestRewriteLabel",
			// 	}
			// },
			// func() testCase {

			// 	fset := token.NewFileSet()
			// 	file, err := parser.ParseFile(fset, "testLit.go", nil, 0)
			// 	if err != nil{
			// 		fmt.Println("Error parsing file")
			// 	}

			// 	var cfgList []*cfg.CFG

			// 	//Get the cfg for testLit.go
			// 	ast.Inspect(file, func(node ast.Node) bool {
			// 		if blockNode, ok := node.(*ast.BlockStmt); ok{
			// 			if len(cfgList) < 2 {
			// 				cfgList = append(cfgList, cfg.New(blockNode, func(expr *ast.CallExpr) bool { return true }))
			// 			}
			// 		}
			// 		return true
			// 	})

			// 	//if len(cfgList) >= 2 {
			// 	//	fmt.Println(cfgList[0].Format(fset))
			// 	//	fmt.Println(cfgList[1].Format(fset))
			// 	//}

			// 	//Create a sample tree
			// 	root := &cfg2.BlockWrapper{Block: cfgList[0].Blocks[0],
			// 		//PathList: cfg2.PathList{Paths: make([]cfg2.Path, 0)},
			// 	}
			// 	tchild := &cfg2.BlockWrapper{Block: cfgList[0].Blocks[1]}
			// 	fchild := &cfg2.BlockWrapper{Block: cfgList[0].Blocks[3]}
			// 	end := &cfg2.BlockWrapper{Block: cfgList[0].Blocks[2]}

			// 	root.AddChild(tchild)
			// 	root.AddChild(fchild)
			// 	root.SetOuterWrapper(root)

			// 	tchild.AddParent(root)
			// 	tchild.AddChild(end)
			// 	tchild.SetOuterWrapper(root)
			// 	fchild.AddParent(root)
			// 	fchild.AddChild(end)
			// 	fchild.SetOuterWrapper(root)

			// 	end.AddParent(tchild)
			// 	end.AddParent(fchild)
			// 	end.SetOuterWrapper(root)

			// 	//Start at end node and label
			// 	cfg2.LabelCFG(end, nil, root)

			// 	//stmts := make(map[string]cfg2.ExecutionLabel)
			// 	//vars := make(map[ast.Node]cfg2.ExecutionLabel)
			// 	stmts := make(map[string]cfg2.ExecutionLabel)
			// 	//vars := make(map[ast.Node]string)
			// 	vars := []ast.Node{}

			// 	//Start at end node
			// 	//var pathList cfg2.PathList
			// 	cfg2.TraverseCFG(end, stmts, vars, root)

			// 	//Print created execution path
			// 	//filter := make(map[string]string)
			// 	fmt.Println("\n========================")
			// 	cfg2.PathInstance.PrintExecPath()
			// 	fmt.Println("PathList Expressions:", cfg2.PathInstance.Expressions)

			return rewriteTestCase{
				Name: "Test Execution Paths",
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
		sum3 := two()

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
