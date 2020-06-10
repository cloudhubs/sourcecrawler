package db

import (
	"fmt"

	"github.com/neo4j/neo4j-go-driver/neo4j"
)

func doThaTest() {
	greeting, err := helloWorld("bolt://localhost:7687", "neo4j", "password")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println(greeting)
}

func connect() (neo4j.Session, neo4j.Driver) {
	uri := "bolt://localhost:7687"
	username := "neo4j"
	password := "sourcecrawler"
	var (
		err      error
		driver   neo4j.Driver
		session  neo4j.Session
		result   neo4j.Result
		greeting interface{}
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
	session, err = driver.Session(neo4j.AccessModeWrite)
	if err != nil {
		panic(err)
	}

	return session, driver
}
