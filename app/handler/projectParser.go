package handler

import (
	"regexp"
	"sourcecrawler/app/model"
)

func parseProject(projectRoot string) []model.LogType {
	// TODO: parse project and create log types

	logTypes := []model.LogType{}

	//Create regex based on specific log type
	createRegex(logTypes)

	return logTypes
}

//Generate list of regexes from log function instances
func createRegex(logTypes []model.LogType){

	//Assign a regex to each logType
	for _, elem := range logTypes {

		//TODO: Assuming ID identifies the type of log? (info, error, etc.)
		if elem.ID == 1 {
			elem.Regex = "^[a-zA-Z0-9]*$"
		}
	}
}
