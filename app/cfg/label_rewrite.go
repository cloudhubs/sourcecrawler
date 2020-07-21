package cfg

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"sourcecrawler/app/helper"
	"sourcecrawler/app/logsource"
	"sourcecrawler/app/model"
	"strings"
)

//---------- Labeling feature for Must/May-haves (rewrite) --------------
//Assumptions: CFG tree already created
func (paths *PathList) LabelCFG(curr Wrapper, logs []model.LogType, root Wrapper, stackInfo []helper.StackTraceStruct) {

	wrapper := curr //holds current wrapper
	fmt.Println("Wrapper in label", wrapper)

	//Iterate up through parents up to root
	for len(wrapper.GetParents()) > 0 {
		if wrapper.GetLabel() == NoLabel {
			switch wrap := wrapper.(type) {
			case *FnWrapper:
				//If it's a function in the stack trace, then it must run
				//TODO: (Need to give it stack trace info from somewhere for matching) -
				// -circular dependency bug if stackTraceStruct is passed (need to refactor)
				//fnwrapper connected to blockWrapper succs
				if CheckFnStatus(wrap) {
					wrap.SetLabel(Must)
				} else {
					wrap.SetLabel(May)
				}

			case *BlockWrapper: //BlockWrapper can represent a condition, but could be a statement, etc
				//Check if it's a condition, if not set as must

				//If it is part of an if-then or if-else, it is labeled as may
				if strings.Contains(wrap.Block.String(), "if.then") ||
					strings.Contains(wrap.Block.String(), "if.else") {
					wrap.SetLabel(May)
				}

				//If-done should always be a must
				if strings.Contains(wrap.Block.String(), "if.done") {
					wrap.SetLabel(Must)
				}

				//Only true if a block has 2 successors
				//if wrap.GetCondition() != nil {
				//	wrap.SetLabel(May)
				//}

				//If two parents, go up to top and label down
				if len(wrap.GetParents()) == 2 {
					wrapper = GetTopAndLabel(wrap, logs, wrap)
				}

				//Check for possible log msg and log matchings
				if CheckLogStatus(wrap.Block.Nodes) {
					//Add log label functionality
					fmt.Println("Add log labeling stuff")
					wrap.SetLabel(Must)
				}
			}
		} else {
			fmt.Println("Wrapper is already labeled", wrapper)
		}

		//Set next wrapper (parents) - two parent case should already be handled already by GetTopAndLabel
		if len(wrapper.GetParents()) == 1 {
			wrapper = wrapper.GetParents()[0]
		}
	}

	//Label root as must
	if len(wrapper.GetParents()) == 0 {
		wrapper.SetLabel(Must)
	}
}

//Helper function to get topmost node where conditionals connect
func GetTopAndLabel(wrapper Wrapper, logs []model.LogType, start Wrapper) Wrapper {

	//Go up until a node with 2 children are found (top condition)
	curr := wrapper
	for len(curr.GetParents()) > 0 && len(curr.GetChildren()) != 2 {
		curr = curr.GetParents()[0]

		if len(curr.GetChildren()) == 2 {
			break
		}
	}

	fmt.Println("Topmost wrapper", curr)

	//Label topmost node as must
	curr.SetLabel(Must)

	var isLog bool = false

	//Go down through children to label nodes
	LabelDown(curr, start, isLog, logs)

	return curr
}

//Helper function used in GetTopAndLabel
func LabelDown(curr Wrapper, start Wrapper, isLog bool, logs []model.LogType) {

	//If at bottom, return
	if curr == start {
		curr.SetLabel(Must)
		return
	}

	//Set label downward
	for _, child := range curr.GetChildren() {

		//Type switch to get specific info
		var currNodes = []ast.Node{}
		switch currType := curr.(type) {
		case *BlockWrapper:
			currNodes = currType.Block.Nodes
		}

		//If there is a regex match,
		if MatchLogRegex(logs) {
			curr.SetLabel(Must)
			isLog = true
		}

		//If it is a log then need to label as must
		if CheckLogStatus(currNodes) {
			isLog = true
		}

		//If it's part of a log branch then label as must
		if isLog {
			curr.SetLabel(Must)
			// fmt.Println("IS LOG NODE")
		} else {
			curr.SetLabel(May)
		}

		LabelDown(child, start, isLog, logs)
	}
}

//Helper for checking if FnWrapper is a may/must (checks function with stack to see if it is relevant)
func CheckFnStatus(wrapper *FnWrapper) bool {
	var isMust = false

	switch funcNode := wrapper.Fn.(type) {
	case *ast.FuncDecl:
		//Check if function is in the stack trace
		//for _, stkInfo := range stackInfo{
		//	for _, funcName := range stkInfo.FuncName {
		//		if funcNode.Name.Name == funcName{
		//			isMust = true
		//			break
		//		}
		//	}
		//}
	case *ast.FuncLit:
		fmt.Println(funcNode)
	}

	return isMust
}

//Helper function to check if a BlockWrapper contains a log, or if it matches
func CheckLogStatus(nodes []ast.Node) bool {
	for _, node := range nodes {
		if n1, ok := node.(*ast.ExprStmt); ok {
			if call, ok := n1.X.(*ast.CallExpr); ok {
				if realSelector, ok := call.Fun.(*ast.SelectorExpr); ok {
					if logsource.IsFromLog(realSelector) { //if any node in the block contains a log statement, exit early
						// fmt.Println(realSelector, " is a log statement -> label as must")
						return true
					}
				}
			}
		}

	}

	return false
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
					if logsource.IsFromLog(sel) {
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
												// fmt.Println(str, " - MATCHES -", fullRegex)
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
