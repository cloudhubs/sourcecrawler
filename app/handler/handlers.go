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

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&request); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	defer r.Body.Close()

	//actually parse the project
	logsTypes := parseProject(request.ProjectRoot)

	for _, logType := range logsTypes {
		fmt.Println(logType.FilePath)
		fmt.Println(logType.LineNumber)
		fmt.Println(logType.Regex)

		// TODO: confirm path format. How much info to pass?

		// save logType to DB
		if err := db.Save(&logType).Error; err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	logType := model.LogType{}
	logType.Regex = "This is a test for type .+"
	logType.FilePath = "some/path/file.go"
	logType.LineNumber = 123

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
