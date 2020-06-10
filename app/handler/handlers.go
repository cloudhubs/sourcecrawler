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
