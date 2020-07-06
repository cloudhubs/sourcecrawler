package cfg

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/cfg"
)

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

	for _, succ := range block.Succs {
		var block *BlockWrapper
		if cachedBlock, ok := cache[succ]; ok {
			block = cachedBlock
		} else if !strings.Contains(succ.String(), "for") {
			block = newBlockWrapper(succ, b, outer, cache)
		}
		b.Succs = append(b.Succs, block)
	}

	return b
}

// GetCondition returns the condition node inside of the
// contained `cfg.Block` given that it is a conditional.
func (b *BlockWrapper) GetCondition() ast.Node {
	if len(b.Succs) == 2 && b.Block != nil && len(b.Block.Nodes) > 0 {
		return b.Block.Nodes[len(b.Block.Nodes)-1]
	}
	return nil
}

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
