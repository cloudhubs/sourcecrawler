package helper

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"runtime"
	"strings"

	"github.com/rs/zerolog/log"
)

//Stores the file path and line # (Node pointers there for extra info)
type panicStruct struct {
	node     ast.Node
	pd       *ast.CallExpr
	filePath string
	lineNum  string
}

//Parsing a panic runtime stack trace (Id, messageLevel, file name and line #, function name)
// -the fileName + LineNum + funcName will be stored in parallel arrays - same index
type StackTraceStruct struct {
	Id       int
	MsgLevel string
	FileName []string
	LineNum  []string
	FuncName []string
}

//Helper function to grab OS separator
func GrabOS() string {
	if runtime.GOOS == "windows" {
		return "\\"
	} else {
		return "/"
	}
}

//Parse through a panic message and find originating file/line number/function name
// Takes in a string of the stack trace error and parse thru
// ** Assuming that the stack trace message ends with a \n **
func ParsePanic(projectRoot string, stackMessage string) []StackTraceStruct {

	//Generates test stack traces (run once and redirect to log file)
	// "go run main.go 2>stackTrace.log"
	//testCondPanic(15)
	//testPanic()

	//Grab separator
	separator := GrabOS()

	//Grab files to parse, split stack trace string, get project root
	filesToParse := GatherGoFiles(projectRoot)
	stackStrs := splitStackTraceString(stackMessage)

	//Helper map for quick lookup
	localFilesMap := make(map[string]string)
	for index := range filesToParse {
		shortFileName := filesToParse[index]
		shortFileName = shortFileName[strings.LastIndex(shortFileName, separator)+1:]
		localFilesMap[shortFileName] = "exists"
	}

	//Helper map for quick function lookup
	functionsMap := functionDeclsMap(filesToParse)

	//Open stack trace log file (assume there will be a log file named this)
	//file, err := os.Open("stackTrace.log")
	//if err != nil {
	//	fmt.Println("Error opening file")
	//}

	//Parse through stack trace log file
	//scanner := bufio.NewScanner(file)
	stackTrc := []StackTraceStruct{}
	tempStackTrace := StackTraceStruct{
		Id:       1,
		MsgLevel: "",
		FileName: []string{},
		LineNum:  []string{},
		FuncName: []string{},
	}
	fileLineNum := 1
	id := 1
	tempFuncName := ""
	functionFound := false

	//Scan through each line of log file and do analysis
	for index := range stackStrs {
		logStr := stackStrs[index]

		//Check for beginning of new stack trace statement (create new trace struct for new statement)
		// keyword "serving" is found in the first line of each new stack trace
		if strings.Contains(logStr, "serving") {

			//Make sure attributes aren't empty before adding it
			if tempStackTrace.MsgLevel != "" && len(tempStackTrace.FileName) != 0 &&
				len(tempStackTrace.LineNum) != 0 && len(tempStackTrace.FuncName) != 0 {
				tempStackTrace.Id = id
				stackTrc = append(stackTrc, tempStackTrace)
				id++
			}

			//New statement trace
			tempStackTrace = StackTraceStruct{
				Id:       id,
				MsgLevel: "",
				FileName: []string{},
				LineNum:  []string{},
				FuncName: []string{},
			}

			//Assign panic type
			if strings.Contains(logStr, "panic") {
				tempStackTrace.MsgLevel = "panic"
			}
		}

		//Read the function line first
		//!-- NOTE: assuming there is no function named go() --!
		//Process function name lines (doesn't contain .go)
		if !strings.Contains(logStr, ".go") &&
			strings.Contains(logStr, "(") && strings.Contains(logStr, ")") {

			functionFound = false

			startIndex := strings.LastIndex(logStr, ".")
			endIndex := strings.LastIndex(logStr, "(")

			//Functions with multiple calls (has multiple . operators)
			if startIndex != -1 && endIndex != -1 && startIndex < endIndex {
				tempFuncName = logStr[startIndex+1 : endIndex]
			}

			//Function is a single standalone function (only example currently is panic())
			if startIndex == -1 {
				tempFuncName = logStr[:endIndex]
			}

			//No parenthesis for function name OR a custom function(ex: .Serve or .callOtherPanic(...))
			if startIndex > endIndex {
				//Function with (...) as args (these are the ones that we are interested in) -- grab first on stack
				if strings.Contains(logStr, "...") {
					specialIndex := strings.Index(logStr, ".")
					tempFuncName = logStr[specialIndex+1 : endIndex]
				} else {
					tempFuncName = logStr[startIndex+1:]
				}
			}

			//Case where a function returns another function (will have two sets of parens)
			if strings.Index(logStr, "(") != strings.LastIndex(logStr, "(") {
				tempStr := logStr[strings.Index(logStr, ")"):strings.LastIndex(logStr, "(")]
				firstNdx := strings.Index(tempStr, ".") + 1

				//If more than 1 function call together, then grab the first
				if strings.Count(tempStr, ".") > 1 {
					tempFuncName = tempStr[firstNdx:strings.LastIndex(tempStr, ".")]
				} else {
					tempFuncName = tempStr[firstNdx:]
				}
			}

			//If found, add to list of function names
			// bug with app.go function -- inside handleRequest issue with returning a function
			if _, found := functionsMap[tempFuncName]; found {
				//fmt.Println("The function ", tempFuncName, "is in the local files", functionsMap[tempFuncName])
				if !strings.Contains(logStr, "testing.") && !strings.Contains(logStr, ".tRunner") {
					tempStackTrace.FuncName = append(tempStackTrace.FuncName, tempFuncName)
					functionFound = true
				}
			}
			//fmt.Println("funct name:", tempFuncName)
		}

		//Read the file name/line number line - ONLY if a corresponding function is found
		if functionFound && strings.Contains(logStr, ".go") {
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
				tempStackTrace.FileName = append(tempStackTrace.FileName, fileName)
				tempStackTrace.LineNum = append(tempStackTrace.LineNum, lineNum)
				//fmt.Println("Matching file+lines: ", fileName,LineNum)
			}
		}

		//Update file line number
		fileLineNum++
	}

	//Add last entry
	stackTrc = append(stackTrc, tempStackTrace)

	//Print struct
	//printErrorList(stackTrc)

	return stackTrc
}

func splitStackTraceString(sts string) []string {
	return strings.Split(sts, "\n")
}

//Helper function to test print parsed info from stack trace
func printErrorList(stackTrc []StackTraceStruct) {
	for _, val := range stackTrc {
		//Should have same # of elements
		for index := range val.FileName {
			fmt.Printf("Depth: %d %s %s %s \n", index, val.FileName[index], val.LineNum[index], val.FuncName[index])
		}
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
