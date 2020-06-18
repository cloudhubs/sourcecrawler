package handler

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"sourcecrawler/app/cfg"
	neoDb "sourcecrawler/app/db"
	"sourcecrawler/app/model"

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

		fmt.Println(f.Name.Name, request.ProjectRoot)
		logInfo, _ := findLogsInFile(f.Name.Name+".go", request.ProjectRoot)
		regexes := mapLogRegex(logInfo)

		c := cfg.FnCfgCreator{}
		ast.Inspect(f, func(node ast.Node) bool {
			if fn, ok := node.(*ast.FuncDecl); ok {
				fmt.Println("parsing", fn)
				decls = append(decls, c.CreateCfg(fn, request.ProjectRoot, fset, regexes))
			}
			return true
		})
		fmt.Println("done parsing")
	}
	fmt.Println("finally done parsing")

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
