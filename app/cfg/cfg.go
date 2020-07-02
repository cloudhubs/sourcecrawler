package cfg

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sourcecrawler/app/db"
	"sourcecrawler/app/logsource"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"golang.org/x/tools/go/cfg"
)

// FnCfgCreator allows you to compute the CFG for a given function declaration
type FnCfgCreator struct {
	blocks map[*cfg.Block]db.Node

	base string
	fset *token.FileSet

	// Map of literal identifiers to their CFG
	FnLiterals map[string]*db.FunctionDeclNode

	// The current package
	CurPkg     string
	curFnDecl  string
	curFnLitID uint

	// Properties for keeping track of scope
	varNameToStack map[string][]string //maps var names to scope names that represent the scopes this variable was seen in
	scopeCount     []uint              //Holds the current counts for each level of the current subscopes

}

// NewFnCfgCreator returns a newly initialized FnCfgCreator
func NewFnCfgCreator(pkg string, base string, fset *token.FileSet) *FnCfgCreator {
	return &FnCfgCreator{
		blocks:         make(map[*cfg.Block]db.Node),
		base:           base,
		fset:           fset,
		FnLiterals:     make(map[string]*db.FunctionDeclNode),
		CurPkg:         pkg,
		curFnLitID:     1,
		varNameToStack: make(map[string][]string),
		scopeCount:     make([]uint, 0),
	}
}

func (fnCfg *FnCfgCreator) getCurrentScope() string {
	b := strings.Builder{}
	for i, s := range fnCfg.scopeCount {
		b.WriteString(strconv.Itoa(int(s)))
		if i < len(fnCfg.scopeCount)-1 {
			b.WriteString(".")
		}
	}
	return b.String()
}

func (fnCfg *FnCfgCreator) enterScope() {
	fnCfg.scopeCount[len(fnCfg.scopeCount)-1]++
	fnCfg.scopeCount = append(fnCfg.scopeCount, 0)
}

func (fnCfg *FnCfgCreator) leaveScope() {
	fnCfg.scopeCount = fnCfg.scopeCount[0 : len(fnCfg.scopeCount)-1]
}

func (fnCfg *FnCfgCreator) debugScope(where string) {
	fmt.Println("scopeCount:", fnCfg.scopeCount, "\t", "varToStack:", fnCfg.varNameToStack, "where:", where)
}

func (fnCfg *FnCfgCreator) currFnLiteralID() string {
	return fmt.Sprintf("%s.%s.func%v", fnCfg.CurPkg, fnCfg.curFnDecl, fnCfg.curFnLitID)
}

// CreateCfg creates the CFG For a given function declaration, filepath and file, and a map of regexes contained within the file.
func (fnCfg *FnCfgCreator) CreateCfg(fn *ast.FuncDecl) db.Node {
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
	fnCfg.FnLiterals = make(map[string]*db.FunctionDeclNode)
	fnCfg.curFnDecl = fn.Name.Name
	fnCfg.curFnLitID = 1
	fnCfg.varNameToStack = make(map[string][]string)
	fnCfg.scopeCount = make([]uint, 1)

	// Function declaration is the root node
	root := fnCfg.getStatementNode(fn)

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

	//fmt.Println("Func", fn.Name.Name)
	//for _, block := range cfg.Blocks{
	//	fmt.Println(block)
	//}
	// fmt.Println(fn.Name.Name)
	// fmt.Println(cfg.Format(fset))

	// Empty function declaration
	if len(cfg.Blocks) < 1 {
		return root
	}

	// Begin constructing the cfg
	fmt.Println("scope entering:", fnCfg.curFnDecl)
	fnCfg.debugScope("begin")
	block := cfg.Blocks[0]
	node := fnCfg.constructSubCfg(block)
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
	fmt.Println()
	return root
}

