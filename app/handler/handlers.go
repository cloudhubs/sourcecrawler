package handler

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/token"
	"net/http"
	"sourcecrawler/app/cfg"
	"sourcecrawler/app/model"

	"github.com/jinzhu/gorm"
)

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
		ProjectRoot string `json:"projectRoot"`
	}{}

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&request); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	defer r.Body.Close()

	//TODO: function to match all log messages to a particular log type (match regex)
	// returns a list of regex string
	//1 -- parse stack trace for functions that led to exception
	parsedStack := parsePanic(request.ProjectRoot, request.StackTrace)

	//2 -- Parse project for log statements with regex + line + file name
	logInfo := parseProject(request.ProjectRoot)

	//Assign log messages (regex)
	//for _, value := range logInfo{
	//	request.LogMessages = append(request.LogMessages, value.Regex)
	//	fmt.Println(value)
	//}

	filesToParse := gatherGoFiles(request.ProjectRoot)
	stackFuncNodes := findFunctionNodes(filesToParse)
	astFdNodes := []*ast.FuncDecl{}

	//Add the function node if its used in the stack trace
	for _, value := range parsedStack{
		for index := range value.funcName{
			fdNode := getFdASTNode(value.fileName[index], value.funcName[index], stackFuncNodes)
			if fdNode != nil{
				astFdNodes = append(astFdNodes,fdNode)
			}
		}
	}

	//Check nil node
	for _, value := range astFdNodes{
		if value != nil{
			fmt.Println("Nodes aren't nil")
		}else{
			fmt.Println("nil node")
		}
	}

	//3 -- create CFG nodes for each function
	cfgCreator := cfg.NewFnCfgCreator()
	fset := token.NewFileSet()
	regexMap := mapLogRegex(logInfo)

	//TODO: currently error with nil memory address
	for _, val := range astFdNodes {
		cfgCreator.CreateCfg(val, request.ProjectRoot, fset, regexMap)
	}

	//4 -- Connect the CFG nodes together?

}
