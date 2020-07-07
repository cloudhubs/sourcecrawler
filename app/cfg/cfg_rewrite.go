package cfg

import (
	"fmt"
	"go/ast"
	"sourcecrawler/app/logsource"

	"golang.org/x/tools/go/cfg"
)

type Wrapper interface {
	AddParent(w Wrapper)
	GetParents(w Wrapper) []Wrapper
	AddChild(w Wrapper)
	GetChildren() []Wrapper
	GetOuterWrapper() Wrapper
	SetOuterWrapper(w Wrapper)
}

type FnWrapper struct {
	Fn         ast.Node // *ast.FuncDel or *ast.FuncLit
	FirstBlock Wrapper
	Parents    []Wrapper
	Outer      Wrapper
	// ...?
}

type BlockWrapper struct {
	Block   *cfg.Block
	Parents []Wrapper
	Succs   []Wrapper
	Outer   Wrapper
	// ...
	// method to get condition (can return nil if not conditional)
}

// ------------------ FnWrapper ----------------------

func (fn *FnWrapper) AddParent(w Wrapper) {
	if w != nil {
		fn.Parents = append(fn.Parents, w)
	}
}

func (fn *FnWrapper) GetParents(w Wrapper) []Wrapper {
	return fn.Parents
}

func (fn *FnWrapper) AddChild(w Wrapper) {
	fn.FirstBlock = w
}

func (fn *FnWrapper) GetChildren() []Wrapper {
	if fn.FirstBlock == nil {
		return []Wrapper{}
	}
	return []Wrapper{fn.FirstBlock}
}

func (fn *FnWrapper) GetOuterWrapper() Wrapper {
	return fn.Outer
}

func (fn *FnWrapper) SetOuterWrapper(w Wrapper) {
	fn.Outer = w
}

// ------------------ BlockWrapper ----------------------

func (b *BlockWrapper) AddParent(w Wrapper) {
	if w != nil {
		b.Parents = append(b.Parents, w)
	}
}

func (b *BlockWrapper) GetParents(w Wrapper) []Wrapper {
	return b.Parents
}

func (b *BlockWrapper) AddChild(w Wrapper) {
	if w != nil {
		b.Succs = append(b.Succs, w)
	}
}

func (b *BlockWrapper) GetChildren() []Wrapper {
	return b.Succs
}

func (b *BlockWrapper) GetOuterWrapper() Wrapper {
	return b.Outer
}

func (b *BlockWrapper) SetOuterWrapper(w Wrapper) {
	b.Outer = w
}

//Method to get condition, nil if not a conditional (specific to block wrapper)
func (b *BlockWrapper) GetCondition() string{

	var condition string = ""
	//Return block or panic, fatal, etc
	if len(b.Succs) == 0{
		return ""
	}
	//Normal block
	if len(b.Succs) == 1 {
		return ""
	}
	//Conditional block (TODO: assuming there's only 1 if stmt per block)
	if len(b.Succs) == 2 && b.Block != nil && len(b.Block.Nodes) > 0{
		for _, node := range b.Block.Nodes{
			ast.Inspect(node, func(currNode ast.Node) bool {
				if ifNode, ok := node.(*ast.IfStmt); ok{
					condition = fmt.Sprint(ifNode.Cond)
				}

				if forNode, ok := node.(*ast.ForStmt); ok{
					fmt.Println("For loop condition", forNode.Cond)
				}
				return true
			})
		}
	}

	return condition
}

// ---- Traversal function (currently a DFS -> goes from root to all children)---
// ----- will need to handle variables later -----
// cfg -> starting block | condStmts -> holds conditional expressions | outerRoot -> outermost wrapper
//
func traverseCFG(cfg Wrapper, condStmts []string, outerRoot Wrapper){

	//Process and exit if there are no more children (last block)
	if len(cfg.GetChildren()) == 0 {
		return
	}

	//Go through each child in the tree (list of blocks)
	for _, child := range cfg.GetChildren(){
		child.SetOuterWrapper(outerRoot) //Set the outer wrapper for each child

		//Check if if is a FnWrapper or BlockWrapper Type
		switch currWrapper := cfg.(type){
		case *FnWrapper:
			fmt.Println("FnWrapper", currWrapper)

		case *BlockWrapper:
			fmt.Println("BlockWrapper", currWrapper)

			//If conditional block, extract condition and process the true and false child
			condition := currWrapper.GetCondition()

			if condition != ""{
				condStmts = append(condStmts, condition)

				traverseCFG(currWrapper.Succs[0], condStmts, outerRoot) //true child
				traverseCFG(currWrapper.Succs[1], condStmts, outerRoot) //false child
			}
		}

		traverseCFG(child, condStmts, outerRoot)
	}


}

//Extract logging statements from a cfg block
func extractLogRegex(block *cfg.Block) (regexes []string){

	//For each node inside the block, check if it contains logging stmts
	for _, currNode := range block.Nodes{
		ast.Inspect(currNode, func(node ast.Node) bool {
			if call, ok := node.(*ast.CallExpr); ok {
				if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
					if logsource.IsFromLog(sel) {
						//get log regex from the node
						for _, arg := range call.Args {
							switch logNode := arg.(type) {
							case *ast.BasicLit:
								regexStr := logsource.CreateRegex(logNode.Value)
								regexes = append(regexes, regexStr)
							}
							//Currently an runtime bug with catching Msgf ->
						}
					}
				}
			}
			return true
		})
	}

	return regexes
}

// func NewCfgWrapper(first *cfg.Block) *CfgWrapper {
// 	return &CfgWrapper{
// 		FirstBlock: NewBlockWrapper(first, nil),
// 	}
// }

// call with nil parent if it's a root block
// func NewBlockWrapper(block *cfg.Block, parent *BlockWrapper) *BlockWrapper {
// 	b := &BlockWrapper{
// 		Block: block,
// 		// Parent: parent,
// 		Succs: make([]*BlockWrapper, 0),
// 	}
// 	for _, succ := range block.Succs {
// 		b.Succs = append(b.Succs, NewBlockWrapper(succ, b)) // right now this will create duplicate wrappers, need caching
// 	}
// 	// need to construct block wrappers for each function literal found
// 	return b
// }

// // Usage assumes you have all the wrapped function blocks already:
// // for each function fn:
// //   for each other function fn2:
// //     fn.connectBlockCalls(fn2)
// func (b *BlockWrapper) connectBlockCalls(fn *BlockWrapper) {
// 	if b.Block == nil {
// 		return
// 	}
// 	for _, _ /*node :*/ = range b.Block.Nodes {
// 		// if node is a function call that corresponds to fn
// 		// slice the Nodes in half and set the successor node of the current
// 		// block to the function, and the function's successors to the
// 		// old block successors, and modify parents accordingly.
// 	}
// }

// func (b *BlockWrapper) getCondition() string {
// 	if len(b.Succs) == 2 && b.Block != nil && len(b.Block.Nodes) > 0 {
// 		_ = b.Block.Nodes[len(b.Block.Nodes)-1]
// 		// ..
// 	}
// 	return ""
// }
