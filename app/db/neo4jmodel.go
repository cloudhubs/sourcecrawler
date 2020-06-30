package db

import (
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
)

type ExecutionLabel int

const (
	NoLabel ExecutionLabel = iota
	Must
	May
	MustNot
)

func (s ExecutionLabel) String() string {
	return [...]string{"NoLabel", "Must", "May", "MustNot"}[s]
}

type Node interface {
	/*
		Returns a map that contains the child nodes as keys, and the relationship's labels as the value
	*/
	GetChildren() map[Node]string

	SetChild([]Node)

	GetParents() []Node

	SetParents(parent Node)
	/*
		Returns a string that contains this node's properties, in cypher's key-value format
	*/
	GetProperties() string

	GetNodeType() string

	GetFilename() string

	GetLineNumber() int

	GetLabel() ExecutionLabel

	SetLabel(l ExecutionLabel)

	SetFilename(filename string)

	SetLineNumber(line int)
}

type FunctionNode struct {
	Filename     string
	LineNumber   int
	FunctionName string
	Child        Node
	Parent       Node
	Label        ExecutionLabel
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
	Parent       Node
	Label        ExecutionLabel
}

// in the event of return fnCall() a FunctionNode will be its predecessor node
// in the event of return fnCall(), fnCall2(), err two FunctionNodes precede it in the CFG
// in the event of return varName, err nothing precedes it
type ReturnNode struct {
	Filename   string
	LineNumber int
	Expression string
	Child      Node
	Parent     Node
	Label      ExecutionLabel
}

type StatementNode struct {
	Filename   string
	LineNumber int
	LogRegex   string
	Child      Node
	Parent     Node
	Label      ExecutionLabel
}

type ConditionalNode struct {
	Filename   string
	LineNumber int
	Condition  string
	TrueChild  Node
	FalseChild Node
	Parent     Node
	Label      ExecutionLabel
}

type EndConditionalNode struct {
	Child   Node
	Parents []Node
	Label   ExecutionLabel
}

type VariableNode struct {
	ScopeId string
	VarName string
	Filename   string
	LineNumber int
	Value string //should hold an expression
	Parent Node
	Child Node
	ValueFromParent bool
}

// STATEMENT NODES

func (n *StatementNode) GetChildren() map[Node]string {
	if n.Child == nil {
		return map[Node]string{}
	}
	var m = map[Node]string{
		n.Child: "",
	}
	return m
}

func (n *StatementNode) SetChild(c []Node) {
	n.Child = c[0]
}

func (n *StatementNode) GetParents() []Node {
	if n.Parent == nil {
		return []Node{}
	}
	return []Node{n.Parent}
}

func (n *StatementNode) SetParents(parent Node) {
	n.Parent = parent
}

