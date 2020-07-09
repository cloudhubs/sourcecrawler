package cfg

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"sourcecrawler/app/helper"
	"strings"

	"golang.org/x/tools/go/cfg"
)

// ---- Represents the execution path --------
type Path struct {
	Stmts []string
	Variables []ast.Node //*ast.AssignStmt or *ast.ValueSpec
}
var executionPath []Path = []Path{}
func GetExecPath() []Path{
	return executionPath
}

//---- Branch Labels ----------
type ExecutionLabel int
const (
	NoLabel ExecutionLabel = iota
	Must
	May
	MustNot
)

func (s ExecutionLabel) String() string {
	return [...]string{"NoLabel", "Must", "May", "MustNot"}[s]
}

type Wrapper interface {
	AddParent(w Wrapper)
	RemoveParent(w Wrapper)
	GetParents() []Wrapper
	AddChild(w Wrapper)
	RemoveChild(w Wrapper)
	GetChildren() []Wrapper
	GetOuterWrapper() Wrapper
	SetOuterWrapper(w Wrapper)
	GetLabel() ExecutionLabel
	SetLabel(label ExecutionLabel)

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
	//Vars []VariableWrapper
	//Vars map[*ast.Ident][]*ast.Ident // map of identifiers to identifiers
	Label	ExecutionLabel

}

