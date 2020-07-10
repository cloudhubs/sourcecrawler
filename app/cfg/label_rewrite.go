package cfg

import (
	"fmt"
	"go/ast"
	"sourcecrawler/app/handler"
	"sourcecrawler/app/logsource"
	"sourcecrawler/app/model"
)

//---------- Labeling feature for Must/May-haves (rewrite) --------------
func LabelCFG(curr Wrapper, logs []model.LogType, root Wrapper) {

	//Label root as must
	if curr == root || len(curr.GetParents()) == 0 {
		fmt.Println("At topmost wrapper")
		curr.SetLabel(Must)
		return
	}

	wrapper := curr //holds current wrapper

	//Iterate up through parents up to root
	for len(wrapper.GetParents()) > 0 {
		if wrapper.GetLabel() == NoLabel {
			switch wrap := wrapper.(type) {
			case *FnWrapper:
				//If it's a function in the stack trace, then it must run
				//TODO: Finish testing function (Need to give it stack trace info from somewhere)
				testStkInfo := []handler.StackTraceStruct{}
				if CheckFnStatus(wrap, testStkInfo) {
					wrap.SetLabel(Must)
				} else {
					wrap.SetLabel(May)
				}
			case *BlockWrapper: //BlockWrapper can represent a condition, but could be a statement, etc
				//Check if it's a condition, if not set as must
				if wrap.GetCondition() == "" {
					wrap.SetLabel(Must)

				} else { //Label May upwards until a single parent (if it isn't a conditional) TODO: finish testing
					wrap.SetLabel(May)
					trueParent := wrap.Parents[0]
					GetTopAndLabel(trueParent, wrap)
					//falseParent := wrap.Parents[1]

					//For two parents, get the top block and label downwards as May
				}

				//If it is a log then need to label entire branch
				//Check each wrapper to see if it is from log
				//TODO: refactor log labeling functionality
				if CheckLogStatus(wrap.Block.Nodes) {
					//Add log label functionality
					fmt.Println("Add log labeling stuff")
				}
			}
		}else{
			fmt.Println("Wrapper is already labeled", wrapper)
		}

		//Set next wrapper (parents)
		if len(curr.GetParents()) == 1 {
			wrapper = curr.GetParents()[0]
		} else if len(curr.GetParents()) == 2 {
			wrapper = curr.GetParents()[0]
			//fwrapper := curr.GetParents()[1]
		}
	}
}


//Helper function to get topmost node where conditionals connect
func GetTopAndLabel(wrapper Wrapper, start Wrapper) Wrapper{

	//Go up until a node with 2 children are found (top condition)
	curr := wrapper
	for len(curr.GetParents()) > 0 && len(curr.GetChildren()) != 2{
		curr =  wrapper.GetParents()[0]
		if len(curr.GetChildren()) == 2{
			break
		}
	}

	//Go down through children to label nodes
	for _, child := range curr.GetChildren(){
		ch := child

		for len(ch.GetChildren()) > 0{
			if len(ch.GetChildren()) == 1{
				ch = ch.GetChildren()[0]
			}else if len(ch.GetChildren()) == 2{

			}
			//If the original node at the bottom is found, then
			if ch == start{

			}
		}
	}

	return curr
}

//Helper for checking if FnWrapper is a may/must (checks function with stack to see if it is relevant)
func CheckFnStatus(wrapper *FnWrapper, stackInfo []handler.StackTraceStruct) bool{
	var isMust = false

	switch funcNode := wrapper.Fn.(type){
	case *ast.FuncDecl:
		//Check if function is in the stack trace
		for _, stkInfo := range stackInfo{
			for _, funcName := range stkInfo.FuncName {
				if funcNode.Name.Name == funcName{
					isMust = true
					break
				}
			}
		}
	case *ast.FuncLit:

	}

	return isMust
}

//Helper function to check if a BlockWrapper contains a log
func CheckLogStatus(nodes []ast.Node) bool {
	for _, node := range nodes{
		if selNode, ok := node.(*ast.SelectorExpr); ok{
			if logsource.IsFromLog(selNode){ //if any node in the block contains a log statement, exit early
				return true
			}
		}
	}

	return false
}