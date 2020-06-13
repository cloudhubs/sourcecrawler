package handler

import (
	"bufio"
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
	// node7 := db.StatementNode{"test.go", 7, "", nil}
	// node6 := db.StatementNode{"test.go", 6, "another log regex", &node7}
	// node5 := db.StatementNode{"test.go", 5, "", &node6}
	// node4 := db.StatementNode{"test.go", 4, "my log regex", &node6}
	// node3 := db.ConditionalNode{"test.go", 3, "myvar != nil", &node4, &node5}
	// node2 := db.StatementNode{"test.go", 2, "", &node3}
	// node1 := db.StatementNode{"test.go", 1, "", &node2}

	dao := db.NodeDaoNeoImpl{}
	// dao.CreateTree(&node1)
	n1, err := dao.FindNode("test.go", 1)
	if err != nil {
		panic(err)
	}
	n2, err := dao.FindNode("test.go", 7)
	if err != nil {
		panic(err)
	}
	msg, err := dao.Connect(n1, n2)
	if err != nil {
		panic(err)
	}
	fmt.Println(msg)
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

// Parse through a panic message and find originating file/line number
func parsePanic(filesToParse []string) {

	//Generates test stack traces (run once and redirect to log file)
	// "go run main.go 2>stackTrace.log"
	//generateStackTrace()

	//Open stack trace log file (assume there will be a log file named this)
	file, err := os.Open("stackTrace.log")
	if err != nil {
		fmt.Println("Error opening file")
	}

	//Parse through stack trace log file
	scanner := bufio.NewScanner(file)
	stackTrc := []stackTraceStruct{}
	tempStackTrace := stackTraceStruct{
		msgType: "",
		fnLine:  map[string]string{},
	}

	//Scan through each line of log file and do analysis
	for scanner.Scan() {
		logStr := scanner.Text()

		//Check for beginning of new stack trace statement (create new trace struct for new statement)
		// keyword "serving" is found in the first line of each new stack trace
		if strings.Contains(logStr, "serving") {

			//Make sure attributes aren't empty before adding it
			if tempStackTrace.msgType != "" && len(tempStackTrace.fnLine) != 0 {
				stackTrc = append(stackTrc, tempStackTrace)
			}

			//New statement trace
			tempStackTrace = stackTraceStruct{
				msgType: "",
				fnLine:  map[string]string{},
			}

			//Assign panic type
			if strings.Contains(logStr, "panic") {
				tempStackTrace.msgType = "panic"
			}
		}

		//Check if line contains a possible file name, store to map of fileName+LineNumber
		if strings.Contains(logStr, ".go") {
			fileName := logStr[strings.LastIndex(logStr, "/")+1 : strings.LastIndex(logStr, ":")]
			indxLineNumStart := strings.LastIndex(logStr, ":")
			lineNumLarge := logStr[indxLineNumStart+1:]

			//If space in line number string with +0xaa, etc
			var lineNum string
			if strings.Contains(lineNumLarge, " ") {
				lineNum = lineNumLarge[0:strings.Index(lineNumLarge, " ")]
			} else {
				lineNum = lineNumLarge
			}

			//Check for originating files where the exception was thrown (could be multiple files, parent calls, etc)
			for index := range filesToParse {
				if strings.Contains(filesToParse[index], fileName) {
					tempStackTrace.fnLine[fileName] = lineNum
				}
			}
		}
	}

	//Add last entry
	stackTrc = append(stackTrc, tempStackTrace)

	//Test print the processed stack traces
	for index := range stackTrc {
		fmt.Println(stackTrc[index])
	}
}

//Helper function to generate a sample panic msg
func generateStackTrace() {
	//num := 5
	//if num != 5{
	//	panic("BADBAD")
	//}else{
	//	panic("Test 3 panic")
	//}
}

//Parse project to create log types
func parseProject(projectRoot string) []model.LogType {

	//fmt.Println("Project root is: " + projectRoot)

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

	//Check all function declarations
	// funcDecList := functionDecls(filesToParse)
	// findPanics(filesToParse)

	//Parses panic stack trace message
	parsePanic(filesToParse)

	return logTypes
}

//Struct for quick access to the function declaration nodes
type fdeclStruct struct {
	node     ast.Node
	fd       *ast.FuncDecl
	filePath string
	lineNum  string
	Name     string
}

//Stores the file path and line # (Node pointers there for extra info)
type panicStruct struct {
	node     ast.Node
	pd       *ast.CallExpr
	filePath string
	lineNum  string
}

//Parsing a panic runtime stack trace
type stackTraceStruct struct {
	msgType string
	fnLine  map[string]string
}

//Helper function to find origin of function (not used but may need later)
func findFuncOrigin(name string, funcDecList []fdeclStruct) {
	for _, value := range funcDecList {
		if name == value.fd.Name.Name {
			fmt.Println(name, value.filePath, value.lineNum)
		}
	}
}

/*
 Determines if a function is called somewhere else based on its name (path and line number)
  -currently goes through all files and finds if it's used
*/
func functionDecls(filesToParse []string) []fdeclStruct {

	//Map of all function names with a [line number, file path]
	// ex: ["HandleMessage" : {"45":"insights-results-aggregator/consumer/processing.go"}]
	//They key is the function name. Go doesn't support function overloading -> so each name will be unique
	functMap := map[string][]string{}
	functCalls := []fdeclStruct{}

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

		//Inspect AST for explicit function declarations
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

				//Add astNode and the FuncDecl node to the function calls
				functCalls = append(functCalls, fdeclStruct{
					currNode,
					fdNode,
					fpath,
					linePos,
					functionName,
				})
			}
			return true
		})

		//Inspect the AST Call Expressions (where they call a function)
		ast.Inspect(node, func(currNode ast.Node) bool {
			callExprNode, ok := currNode.(*ast.CallExpr)
			if ok {
				//Filter single function calls such as parseMsg(msg.Value)
				functionName := packageName + fmt.Sprint(callExprNode.Fun)
				if _, found := functMap[functionName]; found {
					//fmt.Println("The function " + functionName + " was found on line " + val[0] + " in " + val[1])
					fmt.Println("")
				}
			}
			return true
		})
	}

	return functCalls
}

