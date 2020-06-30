package cfg

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sourcecrawler/app/db"
	"sourcecrawler/app/logsource"
	"strings"

	"github.com/rs/zerolog/log"
	"golang.org/x/tools/go/cfg"
)

// FnCfgCreator allows you to compute the CFG for a given function declaration
type FnCfgCreator struct {
	blocks map[*cfg.Block]db.Node
}

// NewFnCfgCreator returns a newly initialized FnCfgCreator
func NewFnCfgCreator() *FnCfgCreator {
	return &FnCfgCreator{
		blocks: make(map[*cfg.Block]db.Node),
	}
}

// CreateCfg creates the CFG For a given function declaration, filepath and file, and a map of regexes contained within the file.
func (fnCfg *FnCfgCreator) CreateCfg(fn *ast.FuncDecl, base string, fset *token.FileSet, regexes map[int]string) db.Node {
	if fn == nil {
		log.Warn().Msg("received a null function declaration")
		return nil
	} else if fn.Body == nil {
		log.Warn().Msg("received a null function body")
		return nil
	} else if fn.Name == nil {
		log.Warn().Msg("received function with no identifier")
		return nil
	}

	//Create a new map of blocks to nodes (not sure if NewFnCfgCreator replaces this?)
	fnCfg.blocks = make(map[*cfg.Block]db.Node)

	// Function declaration is the root node
	root := getStatementNode(fn, base, fset, regexes)

	//Create new CFG, make sure it is not an exit/fatal/panic statement
	cfg := cfg.New(fn.Body, func(call *ast.CallExpr) bool {
		if call != nil {
			// Functions that won't potentially cause the program will return.
			if fn.Name.Name != "Exit" && !strings.Contains(fn.Name.Name, "Fatal") && fn.Name.Name != "panic" {
				return true
			}
		}
		return false
	})

	fmt.Println("Func", fn.Name.Name)
	for _, block := range cfg.Blocks{
		fmt.Println(block)
	}

	// fmt.Println(fn.Name.Name)
	// fmt.Println(cfg.Format(fset))

	// Empty function declaration
	if len(cfg.Blocks) < 1 {
		return root
	}

	// Begin constructing the cfg
	block := cfg.Blocks[0]
	node := fnCfg.constructSubCfg(block, base, fset, regexes)
	if node == nil {
		return root
	}

	// Connect the function declaration to the sub cfg
	if fn, ok := root.(*db.FunctionDeclNode); ok {
		fn.Child = node
		//fn.Parent = root //Set parent to be root (the func declaration)
		if fn.Child != nil {
			fn.Child.SetParents(fn)
		}
	}

	return root
}


//from the name of a function and its file, create the cfg for it
//this will be useful for connecting external functions after an
//initial cfg is created
func (fnCfg *FnCfgCreator) CreateCfgFromFunctionName(fnName, base string, files []string, seenFn []*db.FunctionNode) db.Node{
	fset := token.NewFileSet()
	found := false
	var fn *ast.FuncDecl
	for _, file := range files {
		node, err := parser.ParseFile(fset,file, nil, parser.ParseComments)
		if err != nil {
			panic(err)
		}
		ast.Inspect(node, func(n ast.Node) bool {
			if n, ok := n.(*ast.FuncDecl); ok {
				if strings.Contains(fnName, n.Name.Name){
					fn = n
					//stop when you find it
					found = true
					return false
				}
			}
			//if you don't find it, keep looking
			return true
		})
		if found {
			break
		}
	}
	if found {
		node := fnCfg.CreateCfg(fn,base,fset, map[int]string{})
		//add in functions in this cfg, excluding
		//any functions already seen in this scope
		//or higher scopes
		ConnectExternalFunctions(node,seenFn, files, base)
		return node
	}
	return nil
}

