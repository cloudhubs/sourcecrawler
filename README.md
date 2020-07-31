# Source Crawler version 1.0.0
A REST API to parse a single Go project and create a program slice based on found log statements and a given stack trace, and suggest integer inputs that can recreate the executed path. Project template based on [this sample Go API](https://github.com/mingrammer/go-todo-rest-api-example)

## Installation & Run
Clone this repository

```bash
git clone https://github.com/cloudhubs/sourcecrawler
```

Build and run the project:

```bash
go get
go build
./sourcecrawler
# API now available on http://127.0.0.1:3000
# Slice endpoint: /slice
```

## API

#### /slicer
* `POST` : Parse a project to extract its log types
    - Request format:
```
    {
        "stackTrace": "", // stack trace escaped for JSON
        "logMessages": ["message", "message2"], // array of collected log messages
        "projectRoot": "/path/to/project" // path to project to be sliced
    }
```
