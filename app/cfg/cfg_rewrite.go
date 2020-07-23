package cfg

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"sourcecrawler/app/helper"
	"strconv"
	"strings"

	"golang.org/x/tools/go/cfg"
)

// ------------------ Logical Methods ------------------

func (paths *PathList) TraverseCFG(curr Wrapper, root Wrapper) []Path {

	paths.TraverseCFGRecur(curr, make(map[string]int), make([]ast.Node, 0), root, make(map[string]ast.Node), make([]ExecutionLabel, 0))
	return paths.Paths
}

// ------------- Traversal function ---------------
// Assumptions: outer wrapper has already been assigned, and tree structure has been created.
func (paths *PathList) TraverseCFGRecur(curr Wrapper, ssaInts map[string]int,
	stmts []ast.Node, root Wrapper, varFilter map[string]ast.Node, pathLabels []ExecutionLabel) {
	//Check if if is a FnWrapper or BlockWrapper Type
	switch currWrapper := curr.(type) {
	case *FnWrapper:
	case *BlockWrapper:
		if len(currWrapper.Succs) == 2 {
			ast.Inspect(currWrapper.Block.Nodes[len(currWrapper.Block.Nodes)-1], func(node ast.Node) bool {
				switch node := node.(type) {
				case *ast.Ident:
					//Grab function name and identifier name
					if fn, ok := currWrapper.GetOuterWrapper().(*FnWrapper); ok {
						var fnName string
						switch fn := fn.Fn.(type) {
						case *ast.FuncDecl:
							fnName = fn.Name.Name
						case *ast.FuncLit:
							//TODO: wat do??
							fnName = "lit"
						}

						//Remove extra fnName.fnName.fnName... (works for now)
						if !strings.Contains(node.Name, fnName+".") {
							node.Name = fmt.Sprint(fnName, ".", node.Name)
						}
					}
				}
				return true
			})
		}

		for _, node := range currWrapper.Block.Nodes {
			good := false
			switch node.(type) {
			case *ast.AssignStmt, *ast.IncDecStmt:
				good = true
			}
			if good {
				ast.Inspect(node, func(node ast.Node) bool {
					switch node := node.(type) {
					case *ast.Ident:
						//Grab function name and identifier name
						if fn, ok := currWrapper.GetOuterWrapper().(*FnWrapper); ok {
							var fnName string
							switch fn := fn.Fn.(type) {
							case *ast.FuncDecl:
								fnName = fn.Name.Name
							case *ast.FuncLit:
								//TODO: wat do??
								fnName = "lit"
							}
							if !strings.Contains(node.Name, fnName+".") {
								node.Name = fmt.Sprint(fnName, ".", node.Name)
							}
						}
					}
					return true
				})
			}
		}

		for _, node := range currWrapper.Block.Nodes {
			//Increment counter for each object encountered
			switch node := node.(type) {
			case *ast.AssignStmt, *ast.IncDecStmt:
				artificial := RessignmentConversion(node)
				if artificial != nil {
					for i, l := range artificial.Lhs {
						if id, ok := l.(*ast.Ident); ok {
							name := id.Name
							negative := true
							if i, ok := ssaInts[name]; ok && i > -1 {
								negative = false
								ssaInts[name]--
								// Delete the map entry since a 0 would get prepended to the ID
								if ssaInts[name] == 0 {
									delete(ssaInts, name)
								}
							}
							SSAconversion(artificial.Rhs[i], ssaInts)
							if !negative {
								ssaInts[name]++
							}
							SSAconversion(l, ssaInts)
							ssaInts[name]++
						}
					}
					stmts = append(stmts, artificial)
					pathLabels = append(pathLabels, currWrapper.GetLabel())
					// if currWrapper.GetLabel() == NoLabel{
					// 	fmt.Println("Current wrapper has no label", currWrapper.Block.String(), currWrapper)
					// }
				}
			case *ast.ExprStmt:
				SSAconversion(node.X, ssaInts)
			case ast.Expr:
				SSAconversion(node, ssaInts)
			}
		}

		//=====Integrate-labeling changes
		// varList := GetVariables(currWrapper, varFilter)
		// vars := []ast.Node{}
		// // Filter out variables already in the array again
		// for _, v := range varList {
		// 	contained := false
		// 	for _, existingVar := range stmts {
		// 		if v == existingVar {
		// 			contained = true
		// 			break
		// 		}
		// 	}
		// 	if !contained {
		// 		vars = append([]ast.Node{v}, vars...)
		// 		if currWrapper.GetLabel() == NoLabel{
		// 			fmt.Println("Current wrapper has no label", currWrapper.Block.String(), currWrapper)
		// 		}
		// 		pathLabels = append(pathLabels, currWrapper.GetLabel())
		// 	}
		// }
		// stmts = append(stmts, vars...)

		//=========== rewrite branch changes ===============
		// varList := GetVariables(currWrapper, varFilter)

		// vars := []ast.Node{}
		// // Filter out variables already in the array again
		// for _, v := range varList {
		// 	contained := false
		// 	for _, existingVar := range stmts {
		// 		if v == existingVar {
		// 			contained = true
		// 			break
		// 		}
		// 	}
		// 	if !contained {
		// 		vars = append([]ast.Node{v}, vars...)

		// 	}
		// }
		// stmts = append(stmts, vars...)

		//If conditional block, extract the condition and add to list
		condition := currWrapper.GetCondition()
		if condition != nil {
			ast.Inspect(condition, func(node ast.Node) bool {
				if node, ok := node.(*ast.Ident); ok {
					SSAconversion(node, ssaInts)
				}
				return true
			})

			//pathLabels[condition] = currWrapper.GetLabel() //add label to conditionals
			//pathLabels = append(pathLabels, currWrapper.GetLabel())

			//Prevent duplicates
			contained := false
			for _, existingCondition := range stmts {
				if condition == existingCondition {
					contained = true
					break
				}
			}
			if !contained {
				stmts = append(stmts, condition)
				pathLabels = append(pathLabels, currWrapper.GetLabel())
			}
		}

	}

	//If there are parent blocks to check, continue | otherwise add the path
	if len(curr.GetParents()) != 0 {
		//Go through each parent in the wrapper
		for _, parent := range curr.GetParents() {
			paths.TraverseCFGRecur(parent, ssaInts, stmts, root, varFilter, pathLabels)
		}
	} else {

		// the filter seems to be working but somehow vars
		// gets 3 of the same thing (since there's 3 functions I guess)
		// fmt.Println("hello", stmts)
		paths.AddNewPath(Path{Expressions: stmts, ExecStatus: pathLabels})
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
func NewFnWrapper(root ast.Node, callingArgs []ast.Expr) *FnWrapper {
	var c *cfg.CFG
	params := make([]*ast.Object, 0)
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
		//gather list of parameters
		// fmt.Println(params)
		for _, param := range fn.Type.Params.List {
			for _, name := range param.Names {
				params = append(params, name.Obj)
			}
		}
	case *ast.FuncLit:
		c = cfg.New(fn.Body, func(call *ast.CallExpr) bool {
			return true
		})
		for _, param := range fn.Type.Params.List {
			for _, name := range param.Names {
				params = append(params, name.Obj)
			}
		}
	}

	paramsToArgs := make(map[*ast.Object]ast.Expr, len(callingArgs))

	//map every parameter to the argument in the calling function
	for i, arg := range callingArgs {
		paramsToArgs[params[i]] = arg
	}
	fn := &FnWrapper{
		Fn:           root,
		Parents:      make([]Wrapper, 0),
		ParamsToArgs: paramsToArgs,
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
//by adding in function calls, should be called
//from the root with an empty stack
func ExpandCFG(w Wrapper) {
	ExpandCFGRecur(w, make([]*FnWrapper, 0))
}

func ExpandCFGRecur(w Wrapper, stack []*FnWrapper) {
	if w != nil {
		switch b := w.(type) {
		case *FnWrapper:
			//if function has not been seen in current
			//branch, expand it, otherwise, skip, because
			//it would recurse infinitely
			found := false
			for _, frame := range stack {
				if frame.Fn == b.Fn {
					found = true
					break
				}
			}
			if !found {
				ExpandCFGRecur(b.FirstBlock, append(stack, b))
			}
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
					var call *ast.CallExpr
					ast.Inspect(node, func(node ast.Node) bool {
						if node, ok := node.(*ast.CallExpr); ok {
							call = node
							return false
						}
						if _, ok := node.(*ast.FuncLit); ok {
							return false
						}
						return true
					})
					// switch node := node.(type) {
					// case *ast.CallExpr:
					// 	call = node
					// case *ast.ExprStmt:
					// 	if x, ok := node.X.(*ast.CallExpr); ok {
					// 		call = x
					// 	}
					// }

					if call != nil {
						//get arguments
						//split the block into two pieces
						topBlock, bottomBlock := b.splitAtNodeIndex(i)

						//TODO:
						newFn := GetDeclarationOfFunction(b.Outer, call, call.Args)

						//get new function wrapper
						if newFn != nil {
							newFn.SetOuterWrapper(b.Outer)

							//connect the topBlock to the function
							topBlock.connectCallTo(newFn)
							//replace block with topBlock
							for _, p := range b.Parents {
								//remove block as child?
								p.RemoveChild(b)
								b.RemoveParent(p)
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
									b.RemoveChild(c)
									c.RemoveParent(b)
									c.AddParent(bottomBlock)
									bottomBlock.AddChild(c)
								}
							} else {
								for _, c := range b.Succs {
									b.RemoveChild(c)
									c.RemoveParent(b)
									newFn.connectReturnsTo(c)
								}
							}
							//stop after first function, block is now
							//obsolete, move on to sucessors of topBlock
							for _, succ := range topBlock.Succs {
								ExpandCFGRecur(succ, stack)
							}
						}

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
				ExpandCFGRecur(c, stack)
			}
		}
	}
}

//TODO: test this because it's a mess and I'm pretty sure it'll break
func GetDeclarationOfFunction(w Wrapper, fn ast.Expr, args []ast.Expr) *FnWrapper {
	//if in map, get declaration
	switch v := fn.(type) {
	case *ast.CallExpr:
		if fnName, ok := v.Fun.(*ast.Ident); ok {
			//this is when it is in the map
			if param, ok := w.(*FnWrapper).ParamsToArgs[fnName.Obj]; ok {
				//if literal
				if fnParam, ok := param.(*ast.FuncLit); ok {
					return NewFnWrapper(fnParam, args)
				} else {
					//identifier
					return GetDeclarationOfFunction(w.GetOuterWrapper(), param, args)
				}
			} else {
				//if not a parameter, find it using blind method
				return w.(*FnWrapper).FirstBlock.(*BlockWrapper).GetFunctionWrapperFor(fn.(*ast.CallExpr), args)
			}
		}
	case *ast.Ident:
		//add case for when the ientifier is nested
		//search the params map again
		if param, ok := w.(*FnWrapper).ParamsToArgs[v.Obj]; ok {
			//if literal
			if fnParam, ok := param.(*ast.FuncLit); ok {
				return NewFnWrapper(fnParam, args)
			} else {
				//identifier
				return GetDeclarationOfFunction(w.GetOuterWrapper(), param, args)
			}
		}
		switch decl := v.Obj.Decl.(type) {
		//local functions (foo := func())
		case *ast.AssignStmt:
			return NewFnWrapper(decl.Rhs[0].(*ast.FuncLit), args)
		//package functions (func foo())
		case *ast.FuncDecl:
			return NewFnWrapper(decl.Body, args)

		}
	}
	return nil
}

//Succs of first block are nil, and Parents of second block are nil, must be added
//the inner cfg.Block variables are default values except Nodes, their use must
//be avoided
func (b *BlockWrapper) splitAtNodeIndex(ndx int) (first, second *BlockWrapper) {
	//TODO: make sure the right slices are taken depending on where the split is;
	// need to look into how the cfg is represented if the function is first or
	// last node
	if len(b.Block.Nodes)-1 > ndx {
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
		if len(b.Block.Nodes)-1 == ndx {
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
			// Doing this instead of append(rets, GetLeafNodes(c)...)
			// fixes an issue with duplicate variables when traversing
			// multiple leaf nodes (however this might be due to the global
			// execution path at the moment. This can be changed back
			// when that is fixed I think)
			for _, leaf := range GetLeafNodes(c) {
				contained := false
				for _, r := range rets {
					if leaf == r {
						contained = true
						break
					}
				}
				if !contained {
					rets = append(rets, leaf)
				}
			}
		} else {
			rets = append(rets, c)
		}
	}
	return rets
}

//must be called on a Wrapper to give access to the ASTs
func (b *BlockWrapper) GetFunctionWrapperFor(node *ast.CallExpr, args []ast.Expr) *FnWrapper {
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
		return NewFnWrapper(fn, args)
	}
	return nil
}

func FindPanicWrapper(w Wrapper, traceStruct *helper.StackTraceStruct) Wrapper {
	if w != nil {
		switch w := w.(type) {
		case *BlockWrapper:
			for _, node := range w.Block.Nodes {
				pos := w.GetFileSet().Position(node.Pos())
				if strings.Contains(pos.Filename, traceStruct.FileName[0]) {
					lineNum, err := strconv.Atoi(traceStruct.LineNum[0])
					if err == nil && pos.Line == lineNum {
						return w
					}
				}
			}
		}
		for _, child := range w.GetChildren() {
			ret := FindPanicWrapper(child, traceStruct)
			if ret != nil {
				return ret
			}
		}
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
			var bf bytes.Buffer
			_ = printer.Fprint(&bf, w.GetFileSet(), node)
			fmt.Println(level, bf.String())
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
