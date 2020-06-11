package handler

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"sourcecrawler/app/db"
	"sourcecrawler/app/model"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"golang.org/x/tools/go/cfg"
)

func createTestNeoNodes() {
	node7 := db.StatementNode{"test.go", 7, "", nil}
	node6 := db.StatementNode{"test.go", 6, "another log regex", &node7}
	node5 := db.StatementNode{"test.go", 5, "", &node6}
	node4 := db.StatementNode{"test.go", 4, "my log regex", &node6}
	node3 := db.ConditionalNode{"test.go", 3, "myvar != nil", &node4, &node5}
	node2 := db.StatementNode{"test.go", 2, "", &node3}
	node1 := db.StatementNode{"test.go", 1, "", &node2}

	dao := db.NodeDaoNeoImpl{}
	dao.CreateTree(&node1)
}

type varDecls struct {
	asns  []*ast.AssignStmt
	decls []*ast.ValueSpec
}

//find index of a logtype for value changing
func indexOf(elt model.LogType, arr []model.LogType) (int, bool) {
	for k, v := range arr {
		if elt == v {
			return k, true
		}
	}
	return -1, false
}

//Parse project to create log types
func parseProject(projectRoot string) []model.LogType {

	//Holds a slice of log types
	logTypes := []model.LogType{}
	variableDeclarations := varDecls{}
	variablesUsedInLogs := map[string]struct{}{}
	filesToParse := []string{}
	//gather all go files in project
	filepath.Walk(projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("prevent panic by handling failure accessing a path %q: %v\n", path, err)
			return err
		}
		if filepath.Ext(path) == ".go" {
			fullpath, err := filepath.Abs(path)
			if err != nil {
				fmt.Println(err)
			}
			filesToParse = append(filesToParse, fullpath)
		}
		return nil
	})

	//parse each file to collect logs and the variables used in them
	//as well as collecting variables declared in the file for later use
	for _, file := range filesToParse {
		newLogTypes, newVariablesUsedInLogs := findLogsInFile(file, projectRoot)
		logTypes = append(logTypes, newLogTypes...)
		for key := range newVariablesUsedInLogs {
			variablesUsedInLogs[key] = struct{}{}
		}

		newVarDecls := findVariablesInFile(file)
		variableDeclarations.asns = append(variableDeclarations.asns, newVarDecls.asns...)
		variableDeclarations.decls = append(variableDeclarations.decls, newVarDecls.decls...)
	}

	//match direct references to variables and update regex values
	//note: may require checking scope if duplicate names are used
	for _, asn := range variableDeclarations.asns {
		fnName := fmt.Sprint(asn.Lhs[0])
		_, ok := variablesUsedInLogs[fnName]
		if ok {
			//find variable in logTypes, update value
			for _, logType := range logTypes {
				if logType.Regex == fnName {
					ndx, ok := indexOf(logType, logTypes)
					if ok {
						//This could be enhanced to detect non-literal values
						//such as function returns, but currently just accept
						//literal values
						newVal, ok := asn.Rhs[0].(*ast.BasicLit)
						if ok && len(newVal.Value) > 1 {
							logTypes[ndx].Regex = newVal.Value[1 : len(newVal.Value)-1]
						}
					}
				}
			}
		}
	}
	for _, decl := range variableDeclarations.decls {
		fnName := fmt.Sprint(decl.Names[0])
		_, ok := variablesUsedInLogs[fnName]
		if ok {
			//find variable in logTypes, update value
			for _, logType := range logTypes {
				if logType.Regex == fnName {
					ndx, ok := indexOf(logType, logTypes)
					if ok {
						if len(decl.Values) > 0 {
							//This could be enhanced to detect non-literal values
							//such as function returns, but currently just accept
							//literal values
							newVal, ok := decl.Values[0].(*ast.BasicLit)
							if ok && len(newVal.Value) > 1 {
								logTypes[ndx].Regex = newVal.Value[1 : len(newVal.Value)-1]
							}
						}
					}
				}
			}
		}
	}

	// Doesn't do anything right now
	// functList := functionDecls(filesToParse) //gathers list of functions
	// callFrom(functList, filesToParse) //checks each expression call to see if it uses an explicitly declared function

	return logTypes
}

//This is just a struct
//to quickly access members of the nodes.
//Can be changed/removed if desired,
//currently just a placeholder
type fnStruct struct {
	n              ast.Node
	fn             *ast.CallExpr
	parentFn       *ast.FuncDecl
	usedParentArgs []*ast.Ident
}

