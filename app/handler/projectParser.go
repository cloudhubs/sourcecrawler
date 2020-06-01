package handler

import (
	"fmt"
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
		FilePath:   projectRoot,
		LineNumber: 1,
		Regex:      "",
	}
	logTypes = append(logTypes, type1)

	//Create regex based on specific log type
	for index := range logTypes {

		//TODO: If log type is Msgf - need to build regex for variable string


		//TODO: if basic Msg
		logTypes[index].Regex = "^[a-zA-Z0-9 ]*$"
	}

	fmt.Println("ProjectParser is called")

	return logTypes
}
