package handler

import (
	"encoding/json"
	"fmt"
	"sourcecrawler/app/helper"
	"sourcecrawler/app/unsafe"
	"strconv"
	"strings"

	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"regexp"
	"sourcecrawler/app/cfg"
	neoDb "sourcecrawler/app/db"
	"sourcecrawler/app/model"
	_ "strings" //

	"github.com/jinzhu/gorm"
	"github.com/rs/zerolog/log"
)

func ConnectedCfgTest(db *gorm.DB, w http.ResponseWriter, r *http.Request) {
	request := model.ParseProjectRequest{}

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&request); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	defer r.Body.Close()

	var decls []neoDb.Node

	fset := token.NewFileSet()
	for _, goFile := range helper.GatherGoFiles(request.ProjectRoot) {
		f, err := parser.ParseFile(fset, goFile, nil, parser.ParseComments)
		if err != nil {
			log.Error().Err(err).Msg("unable to parse file")
		}

		//logInfo, _ := findLogsInFile(goFile, request.ProjectRoot)
		//regexes := mapLogRegex(logInfo)

		c := cfg.NewFnCfgCreator("pkg", request.ProjectRoot, fset)
		ast.Inspect(f, func(node ast.Node) bool {
			if fn, ok := node.(*ast.FuncDecl); ok {
				// fmt.Println("parsing", fn)
				decls = append(decls, c.CreateCfg(fn))
			}
			return true
		})
		// fmt.Println("done parsing")
	}
	// fmt.Println("finally done parsing")

	decls = cfg.ConnectFnCfgs(decls)

	for _, decl := range decls {
		cfg.PrintCfg(decl, "")
		fmt.Println()
	}

}

//func NeoTest(db *gorm.DB, w http.ResponseWriter, r *http.Request) {
//	helper.createTestNeoNodes()
//}

func CreateProjectLogTypes(db *gorm.DB, w http.ResponseWriter, r *http.Request) {
	request := model.ParseProjectRequest{}

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&request); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	defer r.Body.Close()

	//actually parse the project
	logsTypes := helper.ParseProject(request.ProjectRoot)

	for _, logType := range logsTypes {

		// save logType to DB
		if err := db.Save(&logType).Error; err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	respondJSON(w, http.StatusNoContent, nil)
}

func FindLogSource(db *gorm.DB, w http.ResponseWriter, r *http.Request) {
	request := model.LogSourceRequest{}

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&request); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	defer r.Body.Close()

	fmt.Println("Requested", request.LogMessage)
	if response, err := matchLog(request.LogMessage, db); response != nil && err != nil {
		// Successfully matched
		respondJSON(w, http.StatusOK, response)
	} else {
		// Could not find a match
		respondError(w, http.StatusNotFound, err.Error())
	}
}

func GetAllLogTypes(db *gorm.DB, w http.ResponseWriter, r *http.Request) {
	types := []model.LogType{}
	db.Find(&types)
	respondJSON(w, http.StatusOK, types)
}

//Test Endpoint for create CFG
func CreateCfgForFile(db *gorm.DB, w http.ResponseWriter, r *http.Request) {
	request := struct {
		FilePath string `json:"filePath"`
	}{}

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&request); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	defer r.Body.Close()
}

func UnsafeEndpoint(db *gorm.DB, w http.ResponseWriter, r *http.Request) {
	request := struct {
		X   int    `json:"x"`
		Msg string `json:"msg"`
	}{}

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&request); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	defer r.Body.Close()

	fmt.Println("ERROR")

	messages, err := unsafe.Unsafe(request.X, request.Msg)

	if err != nil {
		respondJSON(w, http.StatusOK, messages)
	} else {
		respondJSON(w, http.StatusBadRequest, nil)
	}
}

// NOTE: Values can be hardcoded in here for testing (Can't run from Postman without additional docker configuration)
//  Run the following: curl -X POST http://127.0.0.1:3000/container
func ContainerEndpoint(db *gorm.DB, w http.ResponseWriter, r *http.Request) {
	projectRoot := "./"
	fmt.Println("container endpoint")
	respondJSON(w, http.StatusOK, projectRoot)
}

