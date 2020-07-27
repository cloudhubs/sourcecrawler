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

				// fmt.Println("Current wrapper in label", curr)

				//Entry should be a must
				if strings.Contains(wrap.Block.String(), "entry") {
					wrap.SetLabel(Must)
				} else {
					wrap.SetLabel(May)
				}

				//If it is part of an if-then or if-else, it is labeled as may
				if strings.Contains(wrap.Block.String(), "if.then") ||
					strings.Contains(wrap.Block.String(), "if.else") {
					wrap.SetLabel(May)
				}

				//If-done should always be a must
				// if strings.Contains(wrap.Block.String(), "if.done") {
				// 	wrap.SetLabel(Must)
				// }

				//Only true if a block has 2 successors
				if wrap.GetCondition() != nil {
					wrap.SetLabel(May)
				}

				//If two parents, go up to top and label down
				if len(wrap.GetParents()) == 2 {
					wrapper = GetTopAndLabel(wrap, logs, wrap, stackInfo)
				}

				//Check for possible log msg and log matchings
				if CheckLogStatus(wrap.Block.Nodes, logs) {
					wrap.SetLabel(Must)
				}
			}
		} else {
			fmt.Println("Wrapper is already labeled", wrapper)
			return
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

//Helper function to get topmost node where conditionals connect
func GetTopAndLabel(wrapper Wrapper, logs []model.LogType, start Wrapper, stackInfo helper.StackTraceStruct) Wrapper {

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

	isLog := false

	//Go down through children to label nodes
	LabelDown(curr, start, isLog, logs, stackInfo, totalCount)

	return curr
}

//Helper function used in GetTopAndLabel
func LabelDown(curr Wrapper, start Wrapper, isLog bool, logs []model.LogType, stackInfo helper.StackTraceStruct, totalCount int) {

	//If at bottom, return
	if curr == start || totalCount == 0 {
		curr.SetLabel(Must)
		return
	}

	// fmt.Println("Current wrapper in label", curr)

	//Set label downward
	for _, child := range curr.GetChildren() {

		//Type switch to get specific info
		var currNodes = []ast.Node{}
		switch currType := curr.(type) {
		case *BlockWrapper:
			currNodes = currType.Block.Nodes

			//If it's a condition, decrement the count
			if currType.GetCondition() != nil {
				totalCount--
			}
		}

		//If it is a log stmt or matches regex then need to label as must
		if CheckLogStatus(currNodes, logs) {
			isLog = true
			curr.SetLabel(Must)
		}else{
			curr.SetLabel(May)
		}

		//If it's part of a log block then label as must (not catching else conditions in an if-else??)
		if isLog {
			curr.SetLabel(Must)
		}

		LabelDown(child, start, isLog, logs, stackInfo, totalCount)
	}
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
								//Match regex to possible log
								for _, currLog := range logs {
									fullRegex := "^" + currLog.Regex + "$"
									str := strings.Trim(argNode.Value, "\"") //remove double quotes
									str = strings.ReplaceAll(str, "%d", "0") //remove flags with #'s
									if regex, err := regexp.Compile(fullRegex); err == nil {
										if regex.Match([]byte(str)) {
											fmt.Println(str, " - MATCHES -", fullRegex)
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
