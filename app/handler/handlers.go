package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sourcecrawler/app/model"

	"github.com/jinzhu/gorm"
)

func CreateProjectLogTypes(db *gorm.DB, w http.ResponseWriter, r *http.Request) {
	request := model.ParseProjectRequest{}

	fmt.Println("Requesting... Creating project log types")

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&request); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	defer r.Body.Close()

	// TODO: actually parse the project
	logType := parseProject(request.ProjectRoot)

	if err := db.Save(&logType).Error; err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
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

	var logMessage string = request.LogMessage

	// TODO: try to match logMessage to a regex in logTypes
	fmt.Println("Requested", logMessage)
	response := model.LogSourceResponse{}
	response.FilePath = "some/file/path.go"
	response.LineNumber = 888
	respondJSON(w, http.StatusOK, response)
}

func GetAllLogTypes(db *gorm.DB, w http.ResponseWriter, r *http.Request) {
	types := []model.LogType{}
	db.Find(&types)
	respondJSON(w, http.StatusOK, types)
}