//initially called with:
//root = first iteration of cfg containing stacktrace functions
//seenFns = []*db.FunctionNode{} (empty)
//base = project root (needed for cfg construction)
//regexes = NOT NEEDED/DEPRICATED ARRAY REPLACED BY INLINE FUNCTION TO GRAB REGEX
func ConnectExternalFunctions(root db.Node, seenFns []*db.FunctionNode, sourceFiles []string, base string){
	var fnCfg FnCfgCreator
	node := root
	for node != nil {
		var tmp db.Node
		//traverse down tree until
		//encountering a function node
		//that is not followed by a
		//function declaration node
		//(which means it is already connected)
		if node, ok := node.(*db.FunctionNode); ok{
			seen := false
			for _, fn := range seenFns {
				//skip functions we have already added on this recursion path
				if strings.Contains(node.FunctionName, fn.FunctionName) && strings.Contains(node.Filename, fn.Filename) {
					seen = true
					break
				}
			}
			if !seen {
				//check if child is a declaration
				if _, ok2 := node.Child.(*db.FunctionDeclNode); !ok2 {
					//add this function to a list so it doesn't recurse on itself
					//keep track of newly added functions
					//within this scope so they can be
					//removed
					//if not a function declaration, add the cfg, recursively
					newFn := fnCfg.CreateCfgFromFunctionName(node.FunctionName, base, sourceFiles, append(seenFns, node))
					if newFn != nil {
						tmp = node.Child
						leafs := getLeafNodes(newFn)
						for _, leaf := range leafs {
							leaf.SetChild([]db.Node{node.Child})
						}
						node.Child = newFn
					}
				}
			}
		}
		//this wouldn't allow for any function to be called more than once,
		//so switching to a recursive call that only cares about
		//next node please
		if tmp != nil {
			node = tmp
		}else{
			for child := range node.GetChildren(){
				//repeat
				ConnectExternalFunctions(child, seenFns, sourceFiles, base)
			}
			node = nil
		}
	}
}

// Prints out the contents of the CFG (recursively)
func PrintCfg(node db.Node, level string) {
	if node == nil {
		return
	}
	var parStr string = "Parent: "
	if node.GetParents() != nil{
		//parStr += node.GetProperties()
		parStr = node.GetFilename()
	}

	switch node := node.(type) {
	case *db.FunctionDeclNode:
		fmt.Printf("%s(%v) %s(%v) (%v) (%v) (%v)\n", level,
			node.Receivers, node.FunctionName, node.Params, node.Returns, node.Label, parStr)
		PrintCfg(node.Child, level+"  ")
	case *db.FunctionNode:
		fmt.Printf("%s%s (%v) (%v) \n", level, node.FunctionName, node.Label, parStr)
		PrintCfg(node.Child, level)
	case *db.StatementNode:
		fmt.Printf("%s%s (%v) (%v)\n", level, node.LogRegex, node.Label, parStr)
		PrintCfg(node.Child, level)
	case *db.ConditionalNode:
		fmt.Printf("%sif %s (%v) (%v)\n", level, node.Condition, node.Label, parStr)
		PrintCfg(node.TrueChild, level+"  ")
		fmt.Println(level + "else")
		PrintCfg(node.FalseChild, level+"  ")
	case *db.ReturnNode:
		fmt.Printf("%sreturn %s (%v) (%v)\n", level, node.Expression, node.Label, parStr)
		lv := ""
		for i := 0; i < len(level)-2; i++ {
			lv += " "
		}
		PrintCfg(node.Child, lv)
	case *db.EndConditionalNode:
		fmt.Printf("%sendIf (%v)\n", level, node.Label)
		PrintCfg(node.Child, level)
		//case *db.VariableNode:
		//	fmt.Printf(node.GetProperties())
		//	PrintCfg(node.Child, level)
	}
}