//Test for rewrite cfg
func TestRewriteCFG(db *gorm.DB, w http.ResponseWriter, r *http.Request) {
	request := struct {
		StackTrace  string   `json:"stackTrace"`
		LogMessages []string `json:"logMessages"` //it holds raw log statements
		ProjectRoot string   `json:"projectRoot"`
	}{}

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&request); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	defer r.Body.Close()

	//1 -- parse stack trace for functions that led to exception
	parsedStack := helper.ParsePanic(request.ProjectRoot, request.StackTrace)

	//2 -- Parse project for log statements with regex + line + file name
	logTypes := helper.ParseProject(request.ProjectRoot)
	fmt.Println(logTypes)

	// 3 -- Generate CFGs (including log information) for each function in the stack trace
	outerWrapper := cfg.SetupPersistentData(request.ProjectRoot)
	var wrap *cfg.FnWrapper
	for _, goFile := range outerWrapper.GetASTs() {

		// extract CFGs for all relevant functions from this file
		ast.Inspect(goFile, func(node ast.Node) bool {
			if fn, ok := node.(*ast.FuncDecl); ok {
				// only add this function declaration if it is part of the stack trace
				shouldAppendFunction := false
				for _, value := range parsedStack {
					for index, stackFuncName := range value.FuncName {
						if stackFuncName == fn.Name.Name && strings.Contains(goFile.Name.Name, value.FileName[index]) {
							shouldAppendFunction = true
							break
						}
					}
				}

				if shouldAppendFunction && wrap == nil {
					wrap = cfg.NewFnWrapper(fn, make([]ast.Expr, 0))
				}
			}
			return true
		})
	}

	//TODO: Call CFG rewrite functions
	//Set wrapper properties if not nil
	// if wrap != nil {
	// 	wrap.Fset = outerWrapper.Fset
	// 	wrap.ASTs = outerWrapper.GetASTs()
	// 	cfg.ExpandCFG(wrap, make([]*cfg.FnWrapper, 0))
	// }

	// condStmts := make(map[ast.Node]cfg.ExecutionLabel)
	// vars := make([]ast.Node, 0)

	// path := cfg.CreateNewPath()
	// leaves := cfg.GetLeafNodes(w)
	// for _, leaf := range leaves {
	// 	path.TraverseCFG(leaf, condStmts, vars, wrap, make(map[string]ast.Node))
	// }

	response := ""
	respondJSON(w, http.StatusOK, response)
}

