package cfg

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"sourcecrawler/app/helper"
	"sourcecrawler/app/model"
	"strings"
)

//---------- Labeling feature for Must/May-haves (rewrite) --------------
//Assumptions: CFG tree already created
func (paths *PathList) LabelCFG(curr Wrapper, logs []model.LogType, root Wrapper, stackInfo helper.StackTraceStruct) {

	//Nil check
	if curr == nil {
		return
	}
	if curr == root{ //Root should be a must (make sure immediate block before entering exception block is labeled)
		curr.SetLabel(Must)
		if len(curr.GetParents()) == 1{
			curr.GetParents()[0].SetLabel(Must)
			//fmt.Println("Parent labeled as must", curr.GetParents()[0])
		}else if len(curr.GetParents()) == 2{ //Not sure if this ever occurs
			truePar := curr.GetParents()[0]
			falsePar := curr.GetParents()[1]
			fmt.Println("True par", truePar)
			fmt.Println("false par", falsePar)
		}
	}

	wrapper := curr //holds current wrapper
	var prv Wrapper

	//Iterate up through parents up to root
	for len(wrapper.GetParents()) > 0 && prv != wrapper {
		prv = wrapper
		if wrapper.GetLabel() == NoLabel {
			switch wrap := wrapper.(type) {
			case *FnWrapper:

				//If it's a function in the stack trace, then it must run
				//fnwrapper connected to blockWrapper succs
				if CheckFnStatus(wrap, stackInfo) {
					wrap.SetLabel(Must)
				} else {
					wrap.SetLabel(May)
				}
				// fmt.Println("Current wrapper in label", curr)

			case *BlockWrapper: //BlockWrapper can represent a condition, but could be a statement, etc
				//Check if it's a condition, if not set as must

				//fmt.Println("Current wrapper in label", wrap)
				//if strings.Contains(wrap.Block.String(), "block 5") || strings.Contains(wrap.Block.String(), "block 6"){
				//	fmt.Println("BLOCK 5 OR BLOCK 6")
				//}


				//Check for possible log msg and log matchings
				if strings.Contains(wrap.Block.String(), "entry") { //Entry is may
					wrap.SetLabel(May)
				}else if strings.Contains(wrap.Block.String(), "if.then") ||//If it is part of an if-then or if-else, it is labeled as may (parent can be overriden if log found)
					strings.Contains(wrap.Block.String(), "if.else") {
					fmt.Println("Block being processed in if/else", wrap.Block.String())
					LabelIfElseBlock(curr, logs, root)
				}else if wrap.GetCondition() != nil{ //Only true if a block has 2 successors
					//wrap.SetLabel(Must)
					wrap.SetLabel(May)
				}else{
					wrap.SetLabel(May) //label as May if no logs detected
					//wrap.SetLabel(MustNot)
				}

				/*else if strings.Contains(wrap.Block.String(), "if.done"){ //If-done should always be a must
					wrap.SetLabel(Must)
				}*/

				if CheckLogStatus(wrap.Block.Nodes, logs) { //If there's a matching log statement, then it has to be a must
					wrap.SetLabel(Must)
					//wrap.SetLabel(May)
				}


				//If two parents, go up to top and label down
				if len(wrap.GetParents()) == 2 {
					//If the wrapper also has two children, finish labeling other child wrapper first
					if len(wrap.GetChildren()) == 2{
						ProcessChildrenWraps(wrapper, logs, root)
					}
					//fmt.Println(wrap.Block.String(), " has two parents")
					wrapper = GetTopAndLabel(wrap, logs, wrap, stackInfo, root)
				}
			}
		} else {
			//fmt.Println("Wrapper is already labeled", wrapper)
			// return
		}

		//Set next wrapper (parents) - two parent case should already be handled already by GetTopAndLabel
		if len(wrapper.GetParents()) == 1 {
			wrapper = wrapper.GetParents()[0]
		}
	}

	//Label root as must
	if len(wrapper.GetParents()) == 0 {
		//wrapper.SetLabel(Must)
	}
}