func (fnCfg *FnCfgCreator) constructSubCfg(block *cfg.Block, base string, fset *token.FileSet, regexes map[int]string) (root db.Node) {
	if block == nil {
		return nil
	}

	conditional := false
	var prev db.Node
	var current db.Node
	// fmt.Println(block.Succs)

	// Add an endIf node as the root if the block is if.done or for.done
	if strings.Contains(block.String(), "if.done") || strings.Contains(block.String(), "for.done") {
		if endIf, ok := fnCfg.blocks[block]; ok {
			current = endIf
		} else {
			current = &db.EndConditionalNode{}
			fnCfg.blocks[block] = current
		}
		prev = current
		root = current
	}

	//TODO: Add a variable node if the function returns some variable that is uses


	// Convert each node in the block into a db.Node (if it is one we want to keep)
	for i, node := range block.Nodes {
		last := i == len(block.Nodes)-1
		conditional = len(block.Succs) > 1 // 2 successors for conditional block

		//Process node based on its type
		switch node := node.(type) {
		case ast.Stmt:
			current = getStatementNode(node, base, fset, regexes)
		case ast.Expr:
			current = getExprNode(node, base, fset, last && conditional, regexes)
		}
		// Received a nil node, continue to the next one
		if current == nil {
			continue
		}

		// Update predecessor pointers
		if root == nil {
			root = current
		}
		if prev == nil {
			prev = current
		}

		switch prevNode := prev.(type) {
		case *db.FunctionNode:
			// Set the previous pointer's child
			if prev != current {
				// fmt.Println("prev not current, set child")
				prevNode.Child = current
				//prevNode.Parent = prev //Set parent node as previous if not same as current
				if prevNode.Child != nil {
					prevNode.Child.SetParents(prevNode)
				}
			}

			// May need to fast-forward to deepest child node here
			// if there was a statement like _, _ = func1(), func2()
			if call, ok := current.(*db.FunctionNode); ok {
				for call != nil {
					if child, ok := call.Child.(*db.FunctionNode); ok && child != nil {
						prev = call
						call = child
						current = child
					} else {
						call = nil
					}
				}
			}
		case *db.StatementNode:
			// You should never encounter a "previous" conditional inside of a block since
			// the conditional is always the last node in a CFG block if a conditional is present
			prevNode.Child = current
			if prevNode.Child != nil {
				prevNode.Child.SetParents(prevNode)
			}
		case *db.EndConditionalNode:
			prevNode.Child = current
			//prevNode.Parent = prev
			if prevNode.Child != nil {
				prevNode.Child.SetParents(prevNode)
			}
			//Add case for variable node

		}
		prev = current

		// Conditionals are the last node and expression in a block, so if it is a control-flow, handle it
		if expr, ok := node.(ast.Expr); ok && last && conditional {
			// If the current node is the conditional, use it
			// otherwise there was some initialization and it will need to be
			// a new conditional node as the child of the previous initialization
			// node.
			var conditional *db.ConditionalNode
			if cond, ok := current.(*db.ConditionalNode); ok && cond != nil {
				conditional = cond
			} else {
				relPath, _ := filepath.Rel(base, fset.File(expr.Pos()).Name())
				conditional = &db.ConditionalNode{
					Filename:   filepath.ToSlash(relPath),
					LineNumber: fset.Position(expr.Pos()).Line,
					Condition:  expressionString(expr),
				}
			}

			// Compute the success and fail trees if they haven't been computed already
			// and set the respective child pointers
			if succ, ok := fnCfg.blocks[block.Succs[0]]; ok {
				conditional.TrueChild = succ
				//conditional.Parent = current //??
			} else {
				conditional.TrueChild = fnCfg.constructSubCfg(block.Succs[0], base, fset, regexes)
				//conditional.Parent = current //??
				fnCfg.blocks[block.Succs[0]] = conditional.TrueChild
			}
			// set true child's parent
			if conditional.TrueChild != nil {
				conditional.TrueChild.SetParents(conditional)
			}

			if fail, ok := fnCfg.blocks[block.Succs[1]]; ok {
				conditional.FalseChild = fail
				// conditional.Parent = current //??
			} else {
				conditional.FalseChild = fnCfg.constructSubCfg(block.Succs[1], base, fset, regexes)
				// conditional.Parent = current //??
				fnCfg.blocks[block.Succs[1]] = conditional.FalseChild
			}
			// set false child's parent
			if conditional.FalseChild != nil {
				conditional.FalseChild.SetParents(conditional)
			}

			// Set the predecessor's child to be the conditional (which may be some initialization call)
			switch node := prev.(type) {
			case *db.FunctionNode:
				node.Child = db.Node(conditional)
				// node.Parent = current //?
				if node.Child != nil {
					node.Child.SetParents(node)
				}
			case *db.StatementNode:
				node.Child = db.Node(conditional)
				// node.Parent = current //?
				if node.Child != nil {
					node.Child.SetParents(node)
				}
			}
		} else if len(block.Succs) == 1 && last {
			// The last node was not a conditional but is the last statement, so
			// retrieve the child sub-cfg of the next block if it exits,
			// or otherwise compute it
			var child db.Node

			if subCfg, ok := fnCfg.blocks[block.Succs[0]]; ok {
				child = subCfg
			} else {
				child = fnCfg.constructSubCfg(block.Succs[0], base, fset, regexes)
				fnCfg.blocks[block.Succs[0]] = child
			}

			// Update the previous node's child
			switch node := prev.(type) {
			case *db.FunctionNode:
				node.Child = child
				// node.Parent = current
				if node.Child != nil {
					node.Child.SetParents(node)
				}
			case *db.StatementNode:
				node.Child = child
				// node.Parent = current
				if node.Child != nil {
					node.Child.SetParents(node)
				}
			}
		}

		current = nil
	}

	if len(block.Succs) == 1 {
		// The root was nil, so try to get the next block.
		// If the block is part of a for statement it would infinitely recurse, so leave it nil.
		if root == nil && !strings.Contains(block.String(), "for.post") {
			if subCfg, ok := fnCfg.blocks[block.Succs[0]]; ok {
				root = subCfg
			} else {
				root = fnCfg.constructSubCfg(block.Succs[0], base, fset, regexes)
				fnCfg.blocks[block.Succs[0]] = root
			}
		} else if root != nil && len(root.GetChildren()) < 1 && strings.Contains(block.String(), "if.done") {
			// End of a conditional, so make the successor node a child of the endif node
			if subCfg, ok := fnCfg.blocks[block.Succs[0]]; ok {
				root.SetChild([]db.Node{subCfg})
				if subCfg != nil {
					subCfg.SetParents(root)
				}
			} else {
				subCfg = fnCfg.constructSubCfg(block.Succs[0], base, fset, regexes)
				fnCfg.blocks[block.Succs[0]] = subCfg
				root.SetChild([]db.Node{subCfg})
				if subCfg != nil {
					subCfg.SetParents(root)
				}
			}
		} else if prev != nil && strings.Contains(block.String(), "for.body") {
			// Fast-forward through the post block and the loop block
			// and then get the for.done node and chain it with an endIf
			post := block.Succs[0]
			loop := post.Succs[0]
			if len(loop.Succs) > 1 {
				if subCfg, ok := fnCfg.blocks[loop.Succs[1]]; ok {
					prev.SetChild([]db.Node{subCfg})
					if subCfg != nil {
						subCfg.SetParents(prev)
					}
				} else {
					subCfg = fnCfg.constructSubCfg(loop.Succs[1], base, fset, regexes)
					fnCfg.blocks[loop.Succs[1]] = subCfg
					prev.SetChild([]db.Node{subCfg})
					if subCfg != nil {
						subCfg.SetParents(prev)
					}
				}
			}

		}
	}

	return
}

