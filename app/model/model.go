package model

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

type LogType struct {
	gorm.Model
	FilePath   string `json:"filePath"`
	LineNumber int    `json:"lineNumber"`
	Regex      string `json:"regex"`
}

type ParseProjectRequest struct {
	ProjectRoot string `json:"projectRoot"`
}

type LogSourceRequest struct {
	LogMessage string `json:"logMessage"`
}

type LogSourceResponse struct {
	FilePath   string `json:"filePath"`
	LineNumber int    `json:"lineNumber"`
	Regex      string `json:"regex"`
}

// DBMigrate will create and migrate the tables, and then make the some relationships if necessary
func DBMigrate(db *gorm.DB) *gorm.DB {
	db.AutoMigrate(&LogType{})
	return db
}

// // DBMigrate will create and migrate the tables, and then make the some relationships if necessary
// func DBMigrate(db *gorm.DB) *gorm.DB {
// 	db.AutoMigrate(&Project{}, &Task{})
// 	db.Model(&Task{}).AddForeignKey("project_id", "projects(id)", "CASCADE", "CASCADE")
// 	return db
// }
