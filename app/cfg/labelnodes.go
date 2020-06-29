package cfg

import (
	"errors"
	"fmt"
	"sourcecrawler/app/db"
	neoDb "sourcecrawler/app/db"
	"sourcecrawler/app/model"
	"strings"
)

func GrabTestNode() (db.Node, db.Node) {
	endIf2 := &neoDb.EndConditionalNode{}
	t2 := &neoDb.StatementNode{Filename: "t2", Child: endIf2}
	f2 := &neoDb.StatementNode{Filename: "f2", Child: endIf2}
	endIf2.SetParents(t2)
	endIf2.SetParents(f2)
	cond2 := &neoDb.ConditionalNode{Filename: "Cond2", TrueChild: t2, FalseChild: f2}
	t2.SetParents(cond2)
	f2.SetParents(cond2)

	endIf1 := &neoDb.EndConditionalNode{Child: cond2}
	cond2.SetParents(endIf1)
	t1 := &neoDb.FunctionNode{Filename: "t1", Child: endIf1}
	f1 := &neoDb.FunctionNode{Filename: "f1", Child: endIf1}
	endIf1.SetParents(t1)
	endIf1.SetParents(f1)
	cond1 := &neoDb.ConditionalNode{Filename: "Cond1", TrueChild: t1, FalseChild: f1}
	t1.SetParents(cond1)
	f1.SetParents(cond1)

	root := &neoDb.FunctionDeclNode{Filename: "TestRoot", Child: cond1}
	cond1.SetParents(root)

	labels := make(map[neoDb.Node]neoDb.ExecutionLabel, 0)
	labels[endIf2] = neoDb.Must
	labels[t2] = neoDb.May
	labels[f2] = neoDb.May
	labels[cond2] = neoDb.Must
	labels[endIf1] = neoDb.Must
	labels[t1] = neoDb.May
	labels[f1] = neoDb.May
	labels[cond1] = neoDb.Must
	labels[root] = neoDb.Must

	return root, endIf2
}

func GrabTestNode2() (db.Node, db.Node) {
	labels := make(map[db.Node]db.ExecutionLabel)
	leaf := &db.FunctionNode{Filename: "Leaf"}
	outerEndIf := &db.EndConditionalNode{Child: leaf}
	leaf.SetParents(outerEndIf)

	labels[leaf] = db.Must
	labels[outerEndIf] = db.Must

	// outer true branch
	trueEndIf := &db.EndConditionalNode{Child: outerEndIf}
	trueTrue := &db.FunctionNode{Filename: "T-T", Child: trueEndIf}
	trueFalse := &db.FunctionNode{Filename: "T-F", Child: trueEndIf}
	trueEndIf.SetParents(trueTrue)
	trueEndIf.SetParents(trueFalse)
	trueCond := &db.ConditionalNode{Filename: "TCond", TrueChild: trueTrue, FalseChild: trueFalse}
	trueTrue.SetParents(trueCond)
	trueFalse.SetParents(trueCond)
	trueNode2 := &db.FunctionNode{Filename: "TN2", Child: trueCond}
	trueCond.SetParents(trueNode2)
	trueNode1 := &db.FunctionNode{Filename: "TN1", Child: trueNode2}
	trueNode2.SetParents(trueNode1)

	labels[trueEndIf] = db.May
	labels[trueTrue] = db.May
	labels[trueFalse] = db.May
	labels[trueCond] = db.May
	labels[trueNode2] = db.May
	labels[trueNode1] = db.May

	// outer false branch
	falseEndIf := &db.EndConditionalNode{Child: outerEndIf}
	falseTrue := &db.FunctionNode{Filename: "F-T", Child: falseEndIf}
	falseFalse := &db.FunctionNode{Filename: "F-F", Child: falseEndIf}
	falseEndIf.SetParents(falseTrue)
	falseEndIf.SetParents(falseFalse)
	falseNode1 := &db.ConditionalNode{Filename: "FN1", TrueChild: falseTrue, FalseChild: falseFalse}
	falseTrue.SetParents(falseNode1)
	falseFalse.SetParents(falseNode1)

	outerEndIf.SetParents(trueEndIf)
	outerEndIf.SetParents(falseEndIf)

	labels[falseEndIf] = db.May
	labels[falseTrue] = db.May
	labels[falseFalse] = db.May
	labels[falseNode1] = db.May

	root := &db.ConditionalNode{Filename: "Root", TrueChild: trueNode1, FalseChild: falseNode1}
	trueNode1.SetParents(root)
	falseNode1.SetParents(root)
	labels[root] = db.Must

	return root, leaf
}