//Returns the expression Node
func getExprNode(expr ast.Expr, base string, fset *token.FileSet, conditional bool, regexes map[int]string) (node db.Node) {
	relPath, _ := filepath.Rel(base, fset.File(expr.Pos()).Name())
	switch expr := expr.(type) {
	case *ast.CallExpr:
		// fmt.Print("\t\tfound a callexpr ")
		if selectStmt, ok := expr.Fun.(*ast.SelectorExpr); ok {
			val := fmt.Sprint(selectStmt.Sel)
			// fmt.Println(val)

			// Check if the statement is a logging statement, if it is return a StatementNode
			if (strings.Contains(val, "Msg") || strings.Contains(val, "Err")) && logsource.IsFromLog(selectStmt) {
				line := fset.Position(expr.Pos()).Line
				node = db.Node(&db.StatementNode{
					Filename:   filepath.ToSlash(relPath),
					LineNumber: line,
					//TODO: there has to be a better way to assign this value.
					// This array is passed through every single function
					// in the recursion stack only to be used here?
					// If the file and linenumber are known,
					// why not just parse that information when needed?
					LogRegex:   logsource.GetLogRegexFromInfo(fset.File(expr.Pos()).Name(),line),
				})
			} else {
				// Was a method call.
				node = db.Node(&db.FunctionNode{
					Filename:     filepath.ToSlash(relPath),
					LineNumber:   fset.Position(expr.Pos()).Line,
					FunctionName: expressionString(selectStmt),
				})
			}
		} else {
			// fmt.Println(callExprName(expr))

			// Found a function call
			node = db.Node(&db.FunctionNode{
				Filename:     filepath.ToSlash(relPath),
				LineNumber:   fset.Position(expr.Pos()).Line,
				FunctionName: callExprName(expr),
			})
		}
	case *ast.UnaryExpr:
		subExpr := getExprNode(expr.X, base, fset, conditional, regexes)
		if conditional {
			// Found a unary conditional
			conditional := db.Node(&db.ConditionalNode{
				Filename:   filepath.ToSlash(relPath),
				LineNumber: fset.Position(expr.Pos()).Line,
				Condition:  expressionString(expr),
			})
			// subExpr was a function call of some kind
			if subExpr != nil {
				node = subExpr
				connectToLeaf(node, conditional)
			} else {
				// Normal condition
				node = conditional
			}
		} else if subExpr != nil {
			// Was a regular expression
			node = subExpr
		}
	case *ast.BinaryExpr:
		rightSubExpr := getExprNode(expr.X, base, fset, false, regexes)
		leftSubExpr := getExprNode(expr.Y, base, fset, false, regexes)
		if conditional {
			// Found a binary conditional
			conditional := db.Node(&db.ConditionalNode{
				Filename:   filepath.ToSlash(relPath),
				LineNumber: fset.Position(expr.Pos()).Line,
				Condition:  expressionString(expr),
			})
			if rightSubExpr != nil && leftSubExpr != nil {
				node = leftSubExpr
				connectToLeaf(node, rightSubExpr)
				connectToLeaf(rightSubExpr, conditional)
			} else if leftSubExpr != nil {
				node = leftSubExpr
				connectToLeaf(node, conditional)
			} else if rightSubExpr != nil {
				node = rightSubExpr
				connectToLeaf(node, conditional)
			} else {
				node = conditional
			}
		} else {
			// Found a binary sub-condition
			if rightSubExpr != nil && leftSubExpr != nil {
				node = leftSubExpr
				connectToLeaf(node, rightSubExpr)
			} else if leftSubExpr != nil {
				node = leftSubExpr
			} else if rightSubExpr != nil {
				node = rightSubExpr
			}
		}
	default:
		if conditional {
			// fmt.Println("\t\tfound a condition")
			node = db.Node(&db.ConditionalNode{
				Filename:   filepath.ToSlash(relPath),
				LineNumber: fset.Position(expr.Pos()).Line,
				Condition:  expressionString(expr),
			})
		}
	}
	return
}