//Finds all panic statements (not currently used, but may need later)
func findPanics(filesToParse []string) []panicStruct {

	panicList := []panicStruct{}

	for _, file := range filesToParse {
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			log.Error().Err(err).Msg("Error parsing file " + file)
		}

		//Inspect call expressions
		ast.Inspect(node, func(currNode ast.Node) bool {
			callExprNode, ok := currNode.(*ast.CallExpr)
			if ok {
				//If it's a panic statement, add to the struct
				if name := fmt.Sprint(callExprNode.Fun); name == "panic" {
					lnNum := fmt.Sprint(fset.Position(callExprNode.Pos()).Line)
					panicList = append(panicList, panicStruct{
						node:     currNode,
						pd:       callExprNode,
						filePath: file,
						lineNum:  lnNum,
					})
				}
			}
			return true
		})

		//Print file name/line number/panic
		for _, value := range panicList {
			fmt.Println(value.filePath, value.lineNum, fmt.Sprint(value.pd.Fun))
		}
	}
	return panicList
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

// Returns logTypes with map struct
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
			// fmt.Println("checking funcDecl", funcDecl.Name)
			// cfg := FnCfgCreator{}
			// root := cfg.CreateCfg(funcDecl, base, fset, map[int]string{})
			// printCfg(root, "")
			// fmt.Println()
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

	//Create CFG -- NEED to call after regex has been created
	regexes := mapLogRegex(logInfo)
	ast.Inspect(node, func(n ast.Node) bool {
		// Keep track of the current parent function the log statement is contained in
		if funcDecl, ok := n.(*ast.FuncDecl); ok {
			fmt.Println("checking funcDecl", funcDecl.Name)
			cfg := FnCfgCreator{}
			root := cfg.CreateCfg(funcDecl, base, fset, regexes)
			printCfg(root, "")
			fmt.Println()
		}
		return true
	})

	return logInfo, varsInLogs
}

