package handler

import (
	"encoding/json"
	"fmt"
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
		LogMessages []string `json:"logMessages"`
		ProjectRoot string `json:"projectRoot"`
	}{}

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&request); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	defer r.Body.Close()

	//fmt.Println("stack trace:",request.StackTrace)
	//fmt.Println(request.LogMessages)
	//fmt.Println("Project Root,", request.ProjectRoot)

	//1 -- parse stack trace for functions that led to exception
	parsedStack := parsePanic(request.ProjectRoot, request.StackTrace)
	fmt.Println(parsedStack)

	//2 -- Parse project for log statements with regex + line + file name
	logTypes := parseProject(request.ProjectRoot)
	//fmt.Println(logTypes)

	//Assign log messages (regex)
	for _, value := range logTypes{
		request.LogMessages = append(request.LogMessages, value.Regex)
		fmt.Println(value)
	}

	//3 -- Construct the CFG
	regexMap := mapLogRegex(logTypes)
	fset := token.NewFileSet()
	cfgCreator := cfg.NewFnCfgCreator()

	//TODO: pass a func decl node here (use functions from the stack trace?)
	cfgCreator.CreateCfg(nil, request.ProjectRoot, fset, regexMap)

	//4 -- Connect the CFG nodes together?
}