func ProcessChildrenWraps(wrapper Wrapper, logs []model.LogType, root Wrapper) {

	if wrapper == nil {
		return
	}

	for _, child := range wrapper.GetChildren(){
		if wrapper.GetLabel() == NoLabel {
			switch curr := wrapper.(type) {
			case *BlockWrapper:
				LabelIfElseBlock(curr, logs, root) //Should set to must or must not

				//If it's an if.done, or something else, set it to may (may need extra logic later)
				if curr.GetLabel() == NoLabel{
					curr.SetLabel(May)
				}
			}
		}

		ProcessChildrenWraps(child, logs, root)
	}
}
//Helper function to get topmost node where conditionals connect
func GetTopAndLabel(wrapper Wrapper, logs []model.LogType, start Wrapper, stackInfo helper.StackTraceStruct, root Wrapper) Wrapper {

	//if a new endNode is found,
	//require that many more conditional
	//nodes to be found, until the topmost
	//one is found
	curr := wrapper
	totalCount := 0
	done := false

	//Go up to very top
	for !done {

		//Each time a block containing a condition is found, increment the count
		switch curr := curr.(type) {
		case *BlockWrapper:
			condNode := curr.GetCondition()
			if condNode != nil {
				totalCount++
			}
		}

		//Move to next node up the tree
		if len(curr.GetParents()) > 0 {
			curr = curr.GetParents()[0]
		}

		//If at very top, stop
		if len(curr.GetParents()) == 0 {
			// fmt.Println("Finished going up tree")
			if totalCount == 0 {
				fmt.Println("No conditions found")
			}
			done = true
		}
	}

	//Label topmost node as must
	curr.SetLabel(Must)

	//Go down through children to label nodes
	LabelDown(curr, start, false, logs, stackInfo, totalCount, root)

	return curr
}

//Helper function used in GetTopAndLabel
func LabelDown(curr Wrapper, start Wrapper, isLog bool, logs []model.LogType, stackInfo helper.StackTraceStruct, totalCount int, root Wrapper) {

	isLogStmt := isLog
	//If at bottom, return, or if there's already a label
	if curr == start || totalCount == 0{
		return
	}

	// fmt.Println("Current wrapper in label", curr)

	//Set label downward | Process current node before processing child node
	for _, child := range curr.GetChildren() {

		//Type switch to get specific info
		//var currNodes = []ast.Node{}
		switch currType := curr.(type) {
		case *BlockWrapper:

			//If it's a condition, decrement the count
			if currType.GetCondition() != nil {
				totalCount--
			}

			//Label only if not already labeled
			//If it is a log stmt or matches regex then need to label as must
			if curr.GetLabel() == NoLabel {
				LabelIfElseBlock(currType, logs, root) //Matches logs found in an if/else block, and sets its parent's block if there is one

				//if CheckLogStatus(currNodes, logs) {
				//	isLogStmt = true
				//	//curr.SetLabel(Must)
				//	fmt.Println("Current block Matches log", currType.Block.String())
				//} else {
				//	curr.SetLabel(MustNot)
				//	fmt.Println("Current block No Matches", currType.Block.String())
				//}

				//If it's part of a log block then label as must
				//if isLog {
				//	curr.SetLabel(Must)
				//}

		}
	}


		LabelDown(child, start, isLogStmt, logs, stackInfo, totalCount, root)
	}
}

func LabelIfElseBlock(currType Wrapper, logs []model.LogType, root Wrapper){

	switch currType := currType.(type){
	case *BlockWrapper:

		//fmt.Println("Block in label If/else", currType.Block.String())

		if currType.Label == NoLabel {
			//For if.then, if.else, label must Must/MustNot
			if strings.Contains(currType.Block.String(), "if.then") || strings.Contains(currType.Block.String(), "if.else") {
				//If Log match found in an if/else, then label current block and its parent as a must
				if CheckLogStatus(currType.Block.Nodes, logs) {
					currType.SetLabel(Must)

					//Need to set the status of the condition in parent's block as well
					if len(currType.GetParents()) == 1{
						fmt.Println("Parent of ", currType.Block.String(), " set to must")
						//if currType.GetParents()[0].GetLabel() == NoLabel {
						if currType.GetParents()[0] != root.GetParents()[0] { //don't overwrite the exception block's parents
							currType.GetParents()[0].SetLabel(Must)
						}
						//}else{
						//	fmt.Println("Parent is already labeled")
						//}
					}
					fmt.Println("Current block Matches log", currType.Block.String())
				} else {
					currType.SetLabel(MustNot)

					//Need to set the status of the condition in parent's block as well
					if len(currType.GetParents()) == 1 {
						fmt.Println("Parent of ", currType.Block.String(), " set to MustNot")
						//if currType.GetParents()[0].GetLabel() == NoLabel {
						if currType.GetParents()[0] != root.GetParents()[0] { //don't overwrite the exception block's parents
							currType.GetParents()[0].SetLabel(MustNot)
						}
						//}else{
						//	fmt.Println("Parent is already labeled")
						//}
					}
					fmt.Println("Current block (no matches found)", currType.Block.String())
				}
			}
		}
	}
}

