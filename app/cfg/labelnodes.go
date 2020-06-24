package cfg

import (
	"errors"
	"fmt"
	"sourcecrawler/app/db"
)

//Assumes starting at endIf node and tries to find topmost node
func labelBranches(end db.EndConditionalNode) (db.Node, error) {

	var curr db.Node
	var next db.Node

	//Check for nodes without parents
	if len(end.GetParents()) != 0 {
		curr = end.GetParents()[0] //get one of the parents, doesn't matter which
		next = curr.GetParents()[0]
	}else{
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
	//recursively label all children up to end
	//as "may"
	labelBranchesRecur(top, end)

	//return top conditional node as next node to label
	return top, nil
}

func labelBranchesRecur(node db.Node, end db.EndConditionalNode) {
	for child := range node.GetChildren() {
		//stop recursion if child is already labeled
		//or if it is the original end node
		if child, ok := child.(*db.EndConditionalNode); ok {
			if child.GetLabel() == db.NoLabel && child != &end {
				child.SetLabel(db.May)
				labelBranchesRecur(child, end)
			}
		}
	}
}

//Labels the non conditional nodes (needs testing)
// Assume root is the exception node
// start at exception, loop through iteratively through parents an endCondition node
// Then pass to labelBranches and continue
func LabelNonCondNodes(root db.Node) {
	if root == nil {
		fmt.Errorf("Error: nil root node")
		return
	}

	//Currently using a recursive version to process parent nodes when coming back up from recursive calls
	switch root := root.(type) {
	case *db.FunctionNode:
		LabelNonCondNodes(root.Child)
		//fmt.Println("function node",root.GetProperties())
	case *db.FunctionDeclNode:
		LabelNonCondNodes(root.Child)
		//fmt.Println("FuncDecl",root.GetProperties())
	case *db.StatementNode:
		LabelNonCondNodes(root.Child)
		//fmt.Println("statement",root.GetProperties())
	case *db.ReturnNode:
		LabelNonCondNodes(root.Child)
		//fmt.Println("return",root.GetProperties())
	case *db.ConditionalNode:
		LabelNonCondNodes(root.TrueChild) //should arrive at an end conditional node and fall into case below
		LabelNonCondNodes(root.FalseChild)
	case *db.EndConditionalNode: //TODO: Bug with wrong labeling of returns inside a conditional
		topNode, err := labelBranches(*root)	//special case if its an endIf node
		if err != nil || topNode == nil{
			fmt.Println("Error retrieving topmost node")
		}else{
			fmt.Println("Topmost node is",topNode.GetProperties())
		}
	default:
		//fmt.Println("default")
	}

	//Label current node if not labeled(the exception node)
	if root.GetLabel() == db.NoLabel{
		switch nodeType := root.(type) {
		case *db.ReturnNode:
			root.SetLabel(db.Must)
		case *db.EndConditionalNode:
			fmt.Println(nodeType, " was already labeled")
		case *db.ConditionalNode:
			root.SetLabel(db.May)
		default:
			root.SetLabel(db.Must) //Set label to must for non-end conditional nodes
			//fmt.Println("Labeling -> ", root.GetProperties())
		}
	}else{
		fmt.Println("Node", root.GetProperties(), " is already labeled")
	}



	//TODO: Can't get every child of every level connected with parents iteratively
	// causes some parent nodes to be unlabeled if there are nested children nodes
	//for index := range root.GetParents() {
	//
	//	node := &root.GetParents()[index]
	//	fmt.Println("Node is", (*node).GetProperties())
	//
	//	//End if parent nodes are nil
	//	if node == nil {
	//		fmt.Println("nil parent node")
	//		continue
	//	}
	//
	//	// Add label to different types of nodes if no label
	//	// Types are placeholders in case we need specific functionality for each.
	//	if (*node).GetLabel() == db.NoLabel {
	//		fmt.Println("Node is", (*node).GetProperties())
	//		switch node := (*node).(type) {
	//		case *db.FunctionNode:
	//			node.SetLabel(db.Must)
	//			//fmt.Println("function node",root.GetProperties())
	//		case *db.FunctionDeclNode:
	//			node.SetLabel(db.Must)
	//			//fmt.Println("FuncDecl",root.GetProperties())
	//		case *db.StatementNode:
	//			node.SetLabel(db.Must)
	//			//fmt.Println("statement",root.GetProperties())
	//		case *db.ReturnNode:
	//			node.SetLabel(db.Must)
	//			//fmt.Println("return",root.GetProperties())
	//		case *db.ConditionalNode:
	//			node.SetLabel(db.May)
	//			node.SetLabel(db.May)
	//		case *db.EndConditionalNode:
	//			topNode, err := labelBranches(*node) //special case if its an endIf node
	//			if err != nil || topNode == nil {
	//				fmt.Println("Error retrieving topmost node")
	//			} else {
	//				fmt.Println("Topmost node is", topNode.GetProperties())
	//			}
	//		}
	//	}else{
	//		fmt.Println("Node", (*node).GetProperties(), "is already labeled")
	//	}
	//}
}
