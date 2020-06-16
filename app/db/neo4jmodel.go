package db

import (
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog/log"
)

type Node interface {
	/*
		Returns a map that contains the child nodes as keys, and the relationship's labels as the value
	*/
	GetChildren() map[Node]string

	/*
		Returns a string that contains this node's properties, in cypher's key-value format
	*/
	GetProperties() string

	GetNodeType() string

	GetFilename() string

	GetLineNumber() int
}

type FunctionNode struct {
	Filename     string
	LineNumber   int
	FunctionName string
	Child        Node
}

type Return struct {
	Name       string
	ReturnType string
}

type FunctionDeclNode struct {
	Filename     string
	LineNumber   int
	FunctionName string
	Receivers    map[string]string // for methods, name to type
	Params       map[string]string // map of arg name to type
	Returns      []Return          // not a map since you don't have to name return variables
	Child        Node
}

// in the event of return fnCall() a FunctionNode will be its predecessor node
// in the event of return fnCall(), fnCall2(), err two FunctionNodes precede it in the CFG
// in the event of return varName, err nothing precedes it
type ReturnNode struct {
	Filename   string
	LineNumber int
	Expression string
}

type StatementNode struct {
	Filename   string
	LineNumber int
	LogRegex   string
	Child      Node
}

type ConditionalNode struct {
	Filename   string
	LineNumber int
	Condition  string
	TrueChild  Node
	FalseChild Node
}

// STATEMENT NODES

func (n *StatementNode) GetChildren() map[Node]string {
	var m = map[Node]string{
		n.Child: "",
	}
	return m
}

func (n *StatementNode) GetProperties() string {
	val := fmt.Sprintf("filename: \"%v\", linenumber: %v", n.Filename, n.LineNumber)
	if n.LogRegex != "" {
		val = val + fmt.Sprintf(", logregex: \"%v\"", n.LogRegex)
	}
	return "{ " + val + " }"
}

func (n *StatementNode) GetNodeType() string {
	return ":STATEMENT"
}

func (n *StatementNode) GetFilename() string {
	return n.Filename
}

func (n *StatementNode) GetLineNumber() int {
	return n.LineNumber
}

// CONDITIONAL NODES

func (n *ConditionalNode) GetChildren() map[Node]string {
	var m = map[Node]string{
		n.TrueChild:  `{ takeIf: "true" }`,
		n.FalseChild: `{ takeIf: "false" }`,
	}
	return m
}

func (n *ConditionalNode) GetProperties() string {
	val := fmt.Sprintf("filename: \"%v\", linenumber: %v, condition: \"%v\"", n.Filename, n.LineNumber, n.Condition)
	return "{ " + val + " }"
}

func (n *ConditionalNode) GetNodeType() string {
	return ":CONDITIONAL:STATEMENT"
}

func (n *ConditionalNode) GetFilename() string {
	return n.Filename
}

func (n *ConditionalNode) GetLineNumber() int {
	return n.LineNumber
}

// FUNCTION (CALL) NODES

func (n *FunctionNode) GetChildren() map[Node]string {
	var m = map[Node]string{
		n.Child: "",
	}
	return m
}

func (n *FunctionNode) GetProperties() string {
	val := fmt.Sprintf("filename: \"%v\", linenumber: %v, function: \"%v\"", n.Filename, n.LineNumber, n.FunctionName)
	return "{ " + val + " }"
}

func (n *FunctionNode) GetNodeType() string {
	return ":FUNCTIONCALL:STATEMENT"
}

func (n *FunctionNode) GetFilename() string {
	return n.Filename
}

func (n *FunctionNode) GetLineNumber() int {
	return n.LineNumber
}

// FUNCTION DECLARATION NODES

func (n *FunctionDeclNode) GetChildren() map[Node]string {
	var m = map[Node]string{
		n.Child: "",
	}
	return m
}

func (n *FunctionDeclNode) GetProperties() string {
	rcv, err := json.Marshal(n.Receivers)
	if err != nil {
		rcv = []byte{}
		log.Warn().Msg("could not marshal receivers")
	}

	params, err := json.Marshal(n.Params)
	if err != nil {
		params = []byte{}
		log.Warn().Msg("could not marshal params")
	}

	returns, err := json.Marshal(n.Returns)
	if err != nil {
		params = []byte{}
		log.Warn().Msg("could not marshal returns")
	}

	val := fmt.Sprintf("filename: \"%v\", linenumber: %v, function: \"%v\", receivers: \"%v\", parameters: \"%v\", returns: \"%v\"", n.Filename, n.LineNumber, n.FunctionName, string(rcv), string(params), string(returns))
	return "{ " + val + " }"
}

func (n *FunctionDeclNode) GetNodeType() string {
	return ":FUNCTIONDECL:STATEMENT"
}

func (n *FunctionDeclNode) GetFilename() string {
	return n.Filename
}

func (n *FunctionDeclNode) GetLineNumber() int {
	return n.LineNumber
}

// RETURN NODES

func (n *ReturnNode) GetChildren() map[Node]string {
	return nil
}

func (n *ReturnNode) GetProperties() string {
	val := fmt.Sprintf("filename: \"%v\", linenumber: %v, expression: \"%v\"", n.Filename, n.LineNumber, n.Expression)
	return "{ " + val + " }"
}

func (n *ReturnNode) GetNodeType() string {
	return ":RETURN:STATEMENT"
}

func (n *ReturnNode) GetFilename() string {
	return n.Filename
}

func (n *ReturnNode) GetLineNumber() int {
	return n.LineNumber
}
