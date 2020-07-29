package handler

import (
	"encoding/json"
	"fmt"
	"os"
	"sourcecrawler/app/helper"
	"sourcecrawler/app/unsafe"
	"strings"

	"github.com/mitchellh/go-z3"

	"go/ast"
	"go/printer"
	"net/http"
	"regexp"
	"sourcecrawler/app/cfg"
	"sourcecrawler/app/model"
	_ "strings" //

	"github.com/jinzhu/gorm"
)

func UnsafeEndpoint(db *gorm.DB, w http.ResponseWriter, r *http.Request) {
	request := struct {
		X   int    `json:"x"`
		Msg string `json:"msg"`
	}{}

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&request); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	defer r.Body.Close()

	fmt.Println("ERROR")

	messages, err := unsafe.Unsafe(request.X, request.Msg)

	if err != nil {
		respondJSON(w, http.StatusOK, messages)
	} else {
		respondJSON(w, http.StatusBadRequest, nil)
	}
}

// NOTE: Values can be hardcoded in here for testing (Can't run from Postman without additional docker configuration)
//  Run the following: curl -X POST http://127.0.0.1:3000/container
func ContainerEndpoint(db *gorm.DB, w http.ResponseWriter, r *http.Request) {
	projectRoot := "./"
	fmt.Println("container endpoint")
	respondJSON(w, http.StatusOK, projectRoot)
}

//Slices the program - first parses the stack trace, and then parses the project for log calls
// -Afterwards it creates the CFG and attempts to connect each of the functions in the stack trace
func SliceProgram(db *gorm.DB, w http.ResponseWriter, r *http.Request) {
	request := struct {
		StackTrace  string   `json:"stackTrace"`
		LogMessages []string `json:"logMessages"` //it holds raw log statements
		ProjectRoot string   `json:"projectRoot"`
	}{}

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&request); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	defer r.Body.Close()

	//1 -- parse stack trace for functions that led to exception
	parsedStack := helper.ParsePanic(request.ProjectRoot, request.StackTrace)

	//2 -- Parse project for log statements with regex + line + file name
	logTypes := helper.ParseProject(request.ProjectRoot)

	// Matching log messages to a regex (only returns used regexes)
	seenLogTypes := []model.LogType{}
	for _, msg := range request.LogMessages {
		for _, value := range logTypes {
			matched, _ := regexp.MatchString(value.Regex, msg)
			if matched {
				seenLogTypes = append(seenLogTypes, value)
				//fmt.Println("Valid regexes:", value.Regex)
				break
			}
		}
	}

	topLevelWrapper := cfg.SetupPersistentData(request.ProjectRoot)

	stack := parsedStack //likely only one stack trace
	entryPackage := stack.PackageName[len(stack.FileName)-1]
	entryName := stack.FuncName[len(stack.FuncName)-1]

	//grab the entry function
	var entryFnNode ast.Node
	for _, file := range topLevelWrapper.ASTs {
		if strings.Contains(file.Name.Name, entryPackage) {
			for _, decl := range file.Decls {
				if decl, ok := decl.(*ast.FuncDecl); ok {
					if strings.EqualFold(decl.Name.Name, entryName) {
						entryFnNode = decl
						break
					}
				}
			}
		}
	}

	//expand the cfg
	entryWrapper := cfg.NewFnWrapper(entryFnNode, nil)
	entryWrapper.SetOuterWrapper(topLevelWrapper)
	cfg.ExpandCFG(entryWrapper)

	//find the block originating the exception
	exceptionBlock := cfg.FindPanicWrapper(entryWrapper, &stack)

	pathList := cfg.CreateNewPath()

	//label the tree starting from the exception block
	pathList.LabelCFG(exceptionBlock, seenLogTypes, entryWrapper, stack)

	//rename variables to ssa form
	cfg.ConvertCFGtoSSAForm(entryWrapper)

	//gather the paths
	paths := pathList.TraverseCFG(exceptionBlock, entryWrapper)

	for i, path := range paths {
		fmt.Println("PATH", i+1)
		for _, expr := range path.Expressions {
			printer.Fprint(os.Stdout, topLevelWrapper.Fset, expr)
			fmt.Println()
		}
		fmt.Println()
	}

	//transform to z3
	config := z3.NewConfig()
	ctx := z3.NewContext(config)
	config.Close()
	defer ctx.Close()

	s := ctx.NewSolver()
	defer s.Close()

	//solve and display each path
	assignments := make([]map[string]*z3.AST, 0)
	for _, path := range paths {
		var z3group *z3.AST
		for _, expr := range path.Expressions {
			z3group = cfg.ConvertExprToZ3(ctx, expr, topLevelWrapper.Fset)
			if z3group != nil {
				s.Assert(z3group)
			}
		}

		if v := s.Check(); v != z3.True {
			fmt.Println("Unsolvable")
			continue
		}
		m := s.Model()
		newAssignments := m.Assignments()
		cfg.FilterToUserInput(exceptionBlock, path.Expressions, newAssignments)
		assignments = append(assignments, newAssignments)

		for name, val := range newAssignments {
			fmt.Printf("%s = %s\n", name, val)
		}
		fmt.Println()

		m.Close()
	}

}