//from the name of a function and its file, create the cfg for it
//this will be useful for connecting external functions after an
//initial cfg is created
func (fnCfg *FnCfgCreator) CreateCfgFromFunctionName(fnName string, files []string, seenFn []*db.FunctionNode) db.Node {
	fset := token.NewFileSet()
	found := false
	var fn *ast.FuncDecl
	for _, file := range files {
		node, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
		if err != nil {
			panic(err)
		}
		ast.Inspect(node, func(n ast.Node) bool {
			if n, ok := n.(*ast.FuncDecl); ok {
				if strings.Contains(fnName, n.Name.Name) {
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
		node := fnCfg.CreateCfg(fn)
		//add in functions in this cfg, excluding
		//any functions already seen in this scope
		//or higher scopes
		ConnectExternalFunctions(node, seenFn, files, fnCfg.base)
		return node
	}
	return nil
}

//initially called with:
//root = first iteration of cfg containing stacktrace functions
//seenFns = []*db.FunctionNode{} (empty)
//base = project root (needed for cfg construction)
//regexes = NOT NEEDED/DEPRICATED ARRAY REPLACED BY INLINE FUNCTION TO GRAB REGEX
func ConnectExternalFunctions(root db.Node, seenFns []*db.FunctionNode, sourceFiles []string, base string) {
	var fnCfg FnCfgCreator
	node := root
	for node != nil {
		var tmp db.Node
		//traverse down tree until
		//encountering a function node
		//that is not followed by a
		//function declaration node
		//(which means it is already connected)
		if node, ok := node.(*db.FunctionNode); ok {
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
					newFn := fnCfg.CreateCfgFromFunctionName(node.FunctionName, sourceFiles, append(seenFns, node))
					if newFn != nil {
						//add dummy return node to consolidate returns
						tmp = node.Child
						tmpReturn := &db.ReturnNode{
							Filename:   node.Child.GetFilename(),
							LineNumber: node.Child.GetLineNumber(),
							Expression: "",
							Child:      node.Child,
							Parents:    nil,
							Label:      0,
						}

						for _, leaf := range getLeafNodes(newFn) {
							leaf.SetChild([]db.Node{tmpReturn})
							tmpReturn.SetParents(leaf)
						}
						node.Child = newFn
					}
				}
			}
		}

		if tmp != nil {
			node = tmp
		} else {
			for child := range node.GetChildren() {
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
	if node.GetParents() != nil {
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
	case *db.VariableNode:
		fmt.Printf(node.GetProperties())
		PrintCfg(node.Child, level)
	}
}

func (fnCfg *FnCfgCreator) constructSubCfg(block *cfg.Block) (root db.Node) {
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
			fnCfg.leaveScope()
			fnCfg.debugScope("ifdone/fordone")
			current = &db.EndConditionalNode{}
			fnCfg.blocks[block] = current
		}
		prev = current
		root = current
	}

	// Convert each node in the block into a db.Node (if it is one we want to keep)
	for i, node := range block.Nodes {
		last := i == len(block.Nodes)-1
		conditional = len(block.Succs) > 1 // 2 successors for conditional block

		//Process node based on its type
		switch node := node.(type) {
		case ast.Stmt:
			current = fnCfg.getStatementNode(node)
		case ast.Expr:
			current = fnCfg.getExprNode(node, last && conditional)
		case ast.Spec: //TODO: handling variable nodes (declarations)
			current = fnCfg.getSpecNode(node)
			if current != nil {
				//fmt.Println("Variable node exists", current.GetProperties())
			}
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
			//TODO: Variable node case
		case *db.VariableNode:
			prevNode.Child = current
			if prevNode.Child != nil {
				prevNode.Child.SetParents(prevNode)
			}
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
				relPath, _ := filepath.Rel(fnCfg.base, fnCfg.fset.File(expr.Pos()).Name())
				conditional = &db.ConditionalNode{
					Filename:   filepath.ToSlash(relPath),
					LineNumber: fnCfg.fset.Position(expr.Pos()).Line,
					Condition:  expressionString(expr),
				}
			}

			// Compute the success and fail trees if they haven't been computed already
			// and set the respective child pointers
			if succ, ok := fnCfg.blocks[block.Succs[0]]; ok {
				conditional.TrueChild = succ
				//conditional.Parent = current //??
			} else {
				fnCfg.enterScope()
				fnCfg.debugScope("truechild")
				conditional.TrueChild = fnCfg.constructSubCfg(block.Succs[0])
				if len(block.Succs[0].Succs) == 0 {
					fnCfg.leaveScope()
					fnCfg.debugScope("truechild no successor")
				}
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
				fnCfg.enterScope()
				fnCfg.debugScope("falsechild")
				conditional.FalseChild = fnCfg.constructSubCfg(block.Succs[1])
				if len(block.Succs[0].Succs) == 0 {
					fnCfg.leaveScope()
					fnCfg.debugScope("falsechild no successor")
				}
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
				child = fnCfg.constructSubCfg(block.Succs[0])
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
				root = fnCfg.constructSubCfg(block.Succs[0])
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
				subCfg = fnCfg.constructSubCfg(block.Succs[0])
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
					subCfg = fnCfg.constructSubCfg(loop.Succs[1])
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

//Returns a variable node if it is encountered
// TODO: ast.ValueSpec only holds a constant or variable declaration
//  ast.AssignStmt handles variable assignments
func (fnCfg *FnCfgCreator) getSpecNode(spec ast.Spec) (node db.Node) {
	relPath, _ := filepath.Rel(fnCfg.base, fnCfg.fset.File(spec.Pos()).Name())

	switch spec := spec.(type) {
	case *ast.ValueSpec:
		var scopeID string = "" //TODO: may need more work
		var varName string = ""
		initType := ""
		initValues := ""
		stackStr := ""

		//fmt.Println("Current function", fnCfg.curFnDecl)

		//Grab the variable type
		if spec.Type != nil {
			varType := fmt.Sprint(spec.Type)
			//fmt.Println("Var type:", varType)
			initType = varType
		}

		//Set variable name
		if len(spec.Names) > 0 {
			varName = spec.Names[0].Name
			//fmt.Print("Var name:", varName)
		}

		//Get variable initial value if it exists
		for _, expr := range spec.Values {
			if expr != nil {
				varVal := fmt.Sprint(expr)
				//fmt.Println("Initial value:", varVal)
				initValues = varVal
			}
		}
		fmt.Println()

		//TODO: handle variable scoping
		//If variable is not in the map, then add to map and its scope (add a new state)
		value, ok := fnCfg.varNameToStack[varName]
		if ok {
			fmt.Println(varName, value, " was found in the map")
		} else {
			for index := range fnCfg.scopeCount {
				//Exclude last element
				if index == len(fnCfg.scopeCount)-1 {
					break
				}

				//Add the scope (Ex: 1.1.2)
				stackStr += fmt.Sprint(fnCfg.scopeCount[index])

				//If last element, dont add a .
				if index == len(fnCfg.scopeCount)-2 {
					continue
				} else {
					stackStr += "."
				}
			}
			//If there was only 1 element in master stack, set to 0
			if stackStr == "" {
				stackStr = "0"
			}
			//Create entry
			fnCfg.varNameToStack[varName] = append(fnCfg.varNameToStack[varName], stackStr)
		}

		//Create scopeID string (ex: testFunc.1.1.2)
		scopeID = fnCfg.curFnDecl + "." + stackStr

		//Add variable node to cfg
		node = db.Node(&db.VariableNode{
			Filename:        filepath.ToSlash(relPath),
			LineNumber:      fnCfg.fset.Position(spec.Pos()).Line,
			ScopeId:         scopeID,
			VarName:         varName,
			Value:           initType + initValues,
			Parent:          nil,
			Child:           nil,
			ValueFromParent: false,
		})

		fmt.Println("Declaration", node.GetProperties())
		//TODO: connect to parent function?

	case *ast.ImportSpec:
		//fmt.Println(spec.Name.Name)
	case *ast.TypeSpec:
		//fmt.Println(spec.Type)
	}

	return node
}

//Returns the expression Node
func (fnCfg *FnCfgCreator) getExprNode(expr ast.Expr, conditional bool) (node db.Node) {
	relPath, _ := filepath.Rel(fnCfg.base, fnCfg.fset.File(expr.Pos()).Name())
	switch expr := expr.(type) {
	case *ast.CallExpr:
		// fmt.Print("\t\tfound a callexpr ")
		//fmt.Println("EXPRESSION args", expr.Args)
		switch fn := expr.Fun.(type) {
		case *ast.SelectorExpr:

			val := fmt.Sprint(fn.Sel)
			// fmt.Println(val)

			// Check if the statement is a logging statement, if it is return a StatementNode
			if (strings.Contains(val, "Msg") || strings.Contains(val, "Err")) && logsource.IsFromLog(fn) {
				line := fnCfg.fset.Position(expr.Pos()).Line
				node = db.Node(&db.StatementNode{
					Filename:   filepath.ToSlash(relPath),
					LineNumber: line,
					//TODO: there has to be a better way to assign this value.
					// This array is passed through every single function
					// in the recursion stack only to be used here?
					// If the file and linenumber are known,
					// why not just parse that information when needed?
					LogRegex: logsource.GetLogRegexFromInfo(fnCfg.fset.File(expr.Pos()).Name(), line),
				})
			} else {
				// Was a method call.
				node = db.Node(&db.FunctionNode{
					Filename:     filepath.ToSlash(relPath),
					LineNumber:   fnCfg.fset.Position(expr.Pos()).Line,
					FunctionName: expressionString(fn),
				})
			}
		case *ast.FuncLit:
			// CallExpr needs some way of knowing what function it refers to if its a literal, i.e. creating
			// the stack trace type identifier for it (doesn't have to be exactly the same I think) where
			// when we connect everything it can refer to that if it's a function literal call. Maybe add
			// a boolean flag to the FunctionNode
			ident := fnCfg.currFnLiteralID()
			_ = fnCfg.getExprNode(fn, false)
			node = &db.FunctionNode{
				Filename:     filepath.ToSlash(relPath),
				LineNumber:   fnCfg.fset.Position(expr.Pos()).Line,
				FunctionName: ident,
			}
		default:
			// fmt.Println(callExprName(expr))

			// Found a function call
			node = db.Node(&db.FunctionNode{
				Filename:     filepath.ToSlash(relPath),
				LineNumber:   fnCfg.fset.Position(expr.Pos()).Line,
				FunctionName: callExprName(expr),
			})
		}
	case *ast.FuncLit:
		if expr.Body != nil {
			cfg := cfg.New(expr.Body, func(call *ast.CallExpr) bool { return false })
			if cfg != nil && len(cfg.Blocks) > 0 {
				name := fnCfg.currFnLiteralID()
				fnCfg.curFnLitID++
				node := fnCfg.constructSubCfg(cfg.Blocks[0])
				root := &db.FunctionDeclNode{
					Filename:     filepath.ToSlash(relPath),
					LineNumber:   fnCfg.fset.Position(expr.Pos()).Line,
					FunctionName: name,
					Params:       getFuncParams(expr.Type.Params),
					Receivers:    map[string]string{},
					Returns:      getFuncReturns(expr.Type.Results),
					Child:        node,
				}
				node.SetParents(root)
				fnCfg.FnLiterals[name] = root
				// TODO: Return a VariableNode with the function literal identifier (maybe the value too? depends on the interface)
				//       It should probably have some sort of flag to indicate the value is a placeholder for a function literal.
				// node = root
			}
		}
	case *ast.UnaryExpr:
		subExpr := fnCfg.getExprNode(expr.X, conditional)
		if conditional {
			// Found a unary conditional
			conditional := db.Node(&db.ConditionalNode{
				Filename:   filepath.ToSlash(relPath),
				LineNumber: fnCfg.fset.Position(expr.Pos()).Line,
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

		rightSubExpr := fnCfg.getExprNode(expr.X, false)
		leftSubExpr := fnCfg.getExprNode(expr.Y, false)
		if conditional {
			// Found a binary conditional
			conditional := db.Node(&db.ConditionalNode{
				Filename:   filepath.ToSlash(relPath),
				LineNumber: fnCfg.fset.Position(expr.Pos()).Line,
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
				LineNumber: fnCfg.fset.Position(expr.Pos()).Line,
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
func (fnCfg *FnCfgCreator) chainExprNodes(exprs []ast.Expr) (first, current, prev db.Node) {
	for _, expr := range exprs {
		exprNode := fnCfg.getExprNode(expr, false)
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
				//TODO: variable node
			case *db.VariableNode:
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

func (fnCfg *FnCfgCreator) getStatementNode(stmt ast.Node) (node db.Node) {

	relPath, _ := filepath.Rel(fnCfg.base, fnCfg.fset.File(stmt.Pos()).Name())

	switch stmt := stmt.(type) {
	case *ast.ExprStmt:
		node = fnCfg.getExprNode(stmt.X, false)
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
			LineNumber:   fnCfg.fset.Position(stmt.Pos()).Line,
			FunctionName: stmt.Name.Name,
			Receivers:    receivers,
			Params:       params,
			Returns:      returns,
		})
	case *ast.AssignStmt: //TODO: handle variable assignment
		node, _, _ = fnCfg.chainExprNodes(stmt.Rhs)

		//Grab the variable name
		strLHS := fmt.Sprint(stmt.Lhs) //variable name
		varName := strLHS[strings.Index(strLHS, "[")+1 : strings.Index(strLHS, "]")]

		//Grab the value being assigned
		//strRHS := fmt.Sprint(stmt.Rhs) //the value
		//assignValue := strRHS[strings.Index(strRHS, "[")+1:strings.Index(strRHS, "]")]

		//Get the expression
		//strExpr := stmt.Tok.String()   //assignment operator
		//var scopeID string = ""
		//fmt.Printf("(%s %s %s)\n", varName, strExpr, assignValue)

		//Handling variable scoping at assign time
		stackStr := ""
		if value, ok := fnCfg.varNameToStack[varName]; ok {
			//Add all elements as the scope
			for index := range value {
				stackStr += value[index]
				//If last element, dont add .
				if index == len(value)-1 {
					break
				} else {
					stackStr += "."
				}
			}
		} else { //TODO: handle adding scope if variable not in map at assign time
			fnCfg.varNameToStack[varName] = append(fnCfg.varNameToStack[varName], "1")
		}

		//scopeID = fnCfg.curFnDecl + "." + stackStr
		//fmt.Println("Scope id", scopeID)

		//Build variable node
		//TODO: throwing stack overflow error if variable node is created
		//varNode = db.Node(&db.VariableNode{
		//	Filename:        filepath.ToSlash(relPath),
		//	LineNumber:      fset.Position(stmt.Pos()).Line,
		//	ScopeId:         scopeID,
		//	VarName:         strLHS,
		//	Value:           varName + strExpr + assignValue, //the expression (ex: x := 5)
		//	Parent:          nil,
		//	Child:           nil,
		//	ValueFromParent: false,
		//})

		//fmt.Println(varNode.GetProperties())

		//if node != nil {
		//	// Append the function node to the last function call
		//	connectToLeaf(node, varNode)
		//} else {
		//	node = varNode
		//}

	case *ast.ReturnStmt:
		// Find all function calls contained in the return statement
		node, _, _ = fnCfg.chainExprNodes(stmt.Results)

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
			LineNumber: fnCfg.fset.Position(stmt.Pos()).Line,
			Expression: expr,
		})

		if node != nil {
			// Append the return statement to the last function call
			connectToLeaf(node, ret)
		} else {
			node = ret
		}
	case *ast.GoStmt:
		node = fnCfg.getExprNode(stmt.Call, false)
	case *ast.DeferStmt:
		node = fnCfg.getExprNode(stmt.Call, false)
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
	case *ast.FuncType:
		params := getFuncParams(condition.Params)
		rets := getFuncReturns(condition.Results)
		b := strings.Builder{}
		b.Write([]byte("func("))
		i := 0
		for name, t := range params {
			b.Write([]byte(fmt.Sprintf("%s %s", name, t)))
			if i < len(params)-1 {
				b.Write([]byte(", "))
			}
		}
		b.Write([]byte(")"))
		for i, ret := range rets {
			b.Write([]byte(fmt.Sprintf("%s %s", ret.Name, ret.ReturnType)))
			if i < len(params)-1 {
				b.Write([]byte(", "))
			}
		}
		return b.String()
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
