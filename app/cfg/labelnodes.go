package cfg

import (
	"errors"
	"fmt"
	"sourcecrawler/app/db"
	neoDb "sourcecrawler/app/db"
	"strings"

	//"strings"
)

func GrabTestNode(root db.Node) db.Node{
	var testNode db.Node
	for n1 := range root.GetChildren(){
		//fmt.Println(n1.GetProperties())
		for n2 := range n1.GetChildren(){
			//fmt.Println(n2.GetProperties())
			for n3 := range n2.GetChildren(){
				//fmt.Println(n3.GetProperties())
				for n4 := range n3.GetChildren(){
					//fmt.Println(n4.GetProperties())
					for n5 := range n4.GetChildren(){
						//fmt.Println(n5.GetProperties())
						for n6 := range n5.GetChildren(){
							//fmt.Println(n6.GetProperties())
							for n7 := range n6.GetChildren(){
								//fmt.Println(n7.GetProperties())
								for n8 := range n7.GetChildren(){
									if n8 != nil && strings.Contains(n8.GetProperties(), "warning"){
										testNode = n8
										return testNode
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return testNode
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
func LabelParentNodes(root db.Node) {
	if root == nil {
		fmt.Errorf("Error: nil root node")
		return
	}

	if len(root.GetParents()) == 0{
		fmt.Errorf("Error, no parent nodes connected to this node")
	} else{
		fmt.Println("Parent nodes exist")
	}

	//Iteratively using double linked list
	for _, parent := range root.GetParents() {

		node := parent
		fmt.Println("Node is", node.GetProperties())

		//Process next parent if node is nil
		if node == nil {
			fmt.Println("nil parent node")
			continue
		}

		// Add label if not already labeled
		if node.GetLabel() == db.NoLabel {
			fmt.Println("Node is", node.GetProperties())
			switch node := node.(type) {
			case *db.FunctionNode:
				node.SetLabel(db.Must)
				//fmt.Println("function node",root.GetProperties())
			case *db.FunctionDeclNode:
				node.SetLabel(db.Must)
				//fmt.Println("FuncDecl",root.GetProperties())
			case *db.StatementNode:
				node.SetLabel(db.Must)
				//fmt.Println("statement",root.GetProperties())
			case *db.ReturnNode:
				node.SetLabel(db.Must)
				//fmt.Println("return",root.GetProperties())
			case *db.ConditionalNode:
				node.SetLabel(db.May)
				node.TrueChild.SetLabel(db.May)
				node.FalseChild.SetLabel(db.May)
				//node.SetLabel(db.May)
				//node.SetLabel(db.May)
			case *db.EndConditionalNode:
				//topNode, err := labelBranches(*node) //special case if its an endIf node
				//if err != nil || topNode == nil {
				//	fmt.Println("Error retrieving topmost node")
				//} else {
				//	fmt.Println("Topmost node is", topNode.GetProperties())
				//}
			}
		}else{
			fmt.Println("Node", node.GetProperties(), "is already labeled")
		}
	}

	//Currently using a recursive version to process parent nodes when coming back up from recursive calls
	//switch root := root.(type) {
	//case *db.FunctionNode:
	//	LabelNonCondNodes(root.Child)
	//	//fmt.Println("function node",root.GetProperties())
	//case *db.FunctionDeclNode:
	//	LabelNonCondNodes(root.Child)
	//	//fmt.Println("FuncDecl",root.GetProperties())
	//case *db.StatementNode:
	//	LabelNonCondNodes(root.Child)
	//	//fmt.Println("statement",root.GetProperties())
	//case *db.ReturnNode:
	//	LabelNonCondNodes(root.Child)
	//	//fmt.Println("return",root.GetProperties())
	//case *db.ConditionalNode:
	//	LabelNonCondNodes(root.TrueChild) //should arrive at an end conditional node and fall into case below
	//	LabelNonCondNodes(root.FalseChild)
	//case *db.EndConditionalNode: //TODO: Bug with wrong labeling of returns inside a conditional
	//	topNode, err := labelBranches(*root)	//special case if its an endIf node
	//	if err != nil || topNode == nil{
	//		fmt.Println("Error retrieving topmost node")
	//	}else{
	//		fmt.Println("Topmost node is",topNode.GetProperties())
	//	}
	//default:
	//	//fmt.Println("default")
	//}

	//Label current node if not labeled(the exception node)
	//if root.GetLabel() == db.NoLabel{
	//	switch nodeType := root.(type) {
	//	case *db.ReturnNode:
	//		root.SetLabel(db.Must)
	//	case *db.EndConditionalNode:
	//		fmt.Println(nodeType, " was already labeled")
	//	case *db.ConditionalNode:
	//		root.SetLabel(db.May)
	//	default:
	//		root.SetLabel(db.Must) //Set label to must for non-end conditional nodes
	//		//fmt.Println("Labeling -> ", root.GetProperties())
	//	}
	//}else{
	//	fmt.Println("Node", root.GetProperties(), " is already labeled")
	//}

}