// This function will connect the given node to the root's deepest child.
// Assumes only function nodes since it is the only situation I have enountered where this was necessary.
func connectToLeaf(root db.Node, node db.Node) {
	if call, ok := root.(*db.FunctionNode); ok {
		var current *db.FunctionNode
		for call != nil {
			if child, ok := call.Child.(*db.FunctionNode); ok && child != nil {
				current = child
				call = child

			} else {
				current = call
				call = nil
			}
		}
		// Chain the nodes together
		if current != nil {
			current.Child = node
			// current.Parent = root //?
			if current.Child != nil {
				current.Child.SetParents(current)
			}
		} else {
			call.Child = node
			//call.Parent = root //??
			if call.Child != nil {
				call.Child.SetParents(call)
			}
		}
	}
}

// Iterates over each item in a field list and does some operation passed to it
func iterateFields(fieldList *ast.FieldList, op func(returnType, name string)) {
	if fieldList != nil {
		for _, p := range fieldList.List {
			if p != nil {
				returnType := expressionString(p.Type)
				for _, name := range p.Names {
					varName := expressionString(name)
					op(returnType, varName)
				}
			}
		}
	}
}

func getFuncParams(fieldList *ast.FieldList) map[string]string {
	params := make(map[string]string)

	iterateFields(fieldList, func(returnType, name string) {
		params[name] = returnType
	})

	return params
}

