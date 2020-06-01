package handler

import (
	"sourcecrawler/app/model"

	"github.com/jinzhu/gorm"

	"regexp"
)

func matchLog(logMessage string, db *gorm.DB) model.LogSourceResponse {
	// TODO: find log in database

	logTypes := []model.LogType{}
	db.Find(&logTypes)

	// Find the first logType where the logMessage matches the regex
	response := model.LogSourceResponse{}
	for _, logType := range logTypes {
		if regex, err := regexp.Compile(logType.Regex); err == nil {
			if regex.Match([]byte(logMessage)) {
				response.LineNumber = logType.LineNumber
				response.FilePath = logType.FilePath
				break
			}
		}
	}

	return response
}
