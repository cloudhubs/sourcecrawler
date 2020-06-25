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
	//as "may"
	hadLog := labelBranchesRecur(top, end, nil)
	if hadLog {
		top.SetLabel(db.Must)
	}

	//return top conditional node as next node to label
	return top, nil
}

// Returns true if it found a log statement in the branch and sets to must on the way back up
// printedLogs is the LogTypes corresponding to the logs that were all printed at runtime
func labelBranchesRecur(node db.Node, end *db.EndConditionalNode, printedLogs []model.LogType) bool {
	for _,log := range printedLogs {
		fmt.Println(log)
	}
	for child := range node.GetChildren() {
		//stop recursion if child is already labeled
		//or if it is the original end node
		if _, ok := child.(*db.EndConditionalNode); ok {
			if child == end {
				continue
			}
		}

		if child.GetLabel() == db.NoLabel {
			hadLog := labelBranchesRecur(child, end, printedLogs)
			if hadLog {
				child.SetLabel(db.Must)
				return true
			}else {
				child.SetLabel(db.May)
			}
		}

	}
	if stmt, ok := node.(*db.StatementNode); ok {
		fmt.Println("It's a statement")
		for _, log := range printedLogs {
			fmt.Println("There are logs to check")
			if log.LineNumber == stmt.LineNumber && strings.Contains(log.FilePath, stmt.Filename) {
				fmt.Println("HEY, LISTEN",node.GetProperties())
				return true
			}
		}
	}
	return false
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