//Checks if from log (two.name is Info/Err/Error)
func isFromLog(fn *ast.SelectorExpr) bool {
	if strings.Contains(fmt.Sprint(fn.X), "log") {
		return true
	}
	one, ok := fn.X.(*ast.CallExpr)
	if ok {
		two, ok := one.Fun.(*ast.SelectorExpr)
		if ok {
			return isFromLog(two)
		}
	}
	return false
}

/*
Current idea:
	for each function declaration
		if it contains a logging statement
			if the logging statement's args using 1+ of the functions args
				store it with an associated parent function
			else
				store it without an associated parent function

	for each logging statement using parent function argument(s)
		for each callexpr of that function
			store with its message information
*/

func usesParentArgs(parent *ast.FuncDecl, call *ast.CallExpr) []*ast.Ident {
	fmt.Println("checking if", call, "depends on", parent.Name)
	args := make([]*ast.Ident, 0)
	if call == nil || parent == nil || parent.Type == nil || parent.Type.Params == nil {
		return args
	}
	if parent.Type.Params.List == nil || len(parent.Type.Params.List) == 0 {
		return args
	}

	// Gather all parent parameter names
	params := make([]string, 0)
	for _, field := range parent.Type.Params.List {
		if field == nil {
			continue
		}
		for _, param := range field.Names {
			params = append(params, param.Name)
		}
	}

	// Check if any call arguments are from the parent function
	for _, arg := range call.Args {
		switch arg := arg.(type) {
		case *ast.Ident:
			if arg == nil || arg.Obj == nil {
				continue
			}
			for _, param := range params {
				if arg.Name == param {
					if arg.Obj.Kind == ast.Var || arg.Obj.Kind == ast.Con {
						// Found an argument used by the parent in the logging call expression
						// or a constant we can find the value of
						fmt.Println("\tcall expression uses", param)
						args = append(args, arg)
					}
				}
			}
		case *ast.CallExpr:
			// Check for nested function calls that use the parent's argument
			// Still need to keep track of all the nested functions somehow
			if arg != nil {
				args = append(args, usesParentArgs(parent, arg)...)
			}
		}
	}

	return args
}

func findVariablesInFile(path string) varDecls {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		log.Error().Err(err).Msg("unable to parse file")
	}
	vars := varDecls{}

	ast.Inspect(node, func(n ast.Node) bool {
		//The following two blocks are related to finding variables
		//and their values

		//filter nodes that represent variable asns and decls

		//These follow pattern "name :=/= value"
		asn, ok := n.(*ast.AssignStmt)
		if ok {
			vars.asns = append(vars.asns, asn)
		}

		//These nodes follow pattern "var/const name = value"
		expr, ok := n.(*ast.GenDecl)
		if ok {
			spec, ok := expr.Specs[0].(*ast.ValueSpec)
			if ok {
				vars.decls = append(vars.decls, spec)
			}
		}
		return true
	})

	return vars
}

