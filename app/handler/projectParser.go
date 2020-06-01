package handler

import (
	"github.com/jinzhu/gorm"
	"sourcecrawler/app/model"
	"time"
)

//Parse project to create log types
func parseProject(projectRoot string) []model.LogType {
	// TODO: parse project and create log types

	logTypes := []model.LogType{}

	//Sample log type for testing (will remove once parser is implemented)
	tempModel := gorm.Model{
		ID:        1,
		CreatedAt: time.Time{},
		UpdatedAt: time.Time{},
		DeletedAt: nil,
	}
	type1 := model.LogType{
		Model:      tempModel,
		FilePath:   "./",
		LineNumber: 1,
		Regex:      "",
	}
	logTypes = append(logTypes, type1)

	//Create regex based on specific log type
	for index := range logTypes {

		//TODO: Assuming ID identifies the type of log? (info, error, etc.)
		//if logTypes[index].ID == 1 {
		//	logTypes[index].Regex = "^[a-zA-Z0-9]*$"
		//}

		//Currently all log type messages match this pattern
		logTypes[index].Regex = "^[a-zA-Z0-9]*$"
	}

	return logTypes
}


