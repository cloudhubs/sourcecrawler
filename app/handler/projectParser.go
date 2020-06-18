package handler

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
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

//Gathers all go files to parse
func gatherGoFiles(projectRoot string) []string {
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

	return filesToParse
}

//Parse project to create log types
func parseProject(projectRoot string) []model.LogType {

	//Holds a slice of log types
	logTypes := []model.LogType{}
	variableDeclarations := varDecls{}
	variablesUsedInLogs := map[string]struct{}{}
	filesToParse := gatherGoFiles(projectRoot)

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

	//Get stack trace string
	file, err := os.Open("stackTrace.log")
	if err != nil {
		log.Error().Msg("Error opening file")
	}
	scanner := bufio.NewScanner(file)
	stackTraceString := ""
	for scanner.Scan() {
		stackTraceString += scanner.Text() + "\n"
	}

	//Parses panic stack trace message
	parsePanic(projectRoot, stackTraceString)
	//errorList := parsePanic(projectRoot, "")
	//printErrorList(errorList)

	return logTypes
}

//Helper function to find origin of function (not used but may need later)
func findFuncOrigin(name string, funcDecList []fdeclStruct) {
	for _, value := range funcDecList {
		if name == value.fd.Name.Name {
			fmt.Println(name, value.filePath, value.lineNum)
		}
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

/*
 Determines if a function is called somewhere else based on its name (path and line number)
  -currently goes through all files and finds if it's used
*/
func functionDeclsMap(filesToParse []string) map[string][]string {

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
		//packageName := node.Name.Name + ":"

		//Inspect AST for explicit function declarations
		ast.Inspect(node, func(currNode ast.Node) bool {
			fdNode, ok := currNode.(*ast.FuncDecl)
			if ok {
				//package name is appended to separate diff functions across packages
				functionName := fdNode.Name.Name
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
		//ast.Inspect(node, func(currNode ast.Node) bool {
		//	callExprNode, ok := currNode.(*ast.CallExpr)
		//	if ok {
		//		//Filter single function calls such as parseMsg(msg.Value)
		//		//functionName := packageName + fmt.Sprint(callExprNode.Fun)
		//		functionName := packageName
		//		if _, found := functMap[functionName]; found {
		//			//fmt.Println("The function " + functionName + " was found on line " + val[0] + " in " + val[1])
		//			//fmt.Println("")
		//		}
		//	}
		//	return true
		//})
	}

	return functMap
}

/*
 Determines if a function is called somewhere else based on its name (path and line number)
  -currently goes through all files and finds if it's used
*/
func findFunctionNodes(filesToParse []string) []fdeclStruct {

	//Map of all function names with a [line number, file path]
	// ex: ["HandleMessage" : {"45":"insights-results-aggregator/consumer/processing.go"}]
	//They key is the function name. Go doesn't support function overloading -> so each name will be unique
	functCalls := []fdeclStruct{}

	//Inspect each file for calls to this function
	for _, file := range filesToParse {
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			log.Error().Err(err).Msg("Error parsing file " + file)
		}

		//Grab package name - needed to prevent duplicate function names across different packages, keep colon
		//packageName := node.Name.Name + ":"

		//Inspect AST for explicit function declarations
		ast.Inspect(node, func(currNode ast.Node) bool {
			fdNode, ok := currNode.(*ast.FuncDecl)
			if ok {
				//package name is appended to separate diff functions across packages
				functionName := fdNode.Name.Name
				linePos := strconv.Itoa(fset.Position(fdNode.Pos()).Line)
				fpath, _ := filepath.Abs(fset.File(fdNode.Pos()).Name())

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
	}

	return functCalls
}

//Takes in a file name + function name -> returns AST FuncDecl node
func getFdASTNode(fileName string, functionName string, stackFuncNodes []fdeclStruct) *ast.FuncDecl {

	for _, val := range stackFuncNodes {
		if strings.Contains(val.filePath, fileName) && functionName == val.Name {
			return val.fd
		}
	}

	return nil
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
						//fmt.Println("\tcall expression uses", param)
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

	//Check for nil node
	if node == nil {
		return varDecls{
			asns:  nil,
			decls: nil,
		}
	}

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
	node, err := parser.ParseFile(fset, base+"/"+path, nil, parser.ParseComments)
	if err != nil {
		log.Error().Err(err).Msg("unable to parse file")
	}

	varsInLogs := map[string]struct{}{}
	logInfo := []model.LogType{}
	logCalls := []fnStruct{}

	//Helper structure to hold logTypes with function types (msg, msgf, err)
	//lnTypes := make(map[model.LogType]string)

	//Check for nil node
	if node == nil {
		return nil, nil
	}

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
				//fmt.Println("type arg", reflect.TypeOf(a), a)
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
