package handler

import (
	"sourcecrawler/app/model"

	"github.com/jinzhu/gorm"

	"fmt"
	"regexp"
)

func matchLog(logMessage string, db *gorm.DB) (*model.LogSourceResponse, error) {
	// TODO: find log in database

	logTypes := []model.LogType{}
	db.Find(&logTypes)

	// Initialize default response
	var response *model.LogSourceResponse
	err := fmt.Errorf("Could not match any log type to \"%s\"", logMessage)

	// Find the first logType where the logMessage matches the regex
	for _, logType := range logTypes {
		if regex, err := regexp.Compile(logType.Regex); err == nil {
			if regex.Match([]byte(logMessage)) {
				// Found a log type, set values
				response = &model.LogSourceResponse{
					LineNumber: logType.LineNumber,
					FilePath:   logType.FilePath,
				}
				err = nil
				break
			}
		}
	}

	return response, err
}