//Slices the program - first parses the stack trace, and then parses the project for log calls
// -Afterwards it creates the CFG and attempts to connect each of the functions in the stack trace
func SliceProgram(db *gorm.DB, w http.ResponseWriter, r *http.Request) {
	request := struct {
		StackTrace  string   `json:"stackTrace"`
		LogMessages []string `json:"logMessages"` //it holds raw log statements
		ProjectRoot string   `json:"projectRoot"`
	}{}

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&request); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	defer r.Body.Close()

	//1 -- parse stack trace for functions that led to exception
	parsedStack := helper.ParsePanic(request.ProjectRoot, request.StackTrace)

	//2 -- Parse project for log statements with regex + line + file name
	logTypes := helper.ParseProject(request.ProjectRoot)

	// Matching log messages to a regex (only returns used regexes)
	regexes := []string{}
	for index := range request.LogMessages {
		for _, value := range logTypes {
			matched, _ := regexp.MatchString(value.Regex, request.LogMessages[index])
			if matched {
				regexes = append(regexes, value.Regex)
				//fmt.Println("Valid regexes:", value.Regex)
				break
			}
		}
	}

	// 3 -- Generate CFGs (including log information) for each function in the stack trace
	var decls []neoDb.Node
	fset := token.NewFileSet()
	filesToParse := helper.GatherGoFiles(request.ProjectRoot)
	for _, goFile := range filesToParse {

		// only parse this file if it appears in the stack trace
		shouldParseFile := false
		for _, value := range parsedStack {
			for _, stackFileName := range value.FileName {
				if strings.Contains(goFile, stackFileName) {
					shouldParseFile = true
					break
				}
			}
		}

		if !shouldParseFile { // file not in stack trace, skip it
			continue
		}

		// get ast root for this file
		f, err := parser.ParseFile(fset, goFile, nil, parser.ParseComments)
		if err != nil {
			log.Error().Err(err).Msg("unable to parse file")
		}

		// get map of linenumber -> regex for thsi file
		//logInfo, _ := findLogsInFile(goFile, request.ProjectRoot)
		//regexes := mapLogRegex(logInfo)

		// extract CFGs for all relevant functions from this file
		c := cfg.NewFnCfgCreator("pkg", request.ProjectRoot, fset)
		ast.Inspect(f, func(node ast.Node) bool {
			if fn, ok := node.(*ast.FuncDecl); ok {
				// only add this function declaration if it is part of the stack trace
				shouldAppendFunction := false
				for _, value := range parsedStack {
					for index, stackFuncName := range value.FuncName {
						if stackFuncName == fn.Name.Name && strings.Contains(goFile, value.FileName[index]) {
							shouldAppendFunction = true
							break
						}
					}
				}

				if shouldAppendFunction {
					decls = append(decls, c.CreateCfg(fn))
				}
			}
			return true
		})
	}

	// // find all function declarations in this project
	// allFuncDecls := findFunctionNodes(filesToParse)
	// funcDeclMap := make(map[string]*ast.FuncDecl)
	// for _, fn := range allFuncDecls {
	// 	key := fmt.Sprintf("%v", fn.Name)
	// 	funcDeclMap[key] = fn.fd
	// }

	//4 -- Connect the CFG nodes together
	decls = cfg.ConnectFnCfgs(decls)

	for _, decl := range decls {
		fmt.Println(decl)
	}

	line, _ := strconv.Atoi(parsedStack[0].LineNum[0])

	node := *getExceptionNode(&decls[len(decls)-1], parsedStack[0].FileName[0], line)

	fmt.Println(node.GetFilename(), node.GetLineNumber(), node.GetParents())

	c := cfg.NewFnCfgCreator("pkg", request.ProjectRoot, fset)

	c.ConstructExecutionPaths(node, nil, nil, false)

	var returnVal [][]string

	for _, path := range c.GetExecutionPaths() {
		var stringPath []string
		for _, stmt := range path.Statements {
			stringPath = append(stringPath, stmt)
		}
		returnVal = append(returnVal, stringPath)
	}

	respondJSON(w, http.StatusOK, returnVal)

	// funcLabels := map[string]string{}
	// funcCalls := []neoDb.Node{}
	// mustHaves := []neoDb.Node{}
	// mayHaves := []neoDb.Node{}

	// for _, root := range decls {
	// 	newFuncs, newLabels := FindMustHaves(root, parsedStack, regexes)
	// 	funcLabels = cfg.MergeLabelMaps(funcLabels, newLabels)
	// 	funcCalls = append(funcCalls, newFuncs...)
	// }

	// mustHaves, mayHaves = cfg.FilterMustMay(funcCalls, mustHaves, mayHaves, funcLabels)

	// //Test print the declarations
	// for _, decl := range decls {
	// 	cfg.PrintCfg(decl, "")
	// 	fmt.Println()
	// }

	// response := struct {
	// 	MustHaveFunctions []string `json:"mustHaveFunctions"`
	// 	MayHaveFunctions  []string `json:"mayHaveFunctions"`
	// }{}

	// response.MustHaveFunctions = convertNodesToStrings(mustHaves)
	// response.MayHaveFunctions = convertNodesToStrings(mayHaves)

	// respondJSON(w, http.StatusOK, response)
}

// Returns the function names of the passed-in function call nodes
func convertNodesToStrings(elements []neoDb.Node) []string {
	encountered := map[string]bool{}
	result := []string{}
	for _, v := range elements {
		node := v.(*neoDb.FunctionNode)
		if encountered[node.FunctionName] == true {
			// Do not add duplicate.
		} else {
			// Record this element as an encountered element.
			encountered[node.FunctionName] = true
			// Append to result slice.
			result = append(result, node.FunctionName)
		}
	}
	// Return the new slice.
	return result
}

func getExceptionNode(node *neoDb.Node, filename string, linenumber int) *neoDb.Node {
	fmt.Println((*node).GetFilename(), (*node).GetLineNumber())

	if strings.Contains((*node).GetFilename(), filename) && (*node).GetLineNumber() == linenumber {
		return node
	}

	fmt.Println((*node).GetChildren())

	for child, _ := range (*node).GetChildren() {
		fmt.Println("child", child.GetFilename(), child.GetLineNumber())
		newNode := getExceptionNode(&child, filename, linenumber)
		if newNode != nil {
			return newNode
		}
	}

	return nil
}
