# Source Crawler version 0.0.1
A REST API to parse a single Go project, find log events in it, and match log messages to their source statement. Project template based on [this sample Go API](https://github.com/mingrammer/go-todo-rest-api-example)

## Installation & Run
Clone this repository

```bash
git clone https://github.com/cloudhubs/sourcecrawler
```

Make sure you have MySQL running locally, and create the database and user as follows (in the mysql shell):

```
CREATE USER 'sourcecrawler'@'localhost' IDENTIFIED BY 'password';
CREATE DATABASE sourcecrawler;
GRANT ALL ON sourcecrawler.* TO 'sourcecrawler'@'localhost';
```

Build and run the project:

```bash
go get
go build
./sourcecrawler
# API now available on http://127.0.0.1:3000
```

## Structure
```
├── app
│   ├── app.go			 		// API routes registered here
│   ├── handler          		// Handler functions for API routes
│   │   ├── common.go    		// Common response functions
│   │   ├── handlers.go  		// Handler functions, should call functions from logMatcher.go and projectParser.go
│   │   ├── logMatcher.go  		// Functions to match a log message to its source
│   │   └── projectParser.go   	// Functions to parse a project and find its log types
│   └── model
│       └── model.go     		// Models for our application, including log types and the request/response formats
├── config
│   └── config.go        		// Database configuration
├── go.mod						// Go module stuff
├── go.sum						// More Go module stuff
└── main.go
```

## API

#### /parser
* `GET` : Get all found log types
* `POST` : Parse a project to extract its log types

#### /matcher
* `POST` : Find the source line of a log message