type BlockWrapper struct {
	Block   *cfg.Block
	Parents []Wrapper
	Succs   []Wrapper
	Outer   Wrapper
	Label	ExecutionLabel
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
			if i < len(fn.Parents)-1 {
				fn.Parents = append(fn.Parents[:i], fn.Parents[i+1:]...)
			} else {
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

func (fn *FnWrapper) GetLabel() ExecutionLabel{
	return fn.Label
}

func (fn *FnWrapper) SetLabel(label ExecutionLabel){
	fn.Label = label
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
			if i < len(b.Parents)-1 {
				b.Parents = append(b.Parents[:i], b.Parents[i+1:]...)
			} else {
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
			if i < len(b.Succs)-1 {
				b.Succs = append(b.Succs[:i], b.Succs[i+1:]...)
			} else {
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

func (b *BlockWrapper) GetLabel() ExecutionLabel{
	return b.Label
}

func (b *BlockWrapper) SetLabel(label ExecutionLabel){
	b.Label = label
}

// ------------------ Logical Methods ------------------

// ---- Traversal function ---------------
// curr -> starting block | condStmts -> holds conditional expressions | root -> outermost wrapper
// vars -> holds list of variables on path
// Assumptions: outer wrapper has already been assigned, and tree structure has been created.
func TraverseCFG(curr Wrapper, condStmts []string, vars []ast.Node, root Wrapper){

	//Check if if is a FnWrapper or BlockWrapper Type
	switch currWrapper := curr.(type){
	case *FnWrapper:
		fmt.Println("FnWrapper", currWrapper)
		fnName, funcVars := GetFuncInfo(currWrapper.Fn) //Gets the function name and a list of variables
		fmt.Printf("Function name (%s), (%v)\n", fnName, funcVars)

	case *BlockWrapper:
		fmt.Println("BlockWrapper", currWrapper)

		//If conditional block, extract the condition and add to list
		condition := currWrapper.GetCondition()
		if condition != ""{
			condStmts = append(condStmts, condition)
		}

		//Gets a list of all variables inside the block, and add
		// -Filter out relevant variables
		varList := GetVariables(currWrapper.Block.Nodes)
		if len(varList) != 0{
			vars = append(vars, varList...)
		}

	}

	//If there are parent blocks to check, continue | otherwise add the path
	if len(curr.GetParents()) != 0{
		//Go through each parent in the wrapper
		for _, parent := range curr.GetParents(){
			TraverseCFG(parent, condStmts, vars, root)
		}
	}else{
		executionPath = append(executionPath, Path{Stmts: condStmts, Variables: vars}) //If at root node, then add path
	}
}

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
		node, err := parser.ParseFile(ret.Fset, file, nil, parser.ParseComments)
		if err != nil {
			panic(err)
		}
		ret.ASTs = append(ret.ASTs, node)
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
		fn.FirstBlock = NewBlockWrapper(c.Blocks[0], fn, fn)
	}

	return fn
}

// NewBlockWrapper creates a wrapper around a `*cfg.Block` which points to
// the outer `Wrapper`
func NewBlockWrapper(block *cfg.Block, parent Wrapper, outer Wrapper) *BlockWrapper {
	return newBlockWrapper(block, parent, outer, make(map[*cfg.Block]*BlockWrapper))
}

func newBlockWrapper(block *cfg.Block, parent Wrapper, outer Wrapper, cache map[*cfg.Block]*BlockWrapper) *BlockWrapper {
	//Avoid duplicate blocks
	if b, ok := cache[block]; ok {
		b.AddParent(parent)
		return b
	}

	b := &BlockWrapper{
		Block:   block,
		Succs:   make([]Wrapper, 0),
		Outer:   outer,
		Parents: make([]Wrapper, 0),
	}

	if parent != nil {
		b.AddParent(parent)
	}

	// Don't recurse on these otherwise this will infinitely loop
	if !strings.Contains(block.String(), "for.post") && !strings.Contains(block.String(), "range.body") {
		for _, succ := range block.Succs {
			succBlock := newBlockWrapper(succ, b, outer, cache)
			cache[succ] = succBlock
			b.AddChild(succBlock)
		}

		// Handle loops by manually grabbing the cached for.post or range.body
		if strings.Contains(b.Block.String(), "range.loop") {
			if body, ok := cache[block.Succs[0]]; ok {
				body.AddChild(b)
				b.AddParent(body)
			}
		} else if strings.Contains(b.Block.String(), "for.loop") {
			if post, ok := cache[block.Succs[0].Succs[0]]; ok {
				post.AddChild(b)
				b.AddParent(post)
			}
		}
	}

	return b
}

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
			//TODO: confirm conditionals will not
			// have FnWrapper as immediate successor
			shouldConnect := true
			for _, succ := range b.Succs {
				if _, ok := succ.(*FnWrapper); ok {
					shouldConnect = false
					break
				}
			}

			if shouldConnect {
				//For every node in the block
				for i, node := range b.Block.Nodes {
					//check if the node is a callExpr
					if node, ok := node.(*ast.CallExpr); ok {
						//split the block into two pieces
						topBlock, bottomBlock := b.splitAtNodeIndex(i)

						//get new function wrapper
						newFn := b.getFunctionWrapperFor(node)

						//connect the topBlock to the function
						topBlock.connectCallTo(newFn)
						//replace block with topBlock
						for _, p := range b.Parents {
							//remove block as child?
							p.RemoveChild(b)
							p.AddChild(topBlock)
							topBlock.AddParent(p)
						}


						if bottomBlock != nil {
							//connect the function to the
							//second half of the block
							newFn.connectReturnsTo(bottomBlock)
							//replace block with bottomBlock
							for _, c := range b.Succs {
								//remove block as parent?
								c.RemoveParent(b)
								c.AddParent(bottomBlock)
								bottomBlock.AddChild(c)
							}
						} else {
							//no bottomBlock, connect returns directly to
							//successors
							for _, c := range b.Succs {
								newFn.connectReturnsTo(c)
							}
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
			//did not find any function calls in this block,
			//or it was already connected to one, move on to
			//successor

			//NOTE: this will still run after shouldContinue
			// block has removed b from the graph, but
			// it should be harmless since it has no connections
			// and the recursion will still expand its successors
			for _, c := range b.Succs {
				expandCFG(c)
			}
		}
	}
}

//Succs of first block are nil, and Parents of second block are nil, must be added
//the inner cfg.Block variables are default values except Nodes, their use must
//be avoided
func (b *BlockWrapper) splitAtNodeIndex(ndx int) (first, second *BlockWrapper) {
	//TODO: make sure the right slices are taken depending on where the split is;
	// need to look into how the cfg is represented if the function is first or
	// last node
	if len(b.Block.Nodes) - 1 > ndx {
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
				Block: &cfg.Block{
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
		if len(b.Block.Nodes) - 1 == ndx {
			//there is no split, just copy nodes to first block
			return &BlockWrapper{
					Block: &cfg.Block{
						Nodes: b.Block.Nodes,
						Succs: nil,
						Index: 0,
						Live:  false,
					},
					Parents: b.Parents,
					Succs:   nil,
					Outer:   b.Outer,
				}, nil
		}
		//index out-of-bounds
		return nil, nil
	}
}

func (b *BlockWrapper) connectCallTo(fn *FnWrapper) {
	b.AddChild(fn)
	fn.AddParent(b)

}

func (fn *FnWrapper) connectReturnsTo(w Wrapper) {
	for _, leaf := range GetLeafNodes(fn) {
		leaf.AddChild(w)
		w.AddParent(leaf)
	}
}

//should be called on FnWrapper, but recursion
//requires interface
func GetLeafNodes(w Wrapper) []Wrapper {
	var rets []Wrapper
	for _, c := range w.GetChildren() {
		if len(c.GetChildren()) > 0 {
			rets = append(rets, GetLeafNodes(c)...)
		} else {
			rets = append(rets, c)
		}
	}
	return rets
}

//must be called on a Wrapper to give access to the ASTs
func (b *BlockWrapper) getFunctionWrapperFor(node *ast.CallExpr) *FnWrapper {
	var fn *ast.FuncDecl
	//loop through every AST file
	for _, file := range b.GetASTs() {
		stop := false
		ast.Inspect(file, func(n ast.Node) bool {
			if n, ok := n.(*ast.FuncDecl); ok {
				//TODO: double-check this is the proper conversion
				if ident, ok := node.Fun.(*ast.Ident); ok {
					if strings.EqualFold(ident.Name, n.Name.Name) {
						fn = n
						//stop when you find it
						stop = true
						return false
					}
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

func DebugPrint(w Wrapper, level string, printed map[Wrapper]struct{}) {
	printWrapperList := func(w []Wrapper) {
		for _, p := range w {
			switch p := p.(type) {
			case *BlockWrapper:
				fmt.Print(p.Block.String(), ", ")
			case *FnWrapper:
				switch fn := p.Fn.(type) {
				case *ast.FuncDecl:
					fmt.Print(fn.Name.Name, ", ")
				case *ast.FuncLit:
					fmt.Print(fn.Type, ", ")
				}
			}
		}
	}
	// fmt.Printf("%schildren:%v parents:%v outer:%v", level, w.GetChildren(), w.GetParents(), w.GetOuterWrapper())
	switch w := w.(type) {
	case *BlockWrapper:
		if w == nil {
			return
		}
		fmt.Print(level, "meta: block: ", w.Block, " outer: ", w.Outer, " succs: ")
		printWrapperList(w.GetChildren())
		fmt.Print(" parents: ")
		printWrapperList(w.GetParents())
		fmt.Println()
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
		fmt.Print(level, "meta: fn: ", w.Fn, " outer: ", w.Outer, " parents: ")
		printWrapperList(w.GetParents())
		fmt.Println()
	}
	printed[w] = struct{}{}
	for _, s := range w.GetChildren() {
		if _, ok := printed[s]; !ok {
			printed[s] = struct{}{}
			DebugPrint(s, level+"  ", printed)
		} else {
			return
		}
	}
}