func findLogsInFile(path string, base string) ([]model.LogType, map[string]struct{}) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		log.Error().Err(err).Msg("unable to parse file")
	}

	varsInLogs := map[string]struct{}{}
	logInfo := []model.LogType{}
	logCalls := []fnStruct{}

	//Helper structure to hold logTypes with function types (msg, msgf, err)
	//lnTypes := make(map[model.LogType]string)

	//Filter out nodes that do not contain a call to Msg or Msgf
	//then call the recursive function isFromLog to determine
	//if these Msg* calls originated from a log statement to eliminate
	//false positives
	var parentFn *ast.FuncDecl
	ast.Inspect(node, func(n ast.Node) bool {
		// Keep track of the current parent function the log statement is contained in
		if funcDecl, ok := n.(*ast.FuncDecl); ok {
			fmt.Println("checking funcDecl", funcDecl.Name)
			_ = createFnCfg(funcDecl, base, fset)
			parentFn = funcDecl
		}

		//The following block is for finding log statements and the
		//values passed to them as args

		//continue if Node casts as a CallExpr
		if ret, ok := n.(*ast.CallExpr); ok {
			//continue processing if CallExpr casts
			//as a SelectorExpr
			if fn, ok := ret.Fun.(*ast.SelectorExpr); ok {
				// fmt.Printf("%T, %v\n", fn, fn)
				//convert Selector into String for comparison
				val := fmt.Sprint(fn.Sel)

				//fmt.Println("Val: " + val)

				//Should recursively call a function to check if
				//the preceding SelectorExpressions contain a call
				//to log, which means this is most
				//definitely a log statement
				if (strings.Contains(val, "Msg") || val == "Err") && isFromLog(fn) {
					parentArgs := usesParentArgs(parentFn, ret)
					value := fnStruct{
						n:              n,
						fn:             ret,
						parentFn:       nil,
						usedParentArgs: parentArgs,
					}
					// Check if the log call depends on a parent function argument
					// and if it does, specify the parent function
					if len(parentArgs) > 0 {
						value.parentFn = parentFn
					}
					logCalls = append(logCalls, value)
				}
			}
		}
		return true
	})

	//AT THIS POINT
	//all log messages should be collected, the next section
	//demonstrates extracting the string arguments from each
	//log isntance

	//This block is for accessing the argument
	//values of a selector function so we can see
	//what was added by the inspection above
	for _, l := range logCalls {
		currentLog := model.LogType{}
		// fn, _ := l.fn.Fun.(*ast.SelectorExpr)
		// fmt.Printf("Args for %v at line %d\n", fn.Sel, fset.Position(l.n.Pos()).Line)

		relPath, _ := filepath.Rel(base, fset.File(l.n.Pos()).Name())
		currentLog.FilePath = filepath.ToSlash(relPath)
		currentLog.LineNumber = fset.Position(l.n.Pos()).Line
		for _, a := range l.fn.Args {
			good := false
			//later will be used to call functions
			//to extract data more eficiently for multiple
			//types of arguments
			switch v := a.(type) {

			//this case catches string literals,
			//our proof-of-concept case
			case *ast.BasicLit:
				good = true
				// fmt.Println("Basic", v.Value)

				currentLog.Regex = createRegex(v.Value)

				logInfo = append(logInfo, currentLog)

			//this case catches composite literals
			case *ast.CompositeLit:
				// fmt.Println("Composite", v.Elts)

			//This case represents variables used as log arguments
			case *ast.Ident:
				//store var name, will be   updated later
				currentLog.Regex = v.Name
				// fmt.Printf("%v, ", v.Name)

				//add an entry in the map for the variable name
				//so we can check if declarations refer to a
				//variable used in a log statement
				varsInLogs[v.Name] = struct{}{}
				good = true

			default:
				fmt.Println("type arg", reflect.TypeOf(a), a)
			}
			//if the type is known and handled,
			//add it to the result array
			if good {
				logInfo = append(logInfo, currentLog)
			}
		}
		fmt.Println()
	}

	return logInfo, varsInLogs
}

func connectNodes(caller, callee db.FunctionNode) {

	/*
		//query for getting nodes from db
		MATCH (a:Node), (b:Node)
		WHERE a.function = b.function
		AND a.filename = $callerFile AND b.filename = $calleeFile
		AND a.line = $callerLine AND b.line = $calleeLine

		//and adding relationship to connect the two graphs
		CREATE e = (a)-[r:CALLS]->(b)
		RETURN e,
		map[string]interface{}{"callerFile": caller.Filename, "calleeFile": callee.Filename,
		"callerLine": caller.LineNumber, "calleeLine": callee.LineNumber}
	*/
}

// Functions for finding calls across files

/*
 Finds location of all function declarations (path and line number)
  -currently goes through all files and finds if it's used
  Returns map from filename -> [linenumber, filepath] (all strings)
*/
func functionDecls(filesToParse []string) map[string][]string {

	//Map of all function names with a [line number, file path]
	// ex: ["HandleMessage" : {"45":"insights-results-aggregator/consumer/processing.go"}]
	//They key is the function name. Go doesn't support function overloading -> so each name will be unique
	functMap := map[string][]string{}

	//Inspect each file for calls to this function
	for _, file := range filesToParse {
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			log.Error().Err(err).Msg("Error parsing file " + file)
		}

		//Grab package name - needed to prevent duplicate function names across different packages, keep colon
		packageName := node.Name.Name + ":"
		fmt.Print("Package name is " + packageName + " at line ")
		fmt.Println(fset.Position(node.Pos()).Line)

		//Inspect AST for file
		//TODO: handle duplicate function names if they're in different packages
		ast.Inspect(node, func(currNode ast.Node) bool {

			fdNode, ok := currNode.(*ast.FuncDecl)
			if ok {
				//package name is appended to separate diff functions across packages
				functionName := packageName + fdNode.Name.Name
				linePos := strconv.Itoa(fset.Position(fdNode.Pos()).Line)
				fpath, _ := filepath.Abs(fset.File(fdNode.Pos()).Name())

				//Add the data to the function list
				data := []string{linePos, fpath}
				functMap[functionName] = data
			}

			return true
		})
	}

	return functMap
}