//log-if-else
func SimpleIfElse() (db.Node, db.Node, []model.LogType) {
	end := &db.EndConditionalNode{}
	t1 := &db.FunctionNode{Filename: "t1", Child: end}
	f1 := &db.StatementNode{
		Filename:   "/some/path/to/file.go",
		LogRegex:   "this is a log message: .* - f1 node",
		LineNumber: 67,
		Child:      end,
	}
	end.SetParents(t1)
	end.SetParents(f1)

	root := &db.ConditionalNode{Filename: "root", TrueChild: t1, FalseChild: f1}
	t1.SetParents(root)
	f1.SetParents(root)
	labels := make(map[db.Node]db.ExecutionLabel)
	labels[end] = db.Must
	labels[t1] = db.May
	labels[f1] = db.Must
	labels[root] = db.Must

	testLog := []model.LogType{
		{
			LineNumber: 67,
			FilePath:   "/some/path/to/file.go",
			Regex:      "this is a log message: .*",
		},
	}

	return root, end, testLog
}
//log-if-else-ext
func GrabTestNode3() (db.Node, db.Node, []model.LogType) {
	end := &db.EndConditionalNode{}
	t1 := &db.FunctionNode{Filename: "t1", Child: end}
	end.SetParents(t1)
	extraNode2 := &db.FunctionNode{Filename: "extra2", Child: end}
	end.SetParents(extraNode2)
	extraNode1 := &db.FunctionNode{Filename: "extra1", Child: extraNode2}
	extraNode2.SetParents(extraNode1)
	f1 := &db.StatementNode{
		Filename:   "file.go",
		LogRegex:   "err: .*",
		LineNumber: 2,
		Child:      extraNode1,
	}
	extraNode1.SetParents(f1)

	root := &db.ConditionalNode{Filename: "ROOT", TrueChild: t1, FalseChild: f1}
	t1.SetParents(root)
	f1.SetParents(root)
	labels := make(map[db.Node]db.ExecutionLabel)
	labels[end] = db.Must
	labels[t1] = db.May //this is not handled yet
	labels[f1] = db.Must
	labels[extraNode1] = db.Must
	labels[extraNode2] = db.Must
	labels[root] = db.Must

	testLog := []model.LogType{
		{
			LineNumber: 2,
			FilePath:   "/some/path/to/file.go",
			Regex:      "err: .*",
		},
	}

	return root, end, testLog
}
//log-nested
func LogNested()   (db.Node, db.Node, []model.LogType) {
	labels := make(map[db.Node]db.ExecutionLabel)

	end := &db.EndConditionalNode{}
	labels[end] = db.Must

	// outer false branch
	fEnd := &db.FunctionNode{Filename: "fEnd", Child: end, LineNumber: 8}
	fEndIf := &db.EndConditionalNode{Child: fEnd}
	fEnd.SetParents(fEndIf)
	ff := &db.StatementNode{
		LogRegex:   "i don't match",
		LineNumber: 7,
		Filename:   "somefile.go",
		Child:      fEndIf,
	}
	ft := &db.FunctionNode{Filename: "ft", Child: fEndIf, LineNumber: 6}
	fEndIf.SetParents(ft)
	fEndIf.SetParents(ff)
	fCond := &db.ConditionalNode{Filename: "fCond", TrueChild: ft, FalseChild: ff, LineNumber: 5}
	ff.SetParents(fCond)
	ft.SetParents(fCond)

	//TODO: not handled currently
	//labels[fEnd] = db.MustNot
	//labels[fEndIf] = db.MustNot
	//labels[ff] = db.MustNot
	//labels[ft] = db.MustNot
	//labels[fCond] = db.MustNot
	labels[fEnd] = db.May
	labels[fEndIf] = db.May
	labels[ff] = db.May
	labels[ft] = db.May
	labels[fCond] = db.May

	// outer true branch
	tEndIf := &db.EndConditionalNode{Child: end}
	tt := &db.FunctionNode{Filename: "tt", Child: tEndIf, LineNumber: 3}
	tExtraNode := &db.FunctionNode{Filename: "tExtraNode", Child: tEndIf, LineNumber: 4}
	tEndIf.SetParents(tt)
	tEndIf.SetParents(tExtraNode)
	tLog := &db.StatementNode{
		LogRegex:   "hello I am an .* error",
		LineNumber: 999,
		Filename:   "somefile.go",
		Child:      tExtraNode,
	}
	tExtraNode.SetParents(tLog)
	tCond := &db.ConditionalNode{
		TrueChild:  tt,
		FalseChild: tLog,
		LineNumber: 2,
		Filename: "tCond",
	}
	tLog.SetParents(tCond)
	tt.SetParents(tCond)

	labels[tEndIf] = db.Must
	//labels[tt] = db.MustNot
	labels[tt] = db.May
	labels[tExtraNode] = db.Must
	labels[tLog] = db.Must
	labels[tCond] = db.Must //TODO: should be must since the entire branch needs to be labeled

	end.SetParents(tEndIf)
	end.SetParents(fEnd)

	root := &db.ConditionalNode{
		TrueChild:  tCond,
		FalseChild: fCond,
		LineNumber: 1,
		Filename: "root",
	}
	tCond.SetParents(root)
	fCond.SetParents(root)
	labels[root] = db.Must

	testLog := []model.LogType{
		{
			LineNumber: 999,
			FilePath:   "somefile.go",
			Regex:      "hello I am an .* error",
		},
	}

	return root, end, testLog
}