//Helper function to create map of log to regex
func mapLogRegex(logInfo []model.LogType) map[int]string {
	regexMap := make(map[int]string)

	//Grab line number + regex string
	for index := range logInfo {
		ln := logInfo[index].LineNumber
		reg := logInfo[index].Regex
		regexMap[ln] = reg
	}

	return regexMap
}

//Generates regex for a given log string
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

// CreateCfg creates the CFG For a given function declaration, filepath and file, and a map of regexes containde within the file.
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
	if fn, ok := root.(*db.FunctionNode); ok {
		fn.Child = node
	}

	return root
}

//Prints out the contents of the CFG (recursively)
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

func (fnCfg *FnCfgCreator) constructSubCfg(block *cfg.Block, base string, fset *token.FileSet, regexes map[int]string) (root db.Node) {
	if block == nil || block.Nodes == nil {
		return nil
	}

	conditional := false
	var prev db.Node
	var current db.Node
	// fmt.Println(block.Succs)

	// Convert each node in the block into a db.Node (if it is one we want to keep)
	for i, node := range block.Nodes {
		last := i == len(block.Nodes)-1
		conditional = len(block.Succs) > 1 //2 succesors for conditional block

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
			} else {
				conditional.TrueChild = fnCfg.constructSubCfg(block.Succs[0], base, fset, regexes)
				fnCfg.blocks[block.Succs[0]] = conditional.TrueChild
			}

			if fail, ok := fnCfg.blocks[block.Succs[1]]; ok {
				conditional.FalseChild = fail
			} else {
				conditional.FalseChild = fnCfg.constructSubCfg(block.Succs[1], base, fset, regexes)
				fnCfg.blocks[block.Succs[1]] = conditional.FalseChild
			}

			// Set the predecessor's child to be the conditional (which may be some initialization call)
			switch node := prev.(type) {
			case *db.FunctionNode:
				node.Child = db.Node(conditional)
			case *db.StatementNode:
				node.Child = db.Node(conditional)
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
			case *db.StatementNode:
				node.Child = child
			}
		}

		current = nil
	}

	// The root was nil, so try to get the next block.
	// If the block is part of a for statement it would infinitely recurse, so leave it nil.
	if root == nil && len(block.Succs) == 1 && !strings.Contains(block.String(), "for") {
		if subCfg, ok := fnCfg.blocks[block.Succs[0]]; ok {
			root = subCfg
		} else {
			root = fnCfg.constructSubCfg(block.Succs[0], base, fset, regexes)
			fnCfg.blocks[block.Succs[0]] = root
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
			if (strings.Contains(val, "Msg") || strings.Contains(val, "Err")) && isFromLog(selectStmt) {
				line := fset.Position(expr.Pos()).Line
				node = db.Node(&db.StatementNode{
					Filename:   filepath.ToSlash(relPath),
					LineNumber: line,
					LogRegex:   regexes[line],
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
		} else {
			call.Child = node
		}
	}
}

func getStatementNode(stmt ast.Node, base string, fset *token.FileSet, regexes map[int]string) (node db.Node) {
	switch stmt := stmt.(type) {
	case *ast.ExprStmt:
		node = getExprNode(stmt.X, base, fset, false, regexes)
	case *ast.FuncDecl:
		relPath, _ := filepath.Rel(base, fset.File(stmt.Pos()).Name())
		node = db.Node(&db.FunctionNode{
			Filename:     filepath.ToSlash(relPath),
			LineNumber:   fset.Position(stmt.Pos()).Line,
			FunctionName: stmt.Name.Name,
		})
	case *ast.AssignStmt:
		// Found an assignment
		var first db.Node
		var current db.Node
		var prev db.Node
		for _, expr := range stmt.Rhs {
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
				}

				prev = current
			}
		}
		node = first
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