/*
	Check the location (file + line number) of where each function is used
  TODO: create return value
*/
func callFrom(funcList map[string][]string, filesToParse []string) {

	for _, file := range filesToParse {
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			log.Error().Err(err).Msg("Error parsing file " + file)
		}

		//Keep track of package name
		packageName := node.Name.Name + ":"
		//fmt.Print("Package name is " + packageName + " at line ")
		//fmt.Println(fset.Position(node.Pos()).Line)

		//Inspect the AST, starting with call expressions
		ast.Inspect(node, func(currNode ast.Node) bool {
			callExprNode, ok := currNode.(*ast.CallExpr)
			if ok {
				//Filter single function calls such as parseMsg(msg.Value)
				functionName := packageName + fmt.Sprint(callExprNode.Fun)
				if val, found := funcList[functionName]; found {
					fmt.Println("The function " + functionName + " was found on line " + val[0] + " in " + val[1])
				}
			}
			return true
		})
	}
}

func createRegex(value string) string {
	//Regex value currently
	reg := value

	//Converting current regex strings to regex format (parenthesis, %d,%s,%v,',%+v)
	if strings.Contains(reg, "(") {
		reg = strings.ReplaceAll(reg, "(", "\\(")
	}
	if strings.Contains(reg, ")") {
		reg = strings.ReplaceAll(reg, ")", "\\)")
	}

	//Converting %d, %s, %v to regex num, removing single quotes
	if strings.Contains(reg, "%d") {
		reg = strings.ReplaceAll(reg, "%d", "\\d")
	}
	if strings.Contains(reg, "%s") {
		reg = strings.ReplaceAll(reg, "%s", ".*")
	}
	if strings.Contains(reg, "%v") {
		reg = strings.ReplaceAll(reg, "%v", ".*")
	}
	if strings.Contains(reg, "'") {
		reg = strings.ReplaceAll(reg, "'", "")
	}
	if strings.Contains(reg, "%+v") {
		reg = strings.ReplaceAll(reg, "%+v", ".+")
	}

	//Remove the double quotes
	return reg[1 : len(reg)-1]
}

func createFnCfg(fn *ast.FuncDecl, base string, fset *token.FileSet) db.Node {
	if fn == nil {
		fmt.Println("\tfn was nil")
		return nil
	}
	if fn.Body == nil {
		fmt.Println("\tbody was nil")
		return nil
	}
	if fn.Name == nil {
		fmt.Println("\tname was nil")
		return nil
	}

	root := getStatementNode(fn, base, fset)

	cfg := cfg.New(fn.Body, func(call *ast.CallExpr) bool {
		if call != nil {
			if fn.Name.Name != "Exit" && !strings.Contains(fn.Name.Name, "Fatal") && fn.Name.Name != "panic" {
				return true
			}
		}
		return false
	})
	fmt.Println(cfg.Format(fset))

	if len(cfg.Blocks) < 1 {
		return root
	}

	block := cfg.Blocks[0]
	node := constructSubCfg(block, base, fset)
	if node == nil {
		return root
	}

	if fn, ok := root.(*db.FunctionNode); ok {
		fn.Child = node
	}

	printCfg(root, "")
	fmt.Println()

	return root
}

func printCfg(node db.Node, level string) {
	if node == nil {
		return
	}
	switch node := node.(type) {
	case *db.FunctionNode:
		fmt.Printf("%s%s\n", level, node.FunctionName)
		printCfg(node.Child, level)
	case *db.StatementNode:
		fmt.Printf("%s%s\n", level, node.LogRegex)
		printCfg(node.Child, level)
	case *db.ConditionalNode:
		fmt.Printf("%sif %s\n", level, node.Condition)
		printCfg(node.TrueChild, level+"  ")
		fmt.Println(level + "else")
		printCfg(node.FalseChild, level+"  ")
	}
}