//return labelTestCase{
//Name:   "log-if-else",
//Root:   root,
//Leaf: end,
//Labels: labels,
//Logs:   logs,
//}


// given a list of function calls in `funcCalls` and a map of their labels in `funcLabels`,
// append the names of all must-have functions to `mustHaves`, and all the may-have functions to `mayHaves`
func FilterMustMay(funcCalls []neoDb.Node, mustHaves []neoDb.Node, mayHaves []neoDb.Node, funcLabels map[string]string) ([]neoDb.Node, []neoDb.Node) {
	for _, fn := range funcCalls {
		node := fn.(*neoDb.FunctionNode)
		if label := funcLabels[node.FunctionName]; label == "must" {
			mustHaves = append(mustHaves, node)
		} else {
			mayHaves = append(mayHaves, node)
		}
	}
	return mustHaves, mayHaves
}

func MergeLabelMaps(labelMaps ...map[string]string) map[string]string {
	res := map[string]string{}
	// go through each map
	for _, currMap := range labelMaps {
		// get each function/label from this map
		for fnName, newLabel := range currMap {
			// have we added this function before?
			if existLabel, ok := res[fnName]; ok {
				// added this before, see if we need to overwrite it
				if newLabel == "must" || (newLabel == "may" && existLabel != "must") {
					res[fnName] = newLabel
				}
			} else {
				// not added before, so just add the function/label
				res[fnName] = newLabel
			}
		}
	}
	return res
}

//TODO: change branches to Must if a log appears in that branch
//Assumes starting at endIf node and tries to find topmost node
func labelBranches(end *db.EndConditionalNode, printedLogs []model.LogType) (db.Node, error) {
	var curr db.Node
	var next db.Node

	//Check for nodes without parents
	if len(end.GetParents()) != 0 {
		curr = end.GetParents()[0] //get one of the parents, doesn't matter which
		next = curr
		if len(curr.GetParents()) != 0{
			next = curr.GetParents()[0]
		}
	} else {
		return nil, fmt.Errorf("error, no parent nodes")
	}

	var top db.Node
	endCount := 1
	//if a new endNode is found,
	//require that many more conditional
	//nodes to be found, until the topmost
	//one is found
	for endCount != 0 {
		if _, isEnd := curr.(*db.EndConditionalNode); isEnd {
			endCount++
		}
		if _, isCond := curr.(*db.ConditionalNode); isCond {
			endCount--
		}
		if endCount != 0 {
			if curr == next {
				return nil, errors.New("reached top of tree without finding root conditional node")
			}
			curr = next
			if len(next.GetParents()) > 0 {
				next = next.GetParents()[0]
			}
		}
	}
	//now current is the top conditional node
	//which will be returned from this function
	top = curr
	top.SetLabel(db.Must)
	//recursively label all children up to end
	//as "may" or "must" when logs occurred
	labelBranchesRecur(top, end, printedLogs)

	//TODO: run again, but re-label log statement children nodes as must
	labelBranchesLogs(top, end, printedLogs, nil, top, false)

	//return top conditional node as next node to label
	return top, nil
}

