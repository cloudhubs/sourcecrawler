package cfg

import (
	"errors"
	"sourcecrawler/app/db"
)

// ConnectFnCfgs takes as input all of the function declaration
// roots where every other function declaration in the slice
// should be checked for calls to that function and given
// a copy of its declaration to reference for each call.
func ConnectFnCfgs(funcs []db.Node) []db.Node {
	unecessaryDecls := make(map[int]struct{})
	for i := 0; i < 3; i++ {
		for j, fn := range funcs {
			for k, otherFn := range funcs {
				if j != k {
					if foundCalls := connectCallsToDecls(fn, otherFn); foundCalls {
						unecessaryDecls[j+1] = struct{}{}
					}
				}
			}
		}
	}

	for j := len(funcs) - 1; j >= 0; j-- {
		if _, ok := unecessaryDecls[j]; ok {
			funcs = append(funcs[0:j], funcs[j+1:]...)
		}
	}

	return funcs
}

// parent is the function whose children call decl
// return true if connection was made, false else
func connectCallsToDecls(parent db.Node, decl db.Node) bool {
	decl, ok := decl.(*db.FunctionDeclNode)
	if decl == nil || parent == nil || !ok {
		return false
	}
	return ConnectRefsToDecl(parent, decl)
}

// ConnectRefsToDecl connects all function call node children
// in `fn` and connects them to copies of `decl`
func ConnectRefsToDecl(fn db.Node, decl db.Node) (foundRef bool) {
	foundRef = false
	refs := getReferences(decl.(*db.FunctionDeclNode), fn)
	if refs == nil || len(refs) == 0 {
		return
	}

	for _, ref := range refs {
		if _, ok := ref.Child.(*db.FunctionDeclNode); ok {
			continue
		}

		foundRef = true
		copy := CopyCfg(decl)
		child := ref.Child
		child.SetParents(ref) //TODO: ??

		ref.Child = copy

		for _, leaf := range getLeafNodes(copy) {
			if _, ok := leaf.(*db.ConditionalNode); ok || leaf == ref {
				continue
			}
			leaf.SetChild([]db.Node{child})
		}
	}
	return
}

func getReference(fn *db.FunctionDeclNode, parent db.Node) (*db.FunctionNode, error) {
	if refs := getReferences(fn, parent); len(refs) > 0 {
		return refs[0], nil
	}
	return nil, errors.New("No reference found")
}

func getReferences(fn *db.FunctionDeclNode, parent db.Node) []*db.FunctionNode {
	return getReferencesRecur(fn, parent, make([]*db.FunctionNode, 0))
}

func getReferencesRecur(fn *db.FunctionDeclNode, parent db.Node, refs []*db.FunctionNode) []*db.FunctionNode {
	if parent == nil {
		return refs
	}
	for node := range parent.GetChildren() {
		if node, ok := node.(*db.FunctionNode); ok && node.FunctionName == fn.FunctionName {
			if node != nil {
				refs = append(refs, node)
			}
		}
		if node != nil {
			refs = append(refs, getReferencesRecur(fn, node, refs)...)
		}
	}
	return refs
}

func getLeafNodes(fn db.Node) []db.Node {
	rets := []db.Node{}
	for node := range fn.GetChildren() {
		if node == nil {
			continue
		}
		//if a child is nil (for conditionals
		//either both will be nil or neither
		//so only one needs to be checked)
		//this is a return statement, otherwise,
		//call this function on the node only
		//once then break the loop
		if len(node.GetChildren()) > 0 {
			rets = append(rets, getLeafNodes(node)...)
		} else {
			rets = append(rets, node)
		}
	}
	return rets
}

func ConnectStackTrace(fns []db.Node) {
	for i, fn := range fns {
		if i < len(fns)-1 {
			//gather return statements to connect
			returnStmts := getLeafNodes(fn)

			//find reference to fn in parent fn
			referenceNode, err := getReference(fn.(*db.FunctionDeclNode), fns[i+1])
			if err != nil {
				panic(err)
			}

			//get reference to parent's child
			child := referenceNode.Child
			child.(*db.FunctionDeclNode).Parent = referenceNode //??

			//connect parent refernce to fn
			referenceNode.Child = fn
			//and return statements to parent child
			for _, returnStmt := range returnStmts {
				returnStmt.SetChild([]db.Node{child})
			}
		}
	}
}
