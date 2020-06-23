package cfg

import (
	"errors"
	"fmt"
	"sourcecrawler/app/db"
)

func labelBranches(end db.EndConditionalNode) (db.Node, error) {
	curr := end.GetParents()[0] //get one of the parents, doesn't matter which
	next := curr.GetParents()[0]
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
func labelNonCondNodes(root db.Node) {
	if root == nil {
		return
	}

	for childNode := range root.GetChildren() {
		//End if child is nil
		if childNode == nil {
			continue
		}

		// Add label to different types of nodes if no label
		// Types are placeholders in case we need specific functionality for each.
		if childNode.GetLabel() == db.NoLabel {
			switch root := root.(type) {
			case *db.FunctionNode:
				root.SetLabel(db.May)
			case *db.FunctionDeclNode:
				root.SetLabel(db.May)
			case *db.StatementNode:
				root.SetLabel(db.May)
			case *db.ReturnNode:
				root.SetLabel(db.May)
			default:
				fmt.Println("Default")
			}
		} else {
			fmt.Println("Node", childNode.GetProperties(), "is already labeled")
		}
	}

}
