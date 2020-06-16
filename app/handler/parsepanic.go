package handler

import (
	"bufio"
	"fmt"
	"github.com/rs/zerolog/log"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"runtime"
	"strings"
)

//Stores the file path and line # (Node pointers there for extra info)
type panicStruct struct {
	node     ast.Node
	pd       *ast.CallExpr
	filePath string
	lineNum  string
}

//Parsing a panic runtime stack trace (id, messageLevel, file name and line #, function name)
// -the fileName + lineNum + funcName will be stored in parallel arrays - same index
type stackTraceStruct struct {
	id int
	msgLevel string
	fileName []string
	lineNum []string
	funcName []string
}

//Helper function to grab OS separator
func grabOS() string{
	if runtime.GOOS == "windows"{
		return "\\"
	}else{
		return "/"
	}
}

// Parse through a panic message and find originating file/line number/function name
//TODO:  take in a string of the stack trace error and parse thru
//TODO: add array inside the struct that holds each stack trace in order. return the struct
func parsePanic(projectRoot string, stackMessage string) []stackTraceStruct {

	//Generates test stack traces (run once and redirect to log file)
	// "go run main.go 2>stackTrace.log"
	//testCondPanic(15)
	//testPanic()

	//Grab separator
	separator := grabOS()

	//Grab files to parse
	filesToParse := gatherGoFiles(projectRoot)

	//Helper map for quick lookup
	localFilesMap := make(map[string]string)
	for index := range filesToParse{
		shortFileName := filesToParse[index]
		shortFileName = shortFileName[strings.LastIndex(shortFileName, separator)+1:]
		localFilesMap[shortFileName] = "exists"
	}

	//Helper map for quick function lookup
	functionsMap := functionDecls(filesToParse)

	//Open stack trace log file (assume there will be a log file named this)
	file, err := os.Open("stackTrace.log")
	if err != nil {
		fmt.Println("Error opening file")
	}

	//Parse through stack trace log file
	scanner := bufio.NewScanner(file)
	stackTrc := []stackTraceStruct{}
	tempStackTrace := stackTraceStruct{
		id:       1,
		msgLevel: "",
		fileName: []string{},
		lineNum:  []string{},
		funcName: []string{},
	}
	fileLineNum := 1
	id := 1
	//doneAdding := false
	tempFuncName := ""
	//doneFn := false

	//Scan through each line of log file and do analysis
	for scanner.Scan() {
		logStr := scanner.Text()

		//Check for beginning of new stack trace statement (create new trace struct for new statement)
		// keyword "serving" is found in the first line of each new stack trace
		if strings.Contains(logStr, "serving") {

			//Make sure attributes aren't empty before adding it
			if tempStackTrace.msgLevel != "" && len(tempStackTrace.fileName) != 0 &&
				len(tempStackTrace.lineNum) != 0 && len(tempStackTrace.funcName) != 0{
				tempStackTrace.id = id
				stackTrc = append(stackTrc, tempStackTrace)
				//doneAdding = false //status of adding file + line number
				//doneFn = false     //status of adding function name
				id++
			}

			//New statement trace
			tempStackTrace = stackTraceStruct{
				id:       id,
				msgLevel: "",
				fileName: []string{},
				lineNum:  []string{},
				funcName: []string{},
			}

			//Assign panic type
			if strings.Contains(logStr, "panic") {
				tempStackTrace.msgLevel = "panic"
			}

			//Read to skip the goroutine line
			//scanner.Scan()
		}

		//Read the function line first
		//!-- NOTE: assuming there is no function named go() --!
		//Process function name lines (doesn't contain .go)
		if  !strings.Contains(logStr, ".go") &&
			strings.Contains(logStr, "(") && strings.Contains(logStr, ")"){

			startIndex := strings.LastIndex(logStr, ".")
			endIndex := strings.LastIndex(logStr, "(")

			//Functions with multiple calls (has multiple . operators)
			if startIndex != -1 && endIndex != -1 && startIndex < endIndex{
				tempFuncName = logStr[startIndex+1:endIndex]
			}

			//Function is a single standalone function (only example currently is panic())
			if startIndex == -1{
				tempFuncName = logStr[:endIndex]
			}

			//No parenthesis for function name OR a custom function(ex: .Serve or .callOtherPanic(...))
			if startIndex > endIndex {
				//Function with (...) as args (these are the ones that we are interested in) -- grab first on stack
				if strings.Contains(logStr, "..."){
					specialIndex := strings.Index(logStr, ".")
					tempFuncName = logStr[specialIndex+1:endIndex]
					//Add the first origin function (should be the correct function where error occured)
					//if !doneFn {
					//	tempStackTrace.funcName = tempFuncName
					//	doneFn = true
					//}
				}else{
					tempFuncName = logStr[startIndex+1:]
				}
			}

			//Test print other function names
			//fmt.Println("Function name:", tempFuncName)

			//Test if a function is found
			//fmt.Println("funct name:", tempFuncName)
			if _, found := functionsMap[tempFuncName]; found{
				fmt.Println("The function ", tempFuncName, "is in the local files", functionsMap[tempFuncName])
			}
		}

		//Check if line contains a possible file name, store to map of fileName+LineNumber
		if strings.Contains(logStr, ".go") {
			fileName := logStr[strings.LastIndex(logStr, "/")+1 : strings.LastIndex(logStr, ":")]
			indxLineNumStart := strings.LastIndex(logStr, ":")
			lineNumLarge := logStr[indxLineNumStart+1:]

			//If space in line number string with +0xaa, etc
			var lineNum string
			if strings.Contains(lineNumLarge, " ") {
				lineNum = lineNumLarge[0:strings.Index(lineNumLarge, " ")]
			} else {
				lineNum = lineNumLarge
			}

			//Check for originating files where the exception was thrown (could be multiple files, parent calls, etc)
			// We only want to match local files and not any extraneous files
			if _, ok := localFilesMap[fileName]; ok {
				tempStackTrace.fileName = append(tempStackTrace.fileName, fileName)
				tempStackTrace.lineNum = append(tempStackTrace.lineNum, lineNum)
				//fmt.Println("Matching file+lines: ", fileName,lineNum)
			}
		}

		//Update file line number
		fileLineNum++
	}

	//for key, val := range functionsMap{
	//	fmt.Println(key, val)
	//}

	//Add last entry
	stackTrc = append(stackTrc, tempStackTrace)

	return stackTrc
}