// Returns true if it found a log statement in the branch and sets to must on the way back up
// printedLogs is the LogTypes corresponding to the logs that were all printed at runtime
func labelBranchesRecur(node db.Node, end *db.EndConditionalNode, printedLogs []model.LogType) bool {
	//hadLog := false
	for child := range node.GetChildren() {
		//stop recursion if child is already labeled
		//or if it is the original end node
		if _, ok := child.(*db.EndConditionalNode); ok {
			if child == end {
				continue
			}
		}

		//If child is not labeled, add appropriate label
		if child.GetLabel() == db.NoLabel {
			//labelBranchesRecur(child, end, printedLogs)
			//hadLog = labelBranchesRecur(child, end, printedLogs)
			//if hadLog {
			//	child.SetLabel(db.Must)
			//}else {
			//	child.SetLabel(db.May)
			//}
			labelBranchesRecur(child, end, printedLogs)
			child.SetLabel(db.May)
		}

		//if stmt, ok := node.(*db.StatementNode); ok {
		//	fmt.Println("Statement node", stmt.LineNumber, stmt.LogRegex, stmt.Filename)
		//	for _, log := range printedLogs {
		//		fmt.Println("Log info", log.LineNumber, log.Regex, log.FilePath)
		//		if log.LineNumber == stmt.LineNumber && strings.Contains(log.FilePath, stmt.Filename) {
		//			return true
		//		}
		//	}
		//}

	}

	//if hadLog {
	//	return true
	//}
	return false
}

func labelBranchesLogs(node db.Node, end *db.EndConditionalNode,
	printedLogs []model.LogType, logNode *db.StatementNode, startNode db.Node, isDone bool) (*db.StatementNode, bool){

	//If done labeling, end function (not sure if its needed. but it works?)
	if isDone{
		return logNode, isDone
	}
	done := isDone

	//If last endIf node, return back to previous node
	if node == end{
		return logNode, isDone
	}

	//Assign log label at very start of processing
	//holds the "parent must" of the branch
	var ln *db.StatementNode = nil

	//If a log statement node is in the branch, label as must (put before because the last node needs to be labeled)
	if logNode != nil{
		node.SetLabel(db.Must)
		ln = logNode
	}

	for child := range node.GetChildren() {

		//Reset for each child
		//ln = nil


		//If it's a log node then pass it to all children nodes
		if stmt, ok := node.(*db.StatementNode); ok {
			fmt.Println("Statement node", stmt.LineNumber, stmt.LogRegex, stmt.Filename)
			for _, log := range printedLogs {
				fmt.Println("Log info", log.LineNumber, log.Regex, log.FilePath)
				if log.LineNumber == stmt.LineNumber && strings.Contains(log.FilePath, stmt.Filename) {
					node.SetLabel(db.Must) //set current node
					ln = stmt
					break
				}
			}
			//If it isn't matching the log, then it is set to MustNot
			//TODO: will be if MustNot's are labeled, currently breaks the normal if-else tests
			//if ln == nil && {
			//	node.SetLabel(db.MustNot)
				//isBad = true
			//}
		}

		//stop recursion if child is already labeled
		//or if it is the original end node
		if _, ok := child.(*db.EndConditionalNode); ok {
			if child == end {
				continue
			}
		}

		ln, done = labelBranchesLogs(child, end, printedLogs, ln, startNode, isDone)
		//If the child had a log, label current as a must
		if ln != nil && !done{
			node.SetLabel(db.Must)
		}

		// THIS IS NEEDED to check if labeling is finished,
		//  since if the TRUE branch is labeled already, then ignore other side
		//TODO: bugs with mustNot assignment if the other branch is checked
		if done{
			break
		}

		//if ln != nil && !isBad{
		//	node.SetLabel(db.Must)
		//}else if isBad{
		//	node.SetLabel(db.MustNot)
		//}

	}
	//If the current node is the startNode's child, and we've seen a log node, then we are done labeling the Must branches
	if len(node.GetParents()) != 0 {
		if node.GetParents()[0] == startNode && ln != nil {
			done = true
		}
	}

	return ln, done
}

