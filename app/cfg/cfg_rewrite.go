package cfg

import (
	"fmt"
	"go/ast"
	"sourcecrawler/app/logsource"
	"strings"

	"golang.org/x/tools/go/cfg"
)

type Path struct {
	Stmts []string
	//Something to keep track of vars?
}

type Wrapper interface {
	AddParent(w Wrapper)
	GetParents() []Wrapper
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
}

type BlockWrapper struct {
	Block   *cfg.Block
	Parents []Wrapper
	Succs   []Wrapper
	Outer   Wrapper
}

// ------------------ FnWrapper ----------------------

func (fn *FnWrapper) AddParent(w Wrapper) {
	if w != nil {
		fn.Parents = append(fn.Parents, w)
	}
}

func (fn *FnWrapper) GetParents() []Wrapper {
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

func (b *BlockWrapper) GetParents() []Wrapper {
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
	//Conditional block (TODO: assuming the last node in the block node list is the conditional node)
	if len(b.Succs) == 2 && b.Block != nil && len(b.Block.Nodes) > 0{
		condNode := b.Block.Nodes[len(b.Block.Nodes)-1] //Last node in the block's list of nodes
		ast.Inspect(condNode, func(currNode ast.Node) bool {
			if ifNode, ok := condNode.(*ast.IfStmt); ok{
				condition = fmt.Sprint(ifNode.Cond)
			}

			if forNode, ok := condNode.(*ast.ForStmt); ok{
				fmt.Println("For loop condition", forNode.Cond)
			}
			return true
		})
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
// NewFnWrapper creates a wrapper around the `*cfg.CFG` for
// a given function.
func NewFnWrapper(root ast.Node) *FnWrapper {
	var c *cfg.CFG
	switch fn := root.(type) {
	case *ast.FuncDecl:
		c = cfg.New(fn.Body, func(call *ast.CallExpr) bool {
			if call != nil {
				// Functions that won't potentially cause the program will return.
				if fn.Name.Name != "Exit" && !strings.Contains(fn.Name.Name, "Fatal") && fn.Name.Name != "panic" {
					return true
				}
			}
			return false
		})
	case *ast.FuncLit:
		c = cfg.New(fn.Body, func(call *ast.CallExpr) bool {
			return true
		})
	}

	fn := &FnWrapper{
		Fn:      root,
		Parents: make([]Wrapper, 0),
	}

	if c != nil && len(c.Blocks) > 0 {
		fn.FirstBlock = NewBlockWrapper(c.Blocks[0], fn)

	}

	return fn
}

// NewBlockWrapper creates a wrapper around a `*cfg.Block` which points to
// the outer `Wrapper`
func NewBlockWrapper(block *cfg.Block, outer Wrapper) *BlockWrapper {
	return newBlockWrapper(block, nil, outer, make(map[*cfg.Block]*BlockWrapper))
}

func newBlockWrapper(block *cfg.Block, parent *BlockWrapper, outer Wrapper, cache map[*cfg.Block]*BlockWrapper) *BlockWrapper {
	if b, ok := cache[block]; ok {
		b.AddParent(parent)
		return b
	}

	b := &BlockWrapper{
		Block:   block,
		Parents: []Wrapper{parent},
		Succs:   make([]Wrapper, 0),
		Outer:   outer,
	}

	if !strings.Contains(block.String(), "for.post") && !strings.Contains(block.String(), "range.body") {
		for _, succ := range block.Succs {
			var block *BlockWrapper
			if cachedBlock, ok := cache[succ]; ok {
				block = cachedBlock
			} else {
				block = newBlockWrapper(succ, b, outer, cache)
				cache[succ] = block
			}
			b.Succs = append(b.Succs, block)
		}
	}

	return b
}

// GetCondition returns the condition node inside of the
// contained `cfg.Block` given that it is a conditional.
//func (b *BlockWrapper) GetCondition() ast.Node {
//	if len(b.Succs) == 2 && b.Block != nil && len(b.Block.Nodes) > 0 {
//		return b.Block.Nodes[len(b.Block.Nodes)-1]
//	}
//	return nil
//}

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

func DebugPrint(w Wrapper, level string) {

	// fmt.Printf("%schildren:%v parents:%v outer:%v", level, w.GetChildren(), w.GetParents(), w.GetOuterWrapper())
	switch w := w.(type) {
	case *BlockWrapper:
		if w == nil {
			return
		}
		fmt.Println(level, "meta: block:", w.Block, "succs:", w.Succs, "outer:", w.Outer, "parents:", w.Parents)
		if w.Block == nil {
			break
		}
		for _, node := range w.Block.Nodes {
			fmt.Println(level, node)
		}
	case *FnWrapper:
		if w == nil {
			return
		}
		fmt.Println(level, "meta: fn:", w.Fn, "outer:", w.Outer, "parents:", w.Parents)
	}
	for _, s := range w.GetChildren() {
		DebugPrint(s, level+"  ")
	}
}
