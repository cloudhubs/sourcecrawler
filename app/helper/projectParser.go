package helper

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sourcecrawler/app/db"
	"sourcecrawler/app/logsource"
	"sourcecrawler/app/model"
	"strings"

	"github.com/rs/zerolog/log"
)

func createTestNeoNodes() {
	//These literals don't have the right amount of values

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

	// nodeG := db.StatementNode{"connect.go", 7, "do nothing", nil}
	// nodeF := db.FunctionNode{"connect.go", 6, "main", nil}
	// nodeE := db.ConditionalNode{"connect.go", 5, "yes?", &nodeF, &nodeG}
	// nodeD := db.FunctionDeclNode{"connect.go", 4, "func", nil, nil, nil, &nodeE}

	// nodeC := db.StatementNode{"connect.go", 3, "the end", nil}
	// nodeB := db.StatementNode{"connect.go", 2, "", &nodeC}
	// nodeA := db.FunctionDeclNode{"connect.go", 1, "main", nil, nil, nil, &nodeB}

	// cfg.PrintCfg(&nodeD, "")
	// fmt.Println()
	// cfg.ConnectStackTrace([]db.Node{&nodeA, &nodeD})
	// cfg.PrintCfg(&nodeD, "")
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
func ParseProject(projectRoot string) []model.LogType {

	//Holds a slice of log types
	logTypes := []model.LogType{}
	variableDeclarations := varDecls{}
	variablesUsedInLogs := map[string]struct{}{}
	filesToParse := GatherGoFiles(projectRoot)

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

	// //Get stack trace string
	// file, err := os.Open("stackTrace.log")
	// if err != nil {
	// 	log.Error().Msg("Error opening file")
	// }
	// scanner := bufio.NewScanner(file)
	// stackTraceString := ""
	// for scanner.Scan() {
	// 	stackTraceString += scanner.Text() + "\n"
	// }

	// //Parses panic stack trace message
	// parsePanic(projectRoot, stackTraceString)
	// //errorList := parsePanic(projectRoot, "")
	// //printErrorList(errorList)

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
	node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
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

				currentLog.Regex = logsource.CreateRegex(v.Value)

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

func FindMustHaves(root db.Node, stackTrace []StackTraceStruct, regexs []string) ([]db.Node, map[string]string) {
	//must-have is on stack trace or contains a regex
	funcLabels := make(map[string]string)
	return findMustHavesRecur(root, stackTrace, regexs, &funcLabels), funcLabels
}

func findMustHavesRecur(n db.Node, stackTrace []StackTraceStruct, regexs []string, funcLabels *map[string]string) []db.Node {
	funcCalls := []db.Node{}

	if n != nil {
		if n, ok := n.(*db.FunctionNode); ok {
			funcCalls = append(funcCalls, n)
			if isInStack(n, stackTrace) || wasLogged(n, regexs) {
				(*funcLabels)[n.FunctionName] = "must"
			} else {
				(*funcLabels)[n.FunctionName] = "may"
			}
		}
		for child := range n.GetChildren() {
			funcCalls = append(funcCalls, findMustHavesRecur(child, stackTrace, regexs, funcLabels)...)
		}
	}
	return funcCalls
}

func isInStack(fn db.Node, stackTrace []StackTraceStruct) bool {
	//traverse
	for _, trace := range stackTrace {
		for _, funcName := range trace.FuncName {
			if fn, ok := fn.(*db.FunctionNode); ok {
				if fn.FunctionName == funcName {
					return true
				}
			}
		}
	}
	return false
}

func wasLogged(fn db.Node, regexs []string) bool {
	//traverse children that are not function
	//calls and see if any contain log statements seen
	//in regexs
	for child := range fn.GetChildren() {
		//stop at function nodes
		if _, ok := child.(*db.FunctionNode); ok {
			continue
		}
		if child, ok := child.(*db.StatementNode); ok {
			if strings.Contains(strings.Join(regexs, ","), child.LogRegex) {
				return true
			}
		}
		return wasLogged(child, regexs)
	}
	//if no children found a matching log statment
	//this function is not logged
	return false
}