//Labels the non conditional nodes (needs testing)
// Assume root is the exception node
// start at exception, loop through iteratively through parents an endCondition node
// Then pass to labelBranches and continue
func LabelParentNodes(root db.Node, printedLogs []model.LogType) {
	if root == nil {
		fmt.Println("error root is nil")
		return
	}

	//Check if it's a leaf node
	if len(root.GetParents()) == 0 {
		fmt.Println("No parent nodes connected")
		return
	}

	//Test logs
	for _, log := range printedLogs{
		fmt.Println(log)
	}

	//Label exception node itself (has to be a must)
	//root.SetLabel(db.Must)

	var trueNode db.Node
	var falseNode db.Node
	var top db.Node

	//Goes through each parent node of the exception node
	//for _, parent := range root.GetParents() {
	//
	//	//Process next parent if node is nil
	//	if parent == nil {
	//		fmt.Println("nil parent node")
	//		continue
	//	}
	//
	//	//next = parent
	//	fmt.Println("Parent node is", parent.GetProperties())

	node := root

	//Loop through all the way up the chain of parents
	for node != nil {
		// Add label if not already labeled
		if node.GetLabel() == db.NoLabel {
			//fmt.Println("Labeling ", node.GetProperties())
			switch node := node.(type) {
			case *db.FunctionNode:
				node.SetLabel(db.Must)
			case *db.FunctionDeclNode:
				node.SetLabel(db.Must)
			case *db.StatementNode:
				node.SetLabel(db.Must)
			case *db.ReturnNode:
				node.SetLabel(db.Must)
			case *db.ConditionalNode:
				if node != top {
					node.SetLabel(db.May)
					node.TrueChild.SetLabel(db.May)
					node.FalseChild.SetLabel(db.May)
					if trueNode != nil {
						trueNode.SetLabel(db.May)
					}
					if falseNode != nil {
						falseNode.SetLabel(db.May)
					}
				}
			case *db.EndConditionalNode:
				node.SetLabel(db.Must)
				topNode, err := labelBranches(node,printedLogs) //special case if its an endIf node
				if topNode == nil {
					if err != nil {
						fmt.Println(err)
					}
					fmt.Println("Error retrieving topmost node")
				} else {
					fmt.Println("Topmost node is", topNode.GetProperties())
				}
				top = topNode
			}
		} else {
			//fmt.Println("Node", node.GetProperties(), "is already labeled")
		}

		//If it's an end conditional, set the next parent to the topmost node of the conditional
		// and continue
		switch node.(type) {
		case *db.EndConditionalNode:
			node = top
			continue
		}

		// Check if parent node exists and set to next
		// If no more parents, it is done processing
		if len(node.GetParents()) == 1 {
			node = node.GetParents()[0]
		} else if len(node.GetParents()) == 2 {
			trueNode = node.GetParents()[0]
			falseNode = node.GetParents()[1]
		} else {
			break
		}
	}
	//}
}

//recursive version -- temporary
//func LabelParentNodesRecur(root db.Node, printedLogs []model.LogType) {
//
//	if root == nil {
//		fmt.Println("error root is nil")
//		return
//	}
//
//	if len(root.GetParents()) == 0 {
//		fmt.Println("No parent nodes connected")
//	}
//
//	fmt.Println(root.GetParents())
//	fmt.Println(len(root.GetParents()))
//
//	//Currently using a recursive version to process parent nodes when coming back up from recursive calls
//	switch root := root.(type) {
//	case *db.FunctionNode:
//		LabelParentNodesRecur(root.Child, printedLogs)
//		//fmt.Println("function node",root.GetProperties())
//	case *db.FunctionDeclNode:
//		LabelParentNodesRecur(root.Child, printedLogs)
//		//fmt.Println("FuncDecl",root.GetProperties())
//	case *db.StatementNode:
//		LabelParentNodesRecur(root.Child, printedLogs)
//		//fmt.Println("statement",root.GetProperties())
//	case *db.ReturnNode:
//		LabelParentNodesRecur(root.Child, printedLogs)
//		//fmt.Println("return",root.GetProperties())
//	case *db.ConditionalNode:
//		LabelParentNodesRecur(root.TrueChild, printedLogs) //should arrive at an end conditional node and fall into case below
//		LabelParentNodesRecur(root.FalseChild, printedLogs)
//	case *db.EndConditionalNode: //TODO: Bug with wrong labeling of returns inside a conditional
//		topNode, err := labelBranches(*root, printedLogs) //special case if its an endIf node
//		if err != nil || topNode == nil {
//			fmt.Println("Error retrieving topmost node")
//		} else {
//			fmt.Println("Topmost node is", topNode.GetProperties())
//		}
//	default:
//		//fmt.Println("default")
//	}
//
//	//Label current node if not labeled(the exception node)
//	if root.GetLabel() == db.NoLabel {
//		switch nodeType := root.(type) {
//		case *db.ReturnNode:
//			root.SetLabel(db.Must)
//		case *db.EndConditionalNode:
//			fmt.Println(nodeType, " was already labeled")
//		case *db.ConditionalNode:
//			root.SetLabel(db.May)
//		default:
//			root.SetLabel(db.Must) //Set label to must for non-end conditional nodes
//			//fmt.Println("Labeling -> ", root.GetProperties())
//		}
//	} else {
//		fmt.Println("Node", root.GetProperties(), " is already labeled")
//	}
//}
