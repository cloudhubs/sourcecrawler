package handler

import (
	"encoding/json"
	"fmt"
	"github.com/mitchellh/go-z3"
	"sourcecrawler/app/helper"
	"sourcecrawler/app/unsafe"
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

func NeoTest(db *gorm.DB, w http.ResponseWriter, r *http.Request) {
	createTestNeoNodes()
}

func CreateProjectLogTypes(db *gorm.DB, w http.ResponseWriter, r *http.Request) {
	request := model.ParseProjectRequest{}

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&request); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	defer r.Body.Close()

	//actually parse the project
	logsTypes := parseProject(request.ProjectRoot)

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

	messages, err := unsafe.Unsafe(request.X, request.Msg)

	if err != nil {
		respondJSON(w, http.StatusOK, messages)
	} else {
		respondJSON(w, http.StatusBadRequest, nil)
	}

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
	parsedStack := parsePanic(request.ProjectRoot, request.StackTrace)

	//2 -- Parse project for log statements with regex + line + file name
	logTypes := parseProject(request.ProjectRoot)
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

				if shouldAppendFunction && wrap == nil{
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
	// 	cfg.ExpandCFGRecur(wrap, make([]*cfg.FnWrapper, 0))
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
func SliceProgram(w http.ResponseWriter, r *http.Request) {
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
	parsedStack := parsePanic(request.ProjectRoot, request.StackTrace)

	//2 -- Parse project for log statements with regex + line + file name
	logTypes := parseProject(request.ProjectRoot)

	// Matching log messages to a regex (only returns used regexes)
	seenLogTypes := []model.LogType{}
	for _, msg := range request.LogMessages {
		for _, value := range logTypes {
			matched, _ := regexp.MatchString(value.Regex, msg)
			if matched {
				seenLogTypes = append(seenLogTypes, value)
				//fmt.Println("Valid regexes:", value.Regex)
				break
			}
		}
	}

	// 3 -- Generate CFGs (including log information) for each function in the stack trace
	topLevelWrapper := cfg.SetupPersistentData(request.ProjectRoot)

	stack := parsedStack[0] //likely only one stack trace
	entryFile := stack.FileName[len(stack.FileName)-1]
	entryName := stack.FuncName[len(stack.FuncName)-1]

	//grab the entry function
	var entryFnNode ast.Node
	for _, file := range topLevelWrapper.ASTs{
		if strings.Contains(file.Name.Name, entryFile) {
			for _, decl := range file.Decls {
				if decl, ok := decl.(*ast.FuncDecl); ok {
					if strings.EqualFold(decl.Name.Name, entryName) {
						entryFnNode = decl
					}
				}
			}
		}
	}

	//expand the cfg
	entryFn := cfg.NewFnWrapper(entryFnNode, nil)
	entryFn.SetOuterWrapper(topLevelWrapper)
	cfg.ExpandCFG(entryFn)

	//find the block originating the exception
	exceptionBlock := cfg.FindPanicWrapper(entryFn, &stack)

	//label the tree starting from the exception block
	cfg.LabelCFG(exceptionBlock,logTypes,entryFn)

	//gather the paths
	pathList := cfg.CreateNewPath()
	paths := pathList.TraverseCFG(exceptionBlock, entryFn)

	//transform to z3
	config := z3.NewConfig()
	ctx := z3.NewContext(config)
	config.Close()
	defer ctx.Close()

	s := ctx.NewSolver()
	defer s.Close()

	for _, path := range paths {
		var z3group *z3.AST
		for _, expr := range path.Expressions {
			z3group = cfg.ConvertExprToZ3(ctx,expr,topLevelWrapper.Fset)
			if z3group != nil {
				s.Assert(z3group)
			}
		}

		if v := s.Check(); v != z3.True {
			fmt.Println("Unsolvable")
			continue
		}
		m := s.Model()
		assignments := m.Assignments()
		for name, val := range assignments {
			fmt.Printf("%s = %s\n", name, val)
		}
		m.Close()
	}


}