func isAssignment(node ast.Node) bool {
	var isAssignment bool = false

	//for _, node := range nodes{
		//If any node is an assignment statement, then return true
		if _, ok := node.(*ast.AssignStmt); ok {
			isAssignment = ok
			//break
		}
	//}
	return isAssignment
}

//Helper for checking if FnWrapper is a may/must (checks function with stack to see if it is relevant)
func CheckFnStatus(wrapper *FnWrapper, stackInfo helper.StackTraceStruct) bool {
	var isMust = false

	switch funcNode := wrapper.Fn.(type) {
	case *ast.FuncDecl:
		//Check if function is in the stack trace
		for _, funcName := range stackInfo.FuncName {
			fmt.Println("Node func:", funcNode.Name.Name, ", Stack Func:", funcName)
			if funcNode.Name.Name == funcName {
				isMust = true
				fmt.Println("Function ", funcNode.Name.Name, " is in the stack trace")
				break
			}
		}
	case *ast.FuncLit:
		fmt.Println(funcNode)
	}

	return isMust
}

//Helper function to check if a BlockWrapper contains a log, or if it matches a relevant regex
//Checks all nodes within a block and sees if it matches with the list of messages found in output.
func CheckLogStatus(nodes []ast.Node, logs []model.LogType) bool {

	var done = false

	for _, node := range nodes {
		if n1, ok := node.(*ast.ExprStmt); ok {
			if call, ok := n1.X.(*ast.CallExpr); ok {
				possibleLog := fmt.Sprint(call.Fun)
				if realSelector, ok := call.Fun.(*ast.SelectorExpr); ok {

					//Set status of log
					if helper.IsFromLog(realSelector) || strings.Contains(possibleLog, "log") { //if any node in the block contains a log statement, exit early
						// fmt.Println(realSelector, " is a log statement -> label as must")

						//get log from node
						for _, arg := range call.Args {
							switch argNode := arg.(type) {
							case *ast.BasicLit:
								//Match regex in the filtered logs that are found to what we find in a block
								for _, currLog := range logs {
									fullRegex := "^" + currLog.Regex + "$"
									str := strings.Trim(argNode.Value, "\"") //remove double quotes
									str = strings.ReplaceAll(str, "%d", "0") //remove flags with #'s
									if regex, err := regexp.Compile(fullRegex); err == nil {
										if regex.Match([]byte(str)) {
											// fmt.Println(str, " - MATCHES -", fullRegex)
											done = true
											return true
										}
									}
								}

							}
						}

					}
				}
			}
		}
	}

	return done
}

func MatchLogRegex(logs []model.LogType) bool {

	fset := token.NewFileSet()

	var doesMatch = false
	for _, logMsg := range logs {

		fileNode, err := parser.ParseFile(fset, logMsg.FilePath, nil, parser.ParseComments)
		if err != nil {
			continue //Skip if bad file
		}

		//Inspect the file to match regex
		ast.Inspect(fileNode, func(n ast.Node) bool {
			if call, ok := n.(*ast.CallExpr); ok {
				if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
					if helper.IsFromLog(sel) {
						if fset.Position(n.Pos()).Line == logMsg.LineNumber {
							//get log from node
							for _, arg := range call.Args {
								switch v := arg.(type) {
								case *ast.BasicLit:
									//Match regex
									if !doesMatch {
										fullRegex := "^" + logMsg.Regex + "$"
										str := strings.Trim(v.Value, "\"")
										if regex, err := regexp.Compile(fullRegex); err == nil {
											if regex.Match([]byte(str)) {
												//fmt.Println(str, " - MATCHES -", fullRegex)
												doesMatch = true
											}
										}
									}
								}
							}

							//stop
							return false
						}
					}
				}
			}
			return true
		})
	}

	return doesMatch
}