//Helper function to test print parsed info from stack trace
func printErrorList(errorList []stackTraceStruct){
	//Test print the processed stack traces
	for _, value := range errorList {
		fmt.Printf("%d: %s in %s -- line %s from function %s\n",
			value.id, value.msgLevel, value.fileName, value.lineNum, value.funcName)
	}
}

//Finds all panic statements (not currently used, but may need later)
func findPanics(filesToParse []string) []panicStruct {

	panicList := []panicStruct{}

	for _, file := range filesToParse {
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			log.Error().Err(err).Msg("Error parsing file " + file)
		}

		//Inspect call expressions
		ast.Inspect(node, func(currNode ast.Node) bool {
			callExprNode, ok := currNode.(*ast.CallExpr)
			if ok {
				//If it's a panic statement, add to the struct
				if name := fmt.Sprint(callExprNode.Fun); name == "panic" {
					lnNum := fmt.Sprint(fset.Position(callExprNode.Pos()).Line)
					panicList = append(panicList, panicStruct{
						node:     currNode,
						pd:       callExprNode,
						filePath: file,
						lineNum:  lnNum,
					})
				}
			}
			return true
		})

		//Print file name/line number/panic
		for _, value := range panicList {
			fmt.Println(value.filePath, value.lineNum, fmt.Sprint(value.pd.Fun))
		}
	}
	return panicList
}

