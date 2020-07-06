package cfg

import (
	"go/ast"

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
	Parent     []Wrapper
	Outer      Wrapper
	// ...?
}

// func NewCfgWrapper(first *cfg.Block) *CfgWrapper {
// 	return &CfgWrapper{
// 		FirstBlock: NewBlockWrapper(first, nil),
// 	}
// }

type BlockWrapper struct {
	Block   *cfg.Block
	Parents []Wrapper
	Succs   []Wrapper
	Outer   Wrapper
	// ...
	// method to get condition (can return nil if not conditional)
}

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

func (fn *FnWrapper) expandCFG() {
	block := fn.FirstBlock
	for block != nil {
		switch block := block.(type) {
		case *BlockWrapper:
			//check if the next block is a FnWrapper
			// this means it is already conencted
			shouldConnect := true
			for _, succ := range block.Succs {
				if _, ok := succ.(*FnWrapper); ok {
					shouldConnect = false
					break
				}
			}

			if shouldConnect {

			}
			//For every node in the block
			cfgBlock := block.Block
			for _, node := range cfgBlock.Nodes {
				//check if the node is a callExpr
			}


		}
	}
	//
}

func (b *BlockWrapper) connectToFunctionBlock() {

}
