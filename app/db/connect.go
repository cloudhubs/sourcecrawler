package db

import (
	"github.com/neo4j/neo4j-go-driver/neo4j"
)

func getNeoDriver() neo4j.Driver {
	uri := "bolt://localhost:7687"
	username := "neo4j"
	password := "sourcecrawler"
	var (
		err    error
		driver neo4j.Driver
	)
	useConsoleLogger := func(level neo4j.LogLevel) func(config *neo4j.Config) {
		return func(config *neo4j.Config) {
			config.Log = neo4j.ConsoleLogger(level)
			config.Encrypted = false
		}
	}
	driver, err = neo4j.NewDriver(uri, neo4j.BasicAuth(username, password, ""), useConsoleLogger(neo4j.WARNING))
	if err != nil {
		panic(err)
	}

	return driver
}
