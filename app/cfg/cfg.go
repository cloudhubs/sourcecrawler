package cfg

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
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

	//Holds all variable nodes
	varList map[string]*db.VariableNode
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
		varList:        make(map[string]*db.VariableNode),
	}
}

func (fnCfg *FnCfgCreator) getCurrentScope() string {
	b := strings.Builder{}
	b.WriteString(fnCfg.CurPkg)
	b.WriteString(".")
	b.WriteString(fnCfg.curFnDecl)
	b.WriteString(".")
	for i, s := range fnCfg.scopeCount {
		b.WriteString(strconv.Itoa(int(s)))
		if i < len(fnCfg.scopeCount)-1 {
			b.WriteString(".")
		}
	}
	return b.String()
}

func (fnCfg *FnCfgCreator) enterScope() {
	if len(fnCfg.scopeCount) > 0 {
		fnCfg.scopeCount[len(fnCfg.scopeCount)-1]++
	}
	fnCfg.scopeCount = append(fnCfg.scopeCount, 0)
}

func (fnCfg *FnCfgCreator) leaveScope() {
	if len(fnCfg.scopeCount) > 1 {
		fnCfg.scopeCount = fnCfg.scopeCount[0 : len(fnCfg.scopeCount)-1]
	}
	// Remove out of scope variable declarations
	for key, stack := range fnCfg.varNameToStack {
		if len(stack) > 0 {
			// Check if the latest declaration is out of scope
			varScope := stack[len(stack)-1]
			scope := fnCfg.getCurrentScope()
			if len(varScope) > len(scope) {
				fnCfg.varNameToStack[key] = stack[0 : len(stack)-1]
			}
		}
	}
}

func (fnCfg *FnCfgCreator) debugScope(where string) {
	fmt.Println("scopeCount:", fnCfg.scopeCount, "\t", "varToStack:", fnCfg.varNameToStack, "where:", where)
}

func (fnCfg *FnCfgCreator) currFnLiteralID() string {
	return fmt.Sprintf("%s.%s.func%v", fnCfg.CurPkg, fnCfg.curFnDecl, fnCfg.curFnLitID)
}

//Process a variable node if it found
func (fnCfg *FnCfgCreator) processVariable(varNode *db.VariableNode) (string, []*db.VariableNode){
	varExpr := varNode.Value

	involvedVars := []*db.VariableNode{}

	//If it is symbolic -> get the other variable node to keep track of
	//May need more depth if a variable is assigned from another variable that was assigned a variable
	// (ex:  c:=10,  -> b := c, a:= b)
	if !varNode.IsReal{
		startNdx := strings.Index(varExpr, "=")+2
		symbol := varExpr[startNdx:] // could be a func() or a variable name

		//Check if the value came from another variable ()
		for _, node := range fnCfg.varList{
			if strings.Contains(symbol, node.VarName){
				involvedVars = append(involvedVars, fnCfg.varList[node.ScopeId]) //add the other var node to keep track
			}
		}

	}

	return varExpr, involvedVars
}

// CreateCfg creates the CFG For a given function declaration
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
	//fnCfg.debugScope("begin")
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

	//Test print the list of variables
	for key, node := range fnCfg.varList {
		fmt.Println("scope id", key)
		fmt.Println(node.GetProperties())
	}


	return root
}

//from the name of a function and its file, create the cfg for it
//this will be useful for connecting external functions after an
//initial cfg is created
func (fnCfg *FnCfgCreator) CreateCfgFromFunctionName(fnName string, files []string, seenFn []*db.FunctionNode) db.Node {
	found := false
	var fn *ast.FuncDecl
	for _, file := range files {
		node, err := parser.ParseFile(fnCfg.fset, file, nil, parser.ParseComments)
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
		fnCfg.ConnectExternalFunctions(node, seenFn, files)
		return node
	}
	return nil
}

