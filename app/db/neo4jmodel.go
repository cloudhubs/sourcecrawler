package db

import (
	"fmt"
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
}

type FunctionNode struct {
	Filename     string
	LineNumber   int
	FunctionName string
	Child        Node
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

// statement nodes

func (n *StatementNode) GetChildren() map[Node]string {
	var m = map[Node]string{
		n.Child: "",
	}
	return m
}

func (n *StatementNode) GetProperties() string {
	val := fmt.Sprintf("filename: \"%v\", linenumber: \"%v\"", n.Filename, n.LineNumber)
	if n.LogRegex != "" {
		val = val + fmt.Sprintf(", logregex: \"%v\"", n.LogRegex)
	}
	return "{ " + val + " }"
}

func (n *StatementNode) GetNodeType() string {
	return ":STATEMENT"
}

// conditional nodes

func (n *ConditionalNode) GetChildren() map[Node]string {
	var m = map[Node]string{
		n.TrueChild:  `{ takeIf: "true" }`,
		n.FalseChild: `{ takeIf: "false" }`,
	}
	return m
}

func (n *ConditionalNode) GetProperties() string {
	val := fmt.Sprintf("filename: \"%v\", linenumber: \"%v\", condition: \"%v\"", n.Filename, n.LineNumber, n.Condition)
	return "{ " + val + " }"
}

func (n *ConditionalNode) GetNodeType() string {
	return ":STATEMENT:CONDITIONAL"
}

func (n *FunctionNode) GetChildren() map[Node]string {
	var m = map[Node]string{
		n.Child: "",
	}
	return m
}

func (n *FunctionNode) GetProperties() string {
	val := fmt.Sprintf("filename: \"%v\", linenumber: \"%v\", function: \"%v\"", n.Filename, n.LineNumber, n.FunctionName)
	return "{ " + val + " }"
}

func (n *FunctionNode) GetNodeType() string {
	return ":STATEMENT:CONDITIONAL:FUNCTIONCALL"
}