func getFuncReturns(fieldList *ast.FieldList) []db.Return {
	returns := make([]db.Return, 0)

	iterateFields(fieldList, func(returnType, name string) {
		returns = append(returns, db.Return{
			Name:       name,
			ReturnType: returnType,
		})
	})

	return returns
}

//Connect expression nodes together
func chainExprNodes(exprs []ast.Expr, base string, fset *token.FileSet, regexes map[int]string) (first, current, prev db.Node) {
	for _, expr := range exprs {
		exprNode := getExprNode(expr, base, fset, false, regexes)
		// Initialize first node pointer
		if first == nil {
			first = exprNode
			prev = first
		} else {
			// Update current pointer
			switch exprNode := exprNode.(type) {
			case *db.FunctionNode:
				current = exprNode
			case *db.StatementNode:
				current = exprNode
			}

			// Chain nodes together
			if node, ok := prev.(*db.FunctionNode); ok && node != nil {
				node.Child = current
				//node.Parent = prev //??
				if node.Child != nil {
					node.Child.SetParents(node)
				}
			}

			prev = current
		}
	}
	return
}

func getStatementNode(stmt ast.Node, base string, fset *token.FileSet, regexes map[int]string) (node db.Node) {

	relPath, _ := filepath.Rel(base, fset.File(stmt.Pos()).Name())

	switch stmt := stmt.(type) {
	case *ast.ExprStmt:
		node = getExprNode(stmt.X, base, fset, false, regexes)
	case *ast.FuncDecl:
		receivers := getFuncParams(stmt.Recv)
		var params map[string]string
		var returns []db.Return
		if stmt.Type != nil {
			params = getFuncParams(stmt.Type.Params)
			if stmt.Type.Results != nil {
				returns = getFuncReturns(stmt.Type.Results)
			}
		} else {
			params = make(map[string]string)
			returns = make([]db.Return, 0)
		}

		node = db.Node(&db.FunctionDeclNode{
			Filename:     filepath.ToSlash(relPath),
			LineNumber:   fset.Position(stmt.Pos()).Line,
			FunctionName: stmt.Name.Name,
			Receivers:    receivers,
			Params:       params,
			Returns:      returns,
		})
	case *ast.AssignStmt:
		// Found an assignment
		node, _, _ = chainExprNodes(stmt.Rhs, base, fset, regexes)
	case *ast.ReturnStmt:
		// Find all function calls contained in the return statement
		node, _, _ = chainExprNodes(stmt.Results, base, fset, regexes)

		var bldr strings.Builder
		for i, result := range stmt.Results {
			bldr.WriteString(fmt.Sprintf("%s", expressionString(result)))
			if i < len(stmt.Results)-1 {
				bldr.WriteString(", ")
			}
		}
		expr := bldr.String()

		ret := db.Node(&db.ReturnNode{
			Filename:   filepath.ToSlash(relPath),
			LineNumber: fset.Position(stmt.Pos()).Line,
			Expression: expr,
		})

		if node != nil {
			// Append the return statement to the last function call
			connectToLeaf(node, ret)
		} else {
			node = ret
		}
	default:
		// fmt.Println("\t\tdid not cast")
	}
	return
}