//initially called with:
//root = first iteration of cfg containing stacktrace functions
//seenFns = []*db.FunctionNode{} (empty)
//base = project root (needed for cfg construction)
func (fnCfg *FnCfgCreator) ConnectExternalFunctions(root db.Node, seenFns []*db.FunctionNode, sourceFiles []string) {

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
						//TODO: insert VariableNodes here from FunctionNode Args
						// and FunctionDeclNode Params
						// node is FunctionNode, newFn is FunctionDeclNode
						vars := make([]*db.VariableNode, len(node.Args))
						if decl, ok := newFn.(*db.FunctionDeclNode); ok {
							for i, arg := range node.Args {
								vars[i] = &db.VariableNode{
									Filename:        node.Filename,
									LineNumber:      node.LineNumber,
									ScopeId:         fnCfg.getCurrentScope(), //TODO: get scope?
									VarName:         arg.VarName,
									Value:           decl.Params[i].VarName, //TODO: ^^get opinion on the order of these
									Parent:          nil,
									Child:           nil,
									ValueFromParent: false,
								}
							}

						}

						//hold next node to skip the subtree attached
						tmp = node.Child

						//connect first var to ref
						if len(vars) > 0 {
							node.Child = vars[0]
							vars[0].Parent = node
						}

						//chain vars together
						for i, variable := range vars {
							//skip last
							if i != len(vars)-1 {
								variable.Child = vars[i+1]
								vars[i+1].Parent = variable
							}
						}

						//connect last to functionBody
						if len(vars) > 0 {
							vars[len(vars)-1].Child = newFn
							newFn.SetParents(vars[len(vars)-1])
						} else {
							node.Child = newFn
							newFn.SetParents(node)
						}

						//add dummy return node to consolidate returns
						tmpReturn := &db.ReturnNode{
							Filename:   node.Child.GetFilename(),
							LineNumber: node.Child.GetLineNumber(),
							Expression: "",
							Child:      tmp,
							Parents:    nil,
							Label:      0,
						}

						for _, leaf := range getLeafNodes(newFn) {
							leaf.SetChild([]db.Node{tmpReturn})
							tmpReturn.SetParents(leaf)
						}
					}
				}
			}
		}

		if tmp != nil {
			node = tmp
		} else {
			for child := range node.GetChildren() {
				//repeat
				fnCfg.ConnectExternalFunctions(child, seenFns, sourceFiles)
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
	var parStr = "Parent: "
	if node.GetParents() != nil {
		//parStr += node.GetProperties()
		parStr = node.GetFilename()
	}

	switch node := node.(type) {
	case *db.FunctionDeclNode:
		fmt.Printf("%sDeclaration--Receivers: (%v) Signature: %s(%v) Returns: (%v) Label: (%v) ParStr(%v)\n", level,
			node.Receivers, node.FunctionName, node.Params, node.Returns, node.Label, parStr)
		PrintCfg(node.Child, level+"  ")
	case *db.FunctionNode:
		fmt.Printf("%sFunction--Name: %s Label: (%v) parStr: (%v) \n", level, node.FunctionName, node.Label, parStr)
		PrintCfg(node.Child, level)
	case *db.StatementNode:
		fmt.Printf("%sStatement--Regex: %s Label: (%v) ParStr: (%v)\n", level, node.LogRegex, node.Label, parStr)
		PrintCfg(node.Child, level)
	case *db.ConditionalNode:
		fmt.Printf("%sif %s Label: (%v) ParStr: (%v)\n", level, node.Condition, node.Label, parStr)
		PrintCfg(node.TrueChild, level+"  ")
		fmt.Println(level + "else")
		PrintCfg(node.FalseChild, level+"  ")
	case *db.ReturnNode:
		fmt.Printf("%sreturn %s Label: (%v) ParStr: (%v)\n", level, node.Expression, node.Label, parStr)
		lv := ""
		for i := 0; i < len(level)-2; i++ {
			lv += " "
		}
		PrintCfg(node.Child, lv)
	case *db.EndConditionalNode:
		fmt.Printf("%sendIf Label: (%v)\n", level, node.Label)
		PrintCfg(node.Child, level)
	case *db.VariableNode:
		fmt.Printf("%sVariable--Name: %v Value: %v Scope: %v\n", level, node.VarName, node.Value, node.ScopeId)
		if node.Child != node {
			PrintCfg(node.Child, level)
		}
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
			//fnCfg.debugScope("ifdone/fordone")
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

			//TODO: Variable node - set child and parent?
		case *db.VariableNode:
			//fmt.Println("curr is var node")
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
					Condition:  fnCfg.expressionString(expr),
					VarsUsed:   fnCfg.extractUsedVars(expr),
				}

			}

			// Compute the success and fail trees if they haven't been computed already
			// and set the respective child pointers
			if succ, ok := fnCfg.blocks[block.Succs[0]]; ok {
				conditional.TrueChild = succ
				//conditional.Parent = current //??
			} else {
				fnCfg.enterScope()
				//fnCfg.debugScope("truechild")
				conditional.TrueChild = fnCfg.constructSubCfg(block.Succs[0])
				if len(block.Succs[0].Succs) == 0 {
					fnCfg.leaveScope()
					//fnCfg.debugScope("truechild no successor")
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
				//fnCfg.debugScope("falsechild")
				conditional.FalseChild = fnCfg.constructSubCfg(block.Succs[1])
				if len(block.Succs[0].Succs) == 0 {
					fnCfg.leaveScope()
					//fnCfg.debugScope("falsechild no successor")
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
//  ast.ValueSpec only holds a constant or variable declaration
//  ast.AssignStmt handles variable assignments
func (fnCfg *FnCfgCreator) getSpecNode(spec ast.Spec) (node db.Node) {
	relPath, _ := filepath.Rel(fnCfg.base, fnCfg.fset.File(spec.Pos()).Name())

	switch spec := spec.(type) {
	case *ast.ValueSpec:
		var scopeID string = ""
		var varName string = ""
		initType := ""
		initVal := ""
		exprStr := ""

		//Grab the variable type
		if spec.Type != nil {
			varType := fmt.Sprint(spec.Type)
			initType = varType
		}

		//Set variable name
		if len(spec.Names) > 0 {
			varName = spec.Names[0].Name
		}

		//Get variable initial value if it exists
		for _, expr := range spec.Values {
			if expr != nil {
				varVal := fmt.Sprint(expr)
				//fmt.Println("Initial value:", varVal)
				initVal += varVal
			}
		}
		//fmt.Println()

		//Set expr string
		exprStr = "var " + varName + " " + initType + " = " + initVal

		//If variable is not in the map, then add to map and its scope (add a new state)
		value, ok := fnCfg.varNameToStack[varName]
		if ok {
			fmt.Println(varName, value, " was found in the map")
			fnCfg.varNameToStack[varName] = append(fnCfg.varNameToStack[varName], fnCfg.getCurrentScope()) //append
		} else {
			//scopeID = fnCfg.getCurrentScope()
			//for index := range fnCfg.scopeCount {
			//	//Exclude last element
			//	if index == len(fnCfg.scopeCount)-1 {
			//		break
			//	}
			//
			//	//Add the scope (Ex: 1.1.2)
			//	stackStr += fmt.Sprint(fnCfg.scopeCount[index])
			//
			//	//If last element, dont add a .
			//	if index == len(fnCfg.scopeCount)-2 {
			//		continue
			//	} else {
			//		stackStr += "."
			//	}
			//}
			////If there was only 1 element in master stack, set to 0
			//if stackStr == "" {
			//	stackStr = "0"
			//}

			//Create entry
			fnCfg.varNameToStack[varName] = append(fnCfg.varNameToStack[varName], fnCfg.getCurrentScope())
		}


		//Create scopeID string (ex: testFunc.1.1.2)
		//scopeID = fnCfg.curFnDecl + "." + stackStr
		scopeID = fnCfg.getCurrentScope()
		fmt.Println("Current scope in var decl", scopeID)

		//Add variable node to cfg
		//TODO: not shown in current cfg since most vars are handled in the assignStmt
		node = db.Node(&db.VariableNode{
			Filename:        filepath.ToSlash(relPath),
			LineNumber:      fnCfg.fset.Position(spec.Pos()).Line,
			ScopeId:         scopeID,
			VarName:         varName,
			Value:           exprStr,
			Parent:          nil,
			Child:           nil,
			ValueFromParent: false,
		})

		//fmt.Println("Var declaration:", node.GetProperties())

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
					LogRegex:   logsource.GetLogRegexFromInfo(fnCfg.fset.File(expr.Pos()).Name(), line),
				})
			} else {
				// Was a method call.
				node = db.Node(&db.FunctionNode{
					Filename:     filepath.ToSlash(relPath),
					LineNumber:   fnCfg.fset.Position(expr.Pos()).Line,
					FunctionName: fnCfg.expressionString(fn),
					Args:         fnCfg.getFuncArgs(expr.Args), //TODO: generate Args for functionNode
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
				Args:         fnCfg.getFuncParams(fn.Type.Params, relPath), //TODO: generate Args for functionNode
			}
		default:
			// fmt.Println(callExprName(expr))
			// Found a function call
			node = db.Node(&db.FunctionNode{
				Filename:     filepath.ToSlash(relPath),
				LineNumber:   fnCfg.fset.Position(expr.Pos()).Line,
				FunctionName: fnCfg.callExprName(expr),
				Args:         fnCfg.getFuncArgs(expr.Args), //TODO: generate Args for functionNode
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
					Params:       fnCfg.getFuncParams(expr.Type.Params, relPath),
					Receivers:    map[string]string{},
					Returns:      fnCfg.getFuncReturns(expr.Type.Results),
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
				Condition:  fnCfg.expressionString(expr),
				VarsUsed:   fnCfg.extractUsedVars(expr),
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
				Condition:  fnCfg.expressionString(expr),
				VarsUsed:   fnCfg.extractUsedVars(expr),
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
			node = &db.ConditionalNode{
				Filename:   filepath.ToSlash(relPath),
				LineNumber: fnCfg.fset.Position(expr.Pos()).Line,
				Condition:  fnCfg.expressionString(expr),
				VarsUsed:   fnCfg.extractUsedVars(expr),
			}
		} else {
			//TODO: could be a variable?
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
func (fnCfg *FnCfgCreator) iterateFields(fieldList *ast.FieldList, op func(returnType, name string)) {
	if fieldList != nil {
		for _, p := range fieldList.List {
			if p != nil {
				returnType := fnCfg.expressionString(p.Type)
				for _, name := range p.Names {
					varName := fnCfg.expressionString(name)
					op(returnType, varName)
				}
			}
		}
	}
}

func (fnCfg *FnCfgCreator) getFuncReceivers(fieldList *ast.FieldList) map[string]string {
	receivers := make(map[string]string)

	fnCfg.iterateFields(fieldList, func(returnType, name string) {
		receivers[name] = returnType
	})

	return receivers
}

func (fnCfg *FnCfgCreator) getFuncParams(fieldList *ast.FieldList, file string) []db.VariableNode {
	params := make([]db.VariableNode, len(fieldList.List))

	if fieldList != nil {
		for i, p := range fieldList.List {
			if p != nil {
				for _, name := range p.Names {
					variable := db.VariableNode{
						Filename: file,
						VarName:  fnCfg.expressionString(name),
						ScopeId:  fnCfg.getCurrentScope(),
					}
					params[i] = variable
				}
			}
		}
	}

	return params
}

func (fnCfg *FnCfgCreator) getFuncArgs(exprs []ast.Expr) []db.VariableNode {
	args := make([]db.VariableNode, len(exprs))
	if exprs != nil {
		for i, exp := range exprs {
			if exp, ok := exp.(*ast.Ident); ok && exp != nil {
				//fmt.Printf("%T %v\n", exp, exp)
				//TODO: construct VariableNode from Expr
				args[i] = db.VariableNode{
					Filename:        fnCfg.fset.File(exp.Pos()).Name(),
					LineNumber:      fnCfg.fset.Position(exp.Pos()).Line,
					ScopeId:         fnCfg.getCurrentScope(),
					VarName:         exp.Name,
					Value:           "",
					Parent:          nil,
					Child:           nil,
					ValueFromParent: false,
				}
			}
		}
	}
	return args
}

func (fnCfg *FnCfgCreator) getFuncReturns(fieldList *ast.FieldList) []db.Return {
	returns := make([]db.Return, 0)

	fnCfg.iterateFields(fieldList, func(returnType, name string) {
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
		receivers := fnCfg.getFuncReceivers(stmt.Recv)
		var params []db.VariableNode
		var returns []db.Return
		if stmt.Type != nil {
			params = fnCfg.getFuncParams(stmt.Type.Params, relPath)
			if stmt.Type.Results != nil {
				returns = fnCfg.getFuncReturns(stmt.Type.Results)
			}
		} else {
			params = make([]db.VariableNode, 0)
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
	case *ast.AssignStmt: //Handles variables when being assigned
		node, _, _ = fnCfg.chainExprNodes(stmt.Rhs)

		var exprValue string = "" //hold the expression as a string
		var varName string = ""
		var isFromFunction bool = false
		var isReal bool = true

		//Process left side variable name
		for _, lhsExpr := range stmt.Lhs {
			switch expr := lhsExpr.(type) {
			case *ast.SelectorExpr:
				if expr.Sel.Name != "" {
					varName = expr.Sel.Name
				}
			}
		}

		//Set variable name if it is still empty
		if varName == "" {
			strLHS := fmt.Sprint(stmt.Lhs)
			varName = strLHS[strings.Index(strLHS, "[")+1 : strings.Index(strLHS, "]")]
		}

		//Get the expression operator
		exprOp := stmt.Tok.String() //assignment operator

		//Checks if rhs if a variable gets a value from a function or literal
		for _, rhsExpr := range stmt.Rhs {
			switch expr := rhsExpr.(type) {
			//Basic literals indicate the var shouldn't have been returned from a function and a real value
			case *ast.BasicLit:
				if exprOp == ":=" {
					exprValue = varName + " " + exprOp + " " + expr.Value
				} else if exprOp == "=" {
					exprValue = "var " + varName + " " + expr.Kind.String() + " " + exprOp + " " + expr.Value
				}
				isFromFunction = false
				isReal = true

			case *ast.CompositeLit: //Indicates a variable being assigned a struct/slice/array (real value?)
				//fmt.Println("Is composite literal", expr.Type)
				if expr.Incomplete {
					fmt.Println("Source expressions missing in elt list")
				}

				//Grabbing the struct/slice assignment from the composite literal
				litPos := expr.Type.Pos()
				tempFile := fnCfg.fset.Position(litPos).Filename
				lineNum := fnCfg.fset.Position(litPos).Line
				file, err := os.Open(tempFile)
				if err != nil {
					fmt.Println("Error opening file")
				}

				//Read file at specific line to get function name
				cnt := 1
				var rightValue string = ""
				scanner := bufio.NewScanner(file)
				for scanner.Scan() {
					if cnt == lineNum {
						rightValue = scanner.Text()
						break
					}
					cnt++
				}

				//Get the right side value assignment
				rightValue = rightValue[strings.Index(rightValue, "=")+2 : strings.Index(rightValue, "{")]

				isFromFunction = false
				isReal = false
				exprValue = varName + " " + exprOp + " " + rightValue

			default: //If it isn't a literal, it will be a symbolic value (from variable or from function)
				litPos := expr.Pos()
				tempFile := fnCfg.fset.Position(litPos).Filename
				lineNum := fnCfg.fset.Position(litPos).Line
				file, err := os.Open(tempFile)
				if err != nil {
					fmt.Println("Error opening file")
				}

				//Read file at specific line to get function name
				cnt := 1
				var rightValue string = ""
				scanner := bufio.NewScanner(file)
				for scanner.Scan() {
					if cnt == lineNum {
						rightValue = scanner.Text()
						break
					}
					cnt++
				}

				//Sets the variable expression
				isFromFunction, exprValue = isFunctionAssignment(rightValue)
				isReal = false
			}
		}

		//Print the final expression
		if exprValue != "" {
			//fmt.Printf("Var expr: (%v)\n --fromFunction: %v\n --realValue: %v\n", exprValue, isFromFunction, isReal)
		}

		//Handling variable scoping at assign time
		//var scopeID string = ""
		//fmt.Printf("(%s %s %s)\n", varName, strExpr, assignValue)
		scopeID := ""
		if value, ok := fnCfg.varNameToStack[varName]; ok && len(value) > 0 {
			fmt.Printf("scope:'%s'\n", scopeID)
			scopeID = value[len(value)-1]
		} else {
			//handle adding scope if variable not in map at assign time
			//fmt.Printf("record scope:'%s'\n", fnCfg.getCurrentScope())
			fnCfg.varNameToStack[varName] = append(fnCfg.varNameToStack[varName], fnCfg.getCurrentScope())
			scopeID = fnCfg.varNameToStack[varName][len(fnCfg.varNameToStack[varName])-1]
		}

		//Add the scope ID to the variable node

		var separator string
		if runtime.GOOS == "windows" {
			separator = "\\"
		} else {
			separator = "/"
		}
		longFile := fnCfg.fset.File(stmt.Pos()).Name()
		file := longFile[strings.LastIndex(longFile, separator)+1:]

		//Build variable node
		// TODO: Would be easiest to make a chain of variable nodes here, but not sure if child nodes will be overwritten later
		varNode := &db.VariableNode{
			Filename:        file,
			LineNumber:      fnCfg.fset.Position(stmt.Pos()).Line,
			ScopeId:         scopeID,
			VarName:         varName,
			Value:           exprValue, //the expression (ex: x := 5, temp := foo())
			Parent:          nil,
			Child:           nil,
			ValueFromParent: isFromFunction,
			IsReal:          isReal,
		}

		//Check if a variable contains two variables
		//if strings.Contains(exprValue, ",") && strings.Contains(exprValue, ":="){
		//
		//}

		//Add to list of variables
		fnCfg.varList[scopeID+":"+varName] = varNode

		//TODO: Might need to chain vars here?
		if node != nil {
			// Append the variable node to the last function call
			connectToLeaf(node, varNode)
		} else {
			node = varNode
		}

	case *ast.ReturnStmt:
		// Find all function calls contained in the return statement
		node, _, _ = fnCfg.chainExprNodes(stmt.Results)

		var bldr strings.Builder
		for i, result := range stmt.Results {
			bldr.WriteString(fmt.Sprintf("%s", fnCfg.expressionString(result)))
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

//Helper function to determine if a variable gets its value from a function
// May need to handle more cases later, but it works well for now
func isFunctionAssignment(str string) (bool, string) {
	isFunction := false

	var compStr string = str
	var endIndex int = 0
	if strings.Contains(compStr, ";") {
		endIndex = strings.Index(str, ";")
	}

	//slice out just the assignment part
	if strings.Contains(compStr, "if") && strings.Contains(compStr, ";") {
		compStr = str[strings.Index(compStr, "if")+3 : endIndex]
		//fmt.Println("If strings", compStr)
	}
	if strings.Contains(compStr, "for") && strings.Contains(compStr, ";") {
		compStr = str[strings.Index(compStr, "for")+4 : endIndex]
		//fmt.Println("For strings", compStr)
	}

	//If it contains a set of parenthesis, most likely a function
	if strings.Contains(str, "(") && strings.Contains(str, ")") {
		if strings.Contains(str, ".") {
			isFunction = true
		}
	}

	return isFunction, compStr
}

// Recursively creates the string of an `ast.Expr`.
func (fnCfg *FnCfgCreator) expressionString(expr ast.Expr) string {
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
		leftStr = fnCfg.expressionString(condition.X)
		rightStr = fnCfg.expressionString(condition.Y)
		return fmt.Sprint(leftStr, condition.Op, rightStr)
	case *ast.UnaryExpr:
		op := condition.Op.String()
		str := fnCfg.expressionString(condition.X)
		return fmt.Sprint(op, str)
	case *ast.SelectorExpr:
		selector := ""
		if condition.Sel != nil {
			selector = condition.Sel.String()
		}
		str := fnCfg.expressionString(condition.X)
		return fmt.Sprintf("%s.%s", str, selector)
	case *ast.ParenExpr:
		return fmt.Sprintf("(%s)", fnCfg.expressionString(condition.X))
	case *ast.CallExpr:
		fn := fnCfg.expressionString(condition.Fun)
		args := make([]string, 0)
		for _, arg := range condition.Args {
			args = append(args, fnCfg.expressionString(arg))
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
		expr := fnCfg.expressionString(condition.X)
		ndx := fnCfg.expressionString(condition.Index)
		return fmt.Sprintf("%s[%s]", expr, ndx)
	case *ast.KeyValueExpr:
		key := fnCfg.expressionString(condition.Key)
		value := fnCfg.expressionString(condition.Value)
		return fmt.Sprint(key, ":", value)
	case *ast.SliceExpr: // not sure about this one
		expr := fnCfg.expressionString(condition.X)
		low := fnCfg.expressionString(condition.Low)
		high := fnCfg.expressionString(condition.High)
		if condition.Slice3 {
			max := fnCfg.expressionString(condition.Max)
			return fmt.Sprintf("%s[%s : %s : %s]", expr, low, high, max)
		}
		return fmt.Sprintf("%s[%s : %s]", expr, low, high)
	case *ast.StarExpr:
		expr := fnCfg.expressionString(condition.X)
		return fmt.Sprintf("*%s", expr)
	case *ast.TypeAssertExpr:
		expr := fnCfg.expressionString(condition.X)
		typecast := fnCfg.expressionString(condition.Type)
		return fmt.Sprintf("%s(%s)", typecast, expr)
	case *ast.FuncType:
		params := fnCfg.getFuncParams(condition.Params, "")
		rets := fnCfg.getFuncReturns(condition.Results)
		b := strings.Builder{}
		b.Write([]byte("func("))
		i := 0
		for _, param := range params {
			b.Write([]byte(fmt.Sprintf("%s", param.VarName)))
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
func (fnCfg *FnCfgCreator) callExprName(call *ast.CallExpr) string {
	fn := fnCfg.expressionString(call)
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
			VarsUsed:   node.VarsUsed,
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
func (fnCfg *FnCfgCreator) extractUsedVars(expr ast.Expr) []*db.VariableNode {
	return fnCfg.extractUsedVarsRecur(expr, []*db.VariableNode{})
}

func (fnCfg *FnCfgCreator) latestDeclarationOfVar(name string) *db.VariableNode {
	if stack, ok := fnCfg.varNameToStack[name]; ok && len(stack) > 0 {
		topDeclaredScope := stack[len(stack)-1]
		if v, ok := fnCfg.varList[topDeclaredScope+":"+name]; ok {
			return v
		}
	}
	return nil
}

func (fnCfg *FnCfgCreator) extractUsedVarsRecur(expr ast.Expr, vars []*db.VariableNode) []*db.VariableNode {
	if expr == nil {
		return vars
	}

	//Return expression based on the type
	switch condition := expr.(type) {
	case *ast.Ident:
		vars = append(vars, fnCfg.latestDeclarationOfVar(condition.Name))
	case *ast.BinaryExpr:
		vars = fnCfg.extractUsedVarsRecur(condition.X, vars)
		vars = fnCfg.extractUsedVarsRecur(condition.Y, vars)
	case *ast.UnaryExpr:
		vars = fnCfg.extractUsedVarsRecur(condition.X, vars)
	case *ast.SelectorExpr:
		vars = fnCfg.extractUsedVarsRecur(condition.X, vars)
	case *ast.ParenExpr:
		vars = fnCfg.extractUsedVarsRecur(condition.X, vars)
	case *ast.CallExpr:
		for _, arg := range condition.Args {
			vars = fnCfg.extractUsedVarsRecur(arg, vars)
		}
	case *ast.IndexExpr:
		vars = fnCfg.extractUsedVarsRecur(condition.X, vars)
		vars = fnCfg.extractUsedVarsRecur(condition.Index, vars)
	case *ast.KeyValueExpr:
		vars = fnCfg.extractUsedVarsRecur(condition.Key, vars)
		vars = fnCfg.extractUsedVarsRecur(condition.Value, vars)
	case *ast.SliceExpr: // not sure about this one
		vars = fnCfg.extractUsedVarsRecur(condition.X, vars)
		vars = fnCfg.extractUsedVarsRecur(condition.Low, vars)
		vars = fnCfg.extractUsedVarsRecur(condition.High, vars)
		if condition.Slice3 {
			vars = fnCfg.extractUsedVarsRecur(condition.Max, vars)
		}
	case *ast.StarExpr:
		vars = fnCfg.extractUsedVarsRecur(condition.X, vars)
	case *ast.TypeAssertExpr:
		vars = fnCfg.extractUsedVarsRecur(condition.X, vars)
		// vars = fnCfg.extractConditionalVarsRecur(condition.Type, vars)
	case *ast.FuncType:
		if condition.Params == nil {
			break
		}
		for _, param := range condition.Params.List {
			for _, name := range param.Names {
				vars = fnCfg.extractUsedVarsRecur(name, vars)
			}
		}
	}
	return vars
}

func processConditional(node *db.ConditionalNode) (string, []*db.VariableNode) {
	if node != nil {
		return node.Condition, node.VarsUsed
	}
	return "", []*db.VariableNode{}
}