func (n *StatementNode) GetProperties() string {
	val := fmt.Sprintf("filename: \"%v\", linenumber: %v, label: %v", n.Filename, n.LineNumber, n.Label)
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

func (n *StatementNode) GetLabel() ExecutionLabel {
	return n.Label
}

func (n *StatementNode) SetLabel(l ExecutionLabel) {
	n.Label = l
}

func (n *StatementNode) SetFilename(filename string) {
	n.Filename = filename
}

func (n *StatementNode) SetLineNumber(line int) {
	n.LineNumber = line
}

// CONDITIONAL NODES

func (n *ConditionalNode) GetChildren() map[Node]string {
	if n.TrueChild == nil && n.FalseChild == nil {
		return map[Node]string{}
	}
	var m = map[Node]string{
		n.TrueChild:  `{ takeIf: "true" }`,
		n.FalseChild: `{ takeIf: "false" }`,
	}
	return m
}

func (n *ConditionalNode) SetChild(c []Node) {
	n.TrueChild = c[0]
	n.FalseChild = c[1]
}

func (n *ConditionalNode) GetParents() []Node {
	if n.Parent == nil {
		return []Node{}
	}
	return []Node{n.Parent}
}

func (n *ConditionalNode) SetParents(parent Node) {
	n.Parent = parent
}

func (n *ConditionalNode) GetProperties() string {
	val := fmt.Sprintf("filename: \"%v\", linenumber: %v, condition: \"%v\", label: %v",
		n.Filename, n.LineNumber, n.Condition, n.Label)
	return "{ " + val + " }"
}

func (n *ConditionalNode) GetNodeType() string {
	return ":CONDITIONAL:STATEMENT"
}

func (n *ConditionalNode) GetFilename() string {
	return n.Filename
}

func (n *ConditionalNode) GetLabel() ExecutionLabel {
	return n.Label
}

func (n *ConditionalNode) SetLabel(l ExecutionLabel) {
	n.Label = l
}

func (n *ConditionalNode) GetLineNumber() int {
	return n.LineNumber
}

func (n *ConditionalNode) SetFilename(filename string) {
	n.Filename = filename
}

func (n *ConditionalNode) SetLineNumber(line int) {
	n.LineNumber = line
}

// FUNCTION (CALL) NODES

func (n *FunctionNode) GetChildren() map[Node]string {
	if n.Child == nil {
		return map[Node]string{}
	}
	var m = map[Node]string{
		n.Child: "",
	}
	return m
}

func (n *FunctionNode) SetChild(c []Node) {
	n.Child = c[0]
}

func (n *FunctionNode) GetParents() []Node {
	if n.Parent == nil {
		return []Node{}
	}
	return []Node{n.Parent}
}

func (n *FunctionNode) SetParents(parent Node) {
	n.Parent = parent
}

func (n *FunctionNode) GetProperties() string {
	val := fmt.Sprintf("filename: \"%v\", linenumber: %v, function: \"%v\", label: %v",
		n.Filename, n.LineNumber, n.FunctionName, n.Label)
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

func (n *FunctionNode) GetLabel() ExecutionLabel {
	return n.Label
}

func (n *FunctionNode) SetLabel(l ExecutionLabel) {
	n.Label = l
}

func (n *FunctionNode) SetFilename(filename string) {
	n.Filename = filename
}

func (n *FunctionNode) SetLineNumber(line int) {
	n.LineNumber = line
}

// FUNCTION DECLARATION NODES

func (n *FunctionDeclNode) GetChildren() map[Node]string {
	if n.Child == nil {
		return map[Node]string{}
	}
	var m = map[Node]string{
		n.Child: "",
	}
	return m
}
func (n *FunctionDeclNode) SetChild(c []Node) {
	n.Child = c[0]
}

func (n *FunctionDeclNode) GetParents() []Node {
	if n.Parent == nil {
		return []Node{}
	}
	return []Node{n.Parent}
}

func (n *FunctionDeclNode) SetParents(parent Node) {
	n.Parent = parent
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

	val := fmt.Sprintf("filename: \"%v\", linenumber: %v, function: \"%v\", receivers: \"%v\", parameters: \"%v\", returns: \"%v\", label: %v",
		n.Filename, n.LineNumber, n.FunctionName, string(rcv), string(params), string(returns), n.Label)
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

func (n *FunctionDeclNode) GetLabel() ExecutionLabel {
	return n.Label
}

func (n *FunctionDeclNode) SetLabel(l ExecutionLabel) {
	n.Label = l
}

func (n *FunctionDeclNode) SetFilename(filename string) {
	n.Filename = filename
}

func (n *FunctionDeclNode) SetLineNumber(line int) {
	n.LineNumber = line
}

// RETURN NODES

func (n *ReturnNode) GetChildren() map[Node]string {
	if n.Child == nil {
		return map[Node]string{}
	}
	var m = map[Node]string{
		n.Child: "",
	}
	return m
}

func (n *ReturnNode) SetChild(c []Node) {
	n.Child = c[0]
}

func (n *ReturnNode) GetParents() []Node {
	if n.Parent == nil {
		return []Node{}
	}
	return []Node{n.Parent}
}

func (n *ReturnNode) SetParents(parent Node) {
	n.Parent = parent
}

func (n *ReturnNode) GetProperties() string {
	val := fmt.Sprintf("filename: \"%v\", linenumber: %v, expression: \"%v\", label: %v",
		n.Filename, n.LineNumber, n.Expression, n.Label)
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

func (n *ReturnNode) GetLabel() ExecutionLabel {
	return n.Label
}

func (n *ReturnNode) SetLabel(l ExecutionLabel) {
	n.Label = l
}

func (n *ReturnNode) SetFilename(filename string) {
	n.Filename = filename
}

func (n *ReturnNode) SetLineNumber(line int) {
	n.LineNumber = line
}

// END CONDITIONAL NODES

func (n *EndConditionalNode) GetChildren() map[Node]string {
	if n.Child == nil {
		return map[Node]string{}
	}
	var m = map[Node]string{
		n.Child: "",
	}
	return m
}

func (n *EndConditionalNode) SetChild(c []Node) {
	n.Child = c[0]
}

func (n *EndConditionalNode) GetParents() []Node {
	if n.Parents == nil {
		return []Node{}
	}
	return n.Parents
}

func (n *EndConditionalNode) SetParents(parent Node) {
	if n.Parents == nil {
		n.Parents = []Node{}
	}
	n.Parents = append(n.Parents, parent)
}

func (n *EndConditionalNode) GetProperties() string {
	return ""
}

func (n *EndConditionalNode) GetNodeType() string {
	return ":ENDCONDITIONAL:STATEMENT"
}

func (n *EndConditionalNode) GetFilename() string {
	return ""
}

func (n *EndConditionalNode) GetLabel() ExecutionLabel {
	return n.Label
}

func (n *EndConditionalNode) SetLabel(l ExecutionLabel) {
	n.Label = l
}

func (n *EndConditionalNode) GetLineNumber() int {
	return 0
}

func (n *EndConditionalNode) SetFilename(filename string) {

}

func (n *EndConditionalNode) SetLineNumber(line int) {

}

//VARIABLE NODES
func (n *VariableNode) GetChildren() Node {
	return n.Child
}

func (n *VariableNode) SetChild(c Node) {
	n.Child = c
}

func (n *VariableNode) GetParents() Node {
	return n.Parent
}

func (n *VariableNode) SetParents(parent Node) {
	n.Parent = parent
}

func (n *VariableNode) GetProperties() string {
	return fmt.Sprintf("Variable node %d %s %s %s %v",
		n.LineNumber, n.Filename, n.Value, n.ScopeId, n.Parent)
}

func (n *VariableNode) GetNodeType() string {
	return ":VARIABLE:STATEMENT"
}

func (n *VariableNode) GetFilename() string {
	return n.Filename
}

func (n *VariableNode) GetLabel() ExecutionLabel {
	return ExecutionLabel(MustNot)
}

func (n *VariableNode) SetLabel(l ExecutionLabel) {

}

func (n *VariableNode) GetLineNumber() int {
	return n.LineNumber
}

func (n *VariableNode) SetFilename(filename string) {
	n.Filename = filename
}

func (n *VariableNode) SetLineNumber(line int) {
	n.LineNumber = line
}