// Recursively creates the string of an `ast.Expr`.
func expressionString(expr ast.Expr) string {
	if expr == nil {
		return ""
	}

	//Return expression based on the type
	switch condition := expr.(type) {
	case *ast.BasicLit:
		return condition.Value
	case *ast.Ident:
		return condition.Name
	case *ast.BinaryExpr:
		leftStr, rightStr := "", ""
		leftStr = expressionString(condition.X)
		rightStr = expressionString(condition.Y)
		return fmt.Sprint(leftStr, condition.Op, rightStr)
	case *ast.UnaryExpr:
		op := condition.Op.String()
		str := expressionString(condition.X)
		return fmt.Sprint(op, str)
	case *ast.SelectorExpr:
		selector := ""
		if condition.Sel != nil {
			selector = condition.Sel.String()
		}
		str := expressionString(condition.X)
		return fmt.Sprintf("%s.%s", str, selector)
	case *ast.ParenExpr:
		return fmt.Sprintf("(%s)", expressionString(condition.X))
	case *ast.CallExpr:
		fn := expressionString(condition.Fun)
		args := make([]string, 0)
		for _, arg := range condition.Args {
			args = append(args, expressionString(arg))
		}
		if condition.Ellipsis != token.NoPos {
			args[len(args)-1] = fmt.Sprintf("%s...", args[len(args)-1])
		}

		var builder strings.Builder
		_, _ = builder.WriteString(fmt.Sprintf("%s(", fn))
		for i, arg := range args {
			var s string
			if i == len(args)-1 {
				s = fmt.Sprintf("%s)", arg)
			} else {
				s = fmt.Sprintf("%s, ", arg)
			}
			_, _ = builder.WriteString(s)
		}
		if len(args) == 0 {
			_, _ = builder.WriteString(")")
		}

		return builder.String()
	case *ast.IndexExpr:
		expr := expressionString(condition.X)
		ndx := expressionString(condition.Index)
		return fmt.Sprintf("%s[%s]", expr, ndx)
	case *ast.KeyValueExpr:
		key := expressionString(condition.Key)
		value := expressionString(condition.Value)
		return fmt.Sprint(key, ":", value)
	case *ast.SliceExpr: // not sure about this one
		expr := expressionString(condition.X)
		low := expressionString(condition.Low)
		high := expressionString(condition.High)
		if condition.Slice3 {
			max := expressionString(condition.Max)
			return fmt.Sprintf("%s[%s : %s : %s]", expr, low, high, max)
		}
		return fmt.Sprintf("%s[%s : %s]", expr, low, high)
	case *ast.StarExpr:
		expr := expressionString(condition.X)
		return fmt.Sprintf("*%s", expr)
	case *ast.TypeAssertExpr:
		expr := expressionString(condition.X)
		typecast := expressionString(condition.Type)
		return fmt.Sprintf("%s(%s)", typecast, expr)
	}
	return ""
}

//Gets the name of a call expression
func callExprName(call *ast.CallExpr) string {
	fn := expressionString(call)
	name := ""
	if s := strings.Split(fn, "("); len(s) > 0 {
		name = s[0]
	}
	return name
}

// CopyCfg lets you copy a CFG beginning at its root
func CopyCfg(root db.Node) db.Node {
	if root == nil {
		return nil
	}
	return copyCfgRecur(root, make(map[db.Node]db.Node))
}

func copyCfgRecur(node db.Node, copied map[db.Node]db.Node) (copy db.Node) {
	if node == nil {
		return nil
	}

	switch node := node.(type) {
	case *db.FunctionDeclNode:
		copy = &db.FunctionDeclNode{
			FunctionName: node.FunctionName,
			Receivers:    node.Receivers,
			Params:       node.Params,
			Returns:      node.Returns,
			Child:        copyChild(node.Child, copied),
		}
	case *db.FunctionNode:
		copy = &db.FunctionNode{
			FunctionName: node.FunctionName,
			Child:        copyChild(node.Child, copied),
		}
	case *db.ConditionalNode:
		copy = &db.ConditionalNode{
			Condition:  node.Condition,
			TrueChild:  copyChild(node.TrueChild, copied),
			FalseChild: copyChild(node.FalseChild, copied),
		}
	case *db.StatementNode:
		copy = &db.StatementNode{
			LogRegex: node.LogRegex,
			Child:    copyChild(node.Child, copied),
		}
	case *db.ReturnNode:
		copy = &db.ReturnNode{
			Expression: node.Expression,
			Child:      copyChild(node.Child, copied),
		}
	}

	if copy != nil {
		copy.SetLineNumber(node.GetLineNumber())
		copy.SetFilename(node.GetFilename())
	}
	return copy
}

func copyChild(node db.Node, copied map[db.Node]db.Node) db.Node {
	var copy db.Node
	if node != nil {
		if n, ok := copied[node]; ok {
			copy = n
		} else {
			copy = copyCfgRecur(node, copied)
			copied[node] = copy
		}
	}
	return copy
}

func traverse(root db.Node, visit func(db.Node)) {
	if root == nil {
		return
	}
	visit(root)

	children := root.GetChildren()
	for child := range children {
		traverse(child, visit)
	}
}
