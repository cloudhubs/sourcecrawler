package handler

import (
	"sourcecrawler/app/model"

	"github.com/jinzhu/gorm"
)

func matchLog(logMessage string, db *gorm.DB) model.LogSourceResponse {
	// TODO: find log in database

	logTypes := []model.LogType{}
	db.Find(&logTypes)

	response := model.LogSourceResponse{}

	return response
}
