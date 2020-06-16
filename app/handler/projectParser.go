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
	"runtime"
	"sourcecrawler/app/cfg"
	"sourcecrawler/app/db"
	"sourcecrawler/app/logsource"
	"sourcecrawler/app/model"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

func createTestNeoNodes() {
	// node7 := db.StatementNode{"test.go", 7, "", nil}
	// node6 := db.StatementNode{"test.go", 6, "another log regex", &node7}
	// node5 := db.StatementNode{"test.go", 5, "", &node6}
	// node4 := db.StatementNode{"test.go", 4, "my log regex", &node6}
	// node3 := db.ConditionalNode{"test.go", 3, "myvar != nil", &node4, &node5}
	// node2 := db.StatementNode{"test.go", 2, "", &node3}
	// node1 := db.StatementNode{"test.go", 1, "", &node2}

	// dao := db.NodeDaoNeoImpl{}
	// //close driver when dao goes out of scope
	// defer dao.DisconnectFromNeo()
	// dao.CreateTree(&node1)

	nodeG := db.StatementNode{"connect.go", 7, "do nothing", nil}
	nodeF := db.FunctionNode{"connect.go", 6, "main", nil}
	nodeE := db.ConditionalNode{"connect.go", 5, "yes?", &nodeF, &nodeG}
	nodeD := db.FunctionDeclNode{"connect.go", 4, "func", nil, nil, nil, &nodeE}

	nodeC := db.StatementNode{"connect.go", 3, "the end", nil}
	nodeB := db.StatementNode{"connect.go", 2, "", &nodeC}
	nodeA := db.FunctionDeclNode{"connect.go", 1, "main", nil, nil, nil, &nodeB}

	cfg.PrintCfg(&nodeD, "")
	fmt.Println()
	cfg.ConnectStackTrace([]db.Node{&nodeA, &nodeD})
	cfg.PrintCfg(&nodeD, "")
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

//Helper function to grab OS separator
func grabOS() string {
	if runtime.GOOS == "windows" {
		return "\\"
	} else {
		return "/"
	}
}

// Parse through a panic message and find originating file/line number/function name
func parsePanic(filesToParse []string, projectRoot string) []stackTraceStruct {

	//Generates test stack traces (run once and redirect to log file)
	// "go run main.go 2>stackTrace.log"
	//testCondPanic(15)
	//testPanic()
	separator := grabOS()

	//Helper map for quick lookup
	localFilesMap := make(map[string]string)
	for index := range filesToParse {
		shortFileName := filesToParse[index]
		shortFileName = shortFileName[strings.LastIndex(shortFileName, separator)+1:]
		localFilesMap[shortFileName] = "exists"
	}

	//Open stack trace log file (assume there will be a log file named this)
	file, err := os.Open("stackTrace.log")
	if err != nil {
		fmt.Println("Error opening file")
	}

	//Parse through stack trace log file
	scanner := bufio.NewScanner(file)
	stackTrc := []stackTraceStruct{}
	tempStackTrace := stackTraceStruct{
		id:       1,
		msgLevel: "",
		fileName: "",
		lineNum:  "",
		funcName: "",
	}
	fileLineNum := 1
	id := 1
	doneAdding := false
	tempFuncName := ""
	doneFn := false

	//Scan through each line of log file and do analysis
	for scanner.Scan() {
		logStr := scanner.Text()

		//Check for beginning of new stack trace statement (create new trace struct for new statement)
		// keyword "serving" is found in the first line of each new stack trace
		if strings.Contains(logStr, "serving") {

			//Make sure attributes aren't empty before adding it
			if tempStackTrace.msgLevel != "" && tempStackTrace.fileName != "" &&
				tempStackTrace.lineNum != "" && tempStackTrace.funcName != "" {
				tempStackTrace.id = id
				stackTrc = append(stackTrc, tempStackTrace)
				doneAdding = false //status of adding file + line number
				doneFn = false     //status of adding function name
				id++
			}

			//New statement trace
			tempStackTrace = stackTraceStruct{
				id:       id,
				msgLevel: "",
				fileName: "",
				lineNum:  "",
				funcName: "",
			}

			//Assign panic type
			if strings.Contains(logStr, "panic") {
				tempStackTrace.msgLevel = "panic"
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
			// We only want to match local files and not any extraneous files
			if _, ok := localFilesMap[fileName]; ok {
				if !doneAdding {
					tempStackTrace.fileName = fileName
					tempStackTrace.lineNum = lineNum
					doneAdding = true
				}
			}
		}

		//!-- NOTE: function that calls another function in another file
		//         will only contain 1 function call in a stack trace line
		//!-- NOTE: assuming there is no function named go() --!
		//Process function name lines (doesn't contain .go)
		if !strings.Contains(logStr, ".go") &&
			strings.Contains(logStr, "(") && strings.Contains(logStr, ")") {

			startIndex := strings.LastIndex(logStr, ".")
			endIndex := strings.LastIndex(logStr, "(")

			//Functions with multiple calls (has multiple . operators)
			if startIndex != -1 && endIndex != -1 && startIndex < endIndex {
				tempFuncName = logStr[startIndex+1 : endIndex]
			}

			//Function is a single standalone function (only example currently is panic())
			if startIndex == -1 {
				tempFuncName = logStr[:endIndex]
			}

			//No parenthesis for function name OR a custom function(ex: .Serve or .callOtherPanic(...))
			if startIndex > endIndex {
				//Function with (...) as args (these are the ones that we are interested in) -- grab first on stack
				if strings.Contains(logStr, "...") {
					specialIndex := strings.Index(logStr, ".")
					tempFuncName = logStr[specialIndex+1 : endIndex]
					//Add the first origin function (should be the correct function where error occured)
					if !doneFn {
						tempStackTrace.funcName = tempFuncName
						doneFn = true
					}
				} else {
					tempFuncName = logStr[startIndex+1:]
				}
			}
			//Test print other function names
			//fmt.Println("Function name:", tempFuncName)
		}

		//Update file line number
		fileLineNum++
	}

	//Add last entry
	stackTrc = append(stackTrc, tempStackTrace)

	return stackTrc
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
	errorList := parsePanic(filesToParse, projectRoot)
	printErrorList(errorList)

	return logTypes
}

//Helper function to test print parsed info from stack trace
func printErrorList(errorList []stackTraceStruct) {
	//Test print the processed stack traces
	for _, value := range errorList {
		fmt.Printf("%d: %s in %s -- line %s from function %s\n",
			value.id, value.msgLevel, value.fileName, value.lineNum, value.funcName)
	}
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

//Parsing a panic runtime stack trace (id, messageLevel, file name and line #, function name)
type stackTraceStruct struct {
	id       int
	msgLevel string
	fileName string
	lineNum  string
	funcName string
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
	//fmt.Println("checking if", call, "depends on", parent.Name)
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
				if (strings.Contains(val, "Msg") || val == "Err") && logsource.IsFromLog(fn) {
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
		//fmt.Println()
	}

	//Create CFG -- NEED to call after regex has been created
	regexes := mapLogRegex(logInfo)
	ast.Inspect(node, func(n ast.Node) bool {
		// Keep track of the current parent function the log statement is contained in
		if funcDecl, ok := n.(*ast.FuncDecl); ok {
			//fmt.Println("checking funcDecl", funcDecl.Name)
			cfg := cfg.FnCfgCreator{}
			/*root := */ cfg.CreateCfg(funcDecl, base, fset, regexes)
			//printCfg(root, "")
			//fmt.Println()
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
