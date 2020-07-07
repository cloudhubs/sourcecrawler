package cfg

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sourcecrawler/app/helper"
	"strings"

	"golang.org/x/tools/go/cfg"
)

type Wrapper interface {
	AddParent(w Wrapper)
	RemoveParent(w Wrapper)
	GetParents() []Wrapper
	AddChild(w Wrapper)
	RemoveChild(w Wrapper)
	GetChildren() []Wrapper
	GetOuterWrapper() Wrapper
	SetOuterWrapper(w Wrapper)

	GetFileSet() *token.FileSet
	GetASTs() []*ast.File
}

type FnWrapper struct {
	Fn         ast.Node // *ast.FuncDel or *ast.FuncLit
	FirstBlock Wrapper
	Parents    []Wrapper
	Outer      Wrapper
	// ...?
	Fset *token.FileSet
	ASTs []*ast.File
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

func (fn *FnWrapper) RemoveParent(w Wrapper) {
	for i, p := range fn.Parents {
		if p == w {
			if i < len(fn.Parents) - 1 {
				fn.Parents = append(fn.Parents[:i], fn.Parents[i+1:]...)
			}else {
				fn.Parents = fn.Parents[:i]
			}
		}
	}
}


func (fn *FnWrapper) GetParents() []Wrapper {
	return fn.Parents
}

func (fn *FnWrapper) AddChild(w Wrapper) {
	fn.FirstBlock = w
}

func (fn *FnWrapper) RemoveChild(w Wrapper) {
	fn.FirstBlock = nil
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

//must always be defined by the outermost wrapper
func (fn *FnWrapper) GetFileSet() *token.FileSet {
	if fn.Fset != nil {
		return fn.Fset
	} else {
		if fn.Outer != nil {
			return fn.Outer.GetFileSet()
		}
		return nil
	}
}

//must always be defined by the outermost wrapper
func (fn *FnWrapper) GetASTs() []*ast.File {
	if fn.ASTs != nil {
		return fn.ASTs
	} else {
		if fn.Outer != nil {
			return fn.Outer.GetASTs()
		}
		return []*ast.File{}
	}
}

// ------------------ BlockWrapper ----------------------

func (b *BlockWrapper) AddParent(w Wrapper) {
	if w != nil {
		b.Parents = append(b.Parents, w)
	}
}

func (b *BlockWrapper) RemoveParent(w Wrapper) {
	for i, p := range b.Parents {
		if p == w {
			if i < len(b.Parents) - 1 {
				b.Parents = append(b.Parents[:i], b.Parents[i+1:]...)
			}else {
				b.Parents = b.Parents[:i]
			}
		}
	}
}


func (b *BlockWrapper) GetParents() []Wrapper {
	return b.Parents
}

func (b *BlockWrapper) RemoveChild(w Wrapper) {
	for i, c := range b.Succs {
		if c == w {
			if i < len(b.Succs) - 1 {
				b.Succs = append(b.Succs[:i], b.Succs[i+1:]...)
			}else {
				b.Succs = b.Succs[:i]
			}
		}
	}
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

func (b *BlockWrapper) GetFileSet() *token.FileSet {
	if b.Outer != nil {
		return b.Outer.GetFileSet()
	}
	return nil
}

func (b *BlockWrapper) GetASTs() []*ast.File {
	if b.Outer != nil {
		return b.Outer.GetASTs()
	}
	return []*ast.File{}
}

// ------------------ Logical Methods ------------------

//The value returned should be the topmost wrapper
//of the CFG, the entry point of the program should
//be wrapped in this object
func SetupPersistentData(base string) *FnWrapper {
	//declare persisting object
	ret := &FnWrapper{
		Fn:         nil,
		FirstBlock: nil,
		Parents:    make([]Wrapper, 0),
		Outer:      nil,
		Fset:       token.NewFileSet(),
		ASTs:       make([]*ast.File, 0),
	}

	//gather files
	filesToParse := helper.GatherGoFiles(base)
	for _, file := range filesToParse {
		//parse files into fileset and ast slice
		node, err :=  parser.ParseFile(ret.Fset,file,nil,parser.ParseComments)
		if err != nil {
			panic(err)
		}
		ret.ASTs = append(ret.ASTs,node)
	}

	//the persistent data should be available to any of
	//this object's GetChildren()
	return ret
}


// NewFnWrapper creates a wrapper around the `*cfg.CFG` for
// a given function.
//TODO: how to identify FuncLit calls and connect them
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

// func (b *BlockWrapper) getCondition() string {
// 	if len(b.Succs) == 2 && b.Block != nil && len(b.Block.Nodes) > 0 {
// 		_ = b.Block.Nodes[len(b.Block.Nodes)-1]
// 		// ..
// 	}
// 	return ""
// }

//goal is to continuously build the CFG
//by adding in function calls
func expandCFG(w Wrapper) {
	if w != nil {
		switch b := w.(type) {
		case *FnWrapper:
			//this function has already
			//been added, go deeper
			expandCFG(b.FirstBlock)
		case *BlockWrapper:

			//check if the next block is a FnWrapper
			// this means it is already connected
			shouldConnect := true
			for _, succ := range b.Succs {
				if _, ok := succ.(*FnWrapper); ok {
					shouldConnect = false
					break
				}
			}

			if shouldConnect {
				//For every node in the block
				cfgBlock := b.Block
				for i, node := range cfgBlock.Nodes {
					//check if the node is a callExpr
					if node, ok := node.(*ast.CallExpr); ok {
						//split the block into two pieces
						topBlock, bottomBlock := b.splitAtNodeIndex(i)

						//get new function wrapper
						newFn := b.getFunctionWrapperFor(node)

						//TODO: double-check all the connections
						//connect the topBlock to the function
						topBlock.connectCallTo(newFn)

						//connect the function to the
						//second half of the block
						newFn.connectReturnsTo(bottomBlock)

						//replace block with topBlock
						for _, p := range b.Parents {
							//remove block as child?
							p.RemoveChild(b)
							p.AddChild(topBlock)
							topBlock.AddParent(p)
						}

						//replace block with bottomBlock
						for _, c := range b.Succs {
							//remove block as parent?
							c.AddParent(bottomBlock)
							bottomBlock.AddChild(c)
						}

						//stop after first function, block is now
						//obsolete, move on to sucessors of topBlock
						//(the replacement)
						for _, succ := range topBlock.Succs {
							expandCFG(succ)
						}
						break
					}
				}
			}
		}
	}
	//
}

//Succs of first block are nil, and Parents of second block are nil, must be added
//the inner cfg.Block variables are default values except Nodes, their use must
//be avoided
func (b *BlockWrapper) splitAtNodeIndex(ndx int) (first, second *BlockWrapper) {
	//TODO: make sure the right slices are taken depending on where the split is;
	// need to look into how the cfg is represented if the function is first or
	// last node
	if len(b.Block.Nodes) > ndx {
		return &BlockWrapper{
				Block: &cfg.Block{
					Nodes: b.Block.Nodes[:ndx+1],
					Succs: nil,
					Index: 0,
					Live:  false,
				},
				Parents: b.Parents,
				Succs:   nil,
				Outer:   b.Outer,
			}, &BlockWrapper{
				Block:   &cfg.Block{
					Nodes: b.Block.Nodes[ndx+1:],
					Succs: nil,
					Index: 0,
					Live:  false,
				},
				Parents: nil,
				Succs:   b.Succs,
				Outer:   b.Outer,
			}
	} else {
		if len(b.Block.Nodes) == ndx {
			return &BlockWrapper{
					Block: &cfg.Block{
						Nodes: b.Block.Nodes[:ndx],
						Succs: nil,
						Index: 0,
						Live:  false,
					},
					Parents: b.Parents,
					Succs:   nil,
					Outer:   b.Outer,
				}, &BlockWrapper{
					Block:   &cfg.Block{
						Nodes: nil,
						Succs: nil,
						Index: 0,
						Live:  false,
					},
					Parents: nil,
					Succs:   b.Succs,
					Outer:   b.Outer,
				}
		}
		return nil, nil
	}
}

func (b *BlockWrapper) connectCallTo(fn *FnWrapper) {
	b.AddChild(fn)
	fn.AddParent(b)

}

func (fn *FnWrapper) connectReturnsTo(w Wrapper) {
	for _, leaf := range getLeafNodes(fn) {
		leaf.AddChild(w)
		w.AddParent(leaf)
	}
}

//should be called on FnWrapper, but recursion
//requires interface
func getLeafNodes(w Wrapper) []Wrapper{
	var rets []Wrapper
	for _, c := range w.GetChildren(){
		if len(c.GetChildren()) > 0 {
			rets = append(rets, getLeafNodes(c)...)
		}else {
			rets = append(rets, c)
		}
	}
	return rets
}

func (b *BlockWrapper) getFunctionWrapperFor(node *ast.CallExpr) *FnWrapper{
	var fn *ast.FuncDecl
	//loop through
	for _, file := range b.GetASTs() {
		stop := false
		ast.Inspect(file, func(n ast.Node) bool {
			if n, ok := n.(*ast.FuncDecl); ok {
				//TODO: get function name from node.Fun
				if strings.EqualFold("GET NAME", n.Name.Name) {
					fn = n
					//stop when you find it
					stop = true
					return false
				}
			}
			//if you don't find it, keep looking
			return true
		})
		//if found, stop, else search next file
		if stop {
			break
		}
	}

	if fn != nil {
		return NewFnWrapper(fn)
	}
	return nil
}

//TODO: start construction of CFG from the entry point (main or similar)
// with that Wrapper being wrapped in a top-level FnWrapper generated from
// SetupPersistentData.  THe rest should work itself out automatically