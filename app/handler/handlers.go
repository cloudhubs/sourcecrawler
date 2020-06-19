package handler

import (
	"encoding/json"
	"fmt"

	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"regexp"
	"sourcecrawler/app/cfg"
	neoDb "sourcecrawler/app/db"
	"sourcecrawler/app/model"
	_ "strings"

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
	for _, goFile := range gatherGoFiles(request.ProjectRoot) {
		f, err := parser.ParseFile(fset, goFile, nil, parser.ParseComments)
		if err != nil {
			log.Error().Err(err).Msg("unable to parse file")
		}

		logInfo, _ := findLogsInFile(goFile, request.ProjectRoot)
		regexes := mapLogRegex(logInfo)

		c := cfg.FnCfgCreator{}
		ast.Inspect(f, func(node ast.Node) bool {
			if fn, ok := node.(*ast.FuncDecl); ok {
				// fmt.Println("parsing", fn)
				decls = append(decls, c.CreateCfg(fn, request.ProjectRoot, fset, regexes))
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
	parsedStack := parsePanic(request.ProjectRoot, request.StackTrace)
	fmt.Println(parsedStack)

	//2 -- Parse project for log statements with regex + line + file name
	logTypes := parseProject(request.ProjectRoot)

	//Matching log messages to a regex (only returns used regexes)
	regexes := []string{}
	for index := range request.LogMessages {
		for _, value := range logTypes {
			matched, _ := regexp.MatchString(value.Regex, request.LogMessages[index])
			if matched {
				regexes = append(regexes, value.Regex)
				fmt.Println("Valid regexes:", value.Regex)
				break
			}
		}
	}

	filesToParse := gatherGoFiles(request.ProjectRoot)
	stackFuncNodes := findFunctionNodes(filesToParse)
	astFdNodes := []*ast.FuncDecl{} //Contains all the relevant function nodes used in the stack trace

	//Adds function nodes that are used in the stack trace
	for _, value := range parsedStack {
		for index := range value.funcName {
			fdNode := getFdASTNode(value.fileName[index], value.funcName[index], stackFuncNodes)
			if fdNode != nil {
				astFdNodes = append(astFdNodes, fdNode)
			}
		}
	}

	//3 -- create CFG nodes for each function
	regexMap := mapLogRegex(logTypes) //Create regexes based from the parseProject logTypes
	fset := token.NewFileSet()
	var decls []neoDb.Node
	cfgCreator := cfg.FnCfgCreator{}

	//Only pass in the FuncDecl nodes that are used in the stack trace
	for _, fdNode := range astFdNodes {
		decls = append(decls, cfgCreator.CreateCfg(fdNode, request.ProjectRoot, fset, regexMap))
	}

	//4 -- Connect the CFG nodes together?
	decls = cfg.ConnectFnCfgs(decls)

	//Test print the declarations
	for _, decl := range decls {
		cfg.PrintCfg(decl, "")
		fmt.Println()
	}
}