func constructSubCfg(block *cfg.Block, base string, fset *token.FileSet) (root db.Node) {
	if block == nil || block.Nodes == nil {
		return nil
	}

	conditional := false
	var prev db.Node
	var current db.Node
	for i, node := range block.Nodes {
		last := i == len(block.Nodes)-1
		conditional = len(block.Succs) > 1

		switch node := node.(type) {
		case ast.Stmt:
			current = getStatementNode(node, base, fset)
		case ast.Expr:
			current = getExprNode(node, base, fset, last && conditional)
		}
		if current == nil {
			fmt.Println("\treturn node was nil")
			continue
		}
		if root == nil {
			root = current
		}
		if prev == nil {
			prev = current
		}
		if current != nil {
			switch prevNode := prev.(type) {
			case *db.FunctionNode:
				// may need to fast-forward to deepest child node here
				// if there was a statement like _, _ = func1(), func2()
				prevNode.Child = current
				if call, ok := current.(*db.FunctionNode); ok {
					for call != nil {
						if child, ok := call.Child.(*db.FunctionNode); ok && child != nil {
							if child == call {
								call.Child = nil
								if child.Child == call {
									child.Child = nil
								}
							}
							prev = call
							current = child
							call = child
						} else {
							call = nil
						}
					}
				}
			case *db.StatementNode:
				// You should never encounter a "previous" conditional inside of a block since
				// the conditional is always the last node in a CFG block if a conditional is present
				// case *db.ConditionalNode:
				prevNode.Child = current
			}
			prev = current
		}

		if expr, ok := node.(ast.Expr); ok && last && conditional {
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
			conditional.TrueChild = constructSubCfg(block.Succs[0], base, fset)
			conditional.FalseChild = constructSubCfg(block.Succs[1], base, fset)

			switch node := current.(type) {
			case *db.FunctionNode:
				node.Child = db.Node(conditional)
			case *db.StatementNode:
				node.Child = db.Node(conditional)
			}
		} else if len(block.Succs) == 1 {
			subCfg := constructSubCfg(block.Succs[0], base, fset)
			switch node := current.(type) {
			case *db.FunctionNode:
				node.Child = subCfg
			case *db.StatementNode:
				node.Child = subCfg
			}
		}

		current = nil
	}

	return
}

func getExprNode(expr ast.Expr, base string, fset *token.FileSet, conditional bool) (node db.Node) {
	relPath, _ := filepath.Rel(base, fset.File(expr.Pos()).Name())
	switch expr := expr.(type) {
	case *ast.CallExpr:
		fmt.Print("\t\tfound a callexpr ")
		if selectStmt, ok := expr.Fun.(*ast.SelectorExpr); ok {
			val := fmt.Sprint(selectStmt.Sel)
			fmt.Println(val)
			if (strings.Contains(val, "Msg") || strings.Contains(val, "Err")) && isFromLog(selectStmt) {
				regex := ""
				// use createRegex() and somehow rectify the API to reuse the latter half of findLogsInFile fn here
				node = db.Node(&db.StatementNode{
					Filename:   filepath.ToSlash(relPath),
					LineNumber: fset.Position(expr.Pos()).Line,
					LogRegex:   regex,
				})
			} else {
				node = db.Node(&db.FunctionNode{
					Filename:     filepath.ToSlash(relPath),
					LineNumber:   fset.Position(expr.Pos()).Line,
					FunctionName: expressionString(selectStmt),
				})
			}
		} else {
			fmt.Println(callExprName(expr))
			node = db.Node(&db.FunctionNode{
				Filename:     filepath.ToSlash(relPath),
				LineNumber:   fset.Position(expr.Pos()).Line,
				FunctionName: callExprName(expr),
			})
		}
	default:
		if conditional {
			fmt.Println("\t\tfound a condition")
			node = db.Node(&db.ConditionalNode{
				Filename:   filepath.ToSlash(relPath),
				LineNumber: fset.Position(expr.Pos()).Line,
				Condition:  expressionString(expr),
			})
		}
	}
	return
}

func getStatementNode(stmt ast.Node, base string, fset *token.FileSet) (node db.Node) {
	switch stmt := stmt.(type) {
	case *ast.ExprStmt:
		node = getExprNode(stmt.X, base, fset, false)
	case *ast.FuncDecl:
		relPath, _ := filepath.Rel(base, fset.File(stmt.Pos()).Name())
		node = db.Node(&db.FunctionNode{
			Filename:     filepath.ToSlash(relPath),
			LineNumber:   fset.Position(stmt.Pos()).Line,
			FunctionName: stmt.Name.Name,
		})
	case *ast.AssignStmt:
		fmt.Println("\t\tfound assignstmt")
		var first db.Node
		var current db.Node
		var prev db.Node
		for _, expr := range stmt.Rhs {
			exprNode := getExprNode(expr, base, fset, false)
			if prev == nil && current != nil {
				prev = current
			}

			switch exprNode := exprNode.(type) {
			case *db.FunctionNode:
				current = exprNode
			case *db.StatementNode:
				current = exprNode
			}

			if node, ok := prev.(*db.FunctionNode); ok {
				node.Child = current
			}

			if first == nil {
				first = current
			}
		}
		node = first
	default:
		fmt.Println("\t\tdid not cast")
	}
	return
}

func expressionString(expr ast.Expr) string {
	if expr == nil {
		return ""
	}
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
		// may want to return *db.StatementNode for CallExpr I find these to add to the CFG
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

func callExprName(call *ast.CallExpr) string {
	fn := expressionString(call)
	name := ""
	if s := strings.Split(fn, "("); len(s) > 0 {
		name = s[0]
	}
	return name
}
