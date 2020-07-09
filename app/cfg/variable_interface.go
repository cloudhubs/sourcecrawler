package cfg

import (
	"fmt"
	"go/ast"
)

type VariableWrapper interface {
	//This is the identifier
	SetName(name string)
	GetName() string

	//This is the value associated with the variable
	// which will either be another variable, function,
	// or value
	//TODO: define which types are valid for each concrete type
	SetValue(value interface{})
	GetValue() interface{}
}

func (f FnVariableWrapper) SetName(name string) {
	f.Name = name
}

func (f FnVariableWrapper) GetName() string {
	return f.Name
}

//TODO: the input should be a FnWrapper
// or a reference to another FnVariableWrapper
func (f FnVariableWrapper) SetValue(value interface{}) {
	f.Value = value
}

func (f FnVariableWrapper) GetValue() interface{} {
	return f.Value
}

// this concrete type represents variables that hold functions
// (literal and named)
type FnVariableWrapper struct {
	Value interface{} //should be VariableConnector or FnWrapper if assigned
	Name string //Can be swapped for ast.Ident
	//Ident ast.Ident
}

//------------------ VariableWrapper helper functions -------------
//Function to search for literals or assignment (should be run at parse-time given an AST node)
func SearchFuncLits(node ast.Node) []VariableWrapper{
	variables := []VariableWrapper{}

	ast.Inspect(node, func(currNode ast.Node) bool {

		return true
	})

	switch fn := node.(type) {
	case *ast.AssignStmt:
		varName := GetVarName(fn)

		//Need to look at rhs expr if it's an assignment (assuming theres 1 expr in Rhs)
		varVal := GetVarWrap(fn.Rhs[0])
		varWrap := &FnVariableWrapper{
			Value: varVal,
			Name:  varName,
		}
		//If name and value are valid, then add to list
		if varWrap.GetName() != ""&& varWrap.GetValue() != nil{
			variables = append(variables, varWrap)
		}

		//TODO: parse expression for the Identifier
		left := fn.Lhs[0]
		right := fn.Rhs[0]

		if id, ok := left.(*ast.Ident); ok {
			fmt.Println("ident left",id)
		}
		if call, ok := left.(*ast.CallExpr); ok {
			fmt.Println("callexpr left",call)
		}

		switch temp := left.(type){
		case *ast.CallExpr:
			fmt.Println("Expr x is callexpr", temp)
		case *ast.Ident:
			fmt.Println("is ident left", temp)
		}

		//right---
		if id, ok := right.(*ast.Ident); ok {
			fmt.Println("ident right",id)
		}
		if call, ok := right.(*ast.CallExpr); ok {
			fmt.Println("ex right",call)
		}

		switch temp := right.(type){
		case *ast.CallExpr:
			fmt.Println("Expr x is callexpr right", temp)
		case *ast.Ident:
			fmt.Println("is ident right", temp)
		}

		//if expr, ok := node.(*ast.ExprStmt); ok {
		//	fmt.Println("exprStmt",expr)
		//	switch temp := expr.X.(type){
		//	case *ast.CallExpr:
		//		fmt.Println("Expr x is callexpr", temp)
		//	case *ast.Ident:
		//		fmt.Println("is ident", temp)
		//	}
		//}

	case *ast.FuncLit:
		fmt.Println("Func lit", fn.Type, fn.Body)
		if expr, ok := node.(*ast.CallExpr); ok{
			fmt.Println("Is also expression", expr)
		}
		//varVal := &FnVariableWrapper{
		//	Value: Get,
		//	Name:  "",
		//}
	//case *ast.Ident:
	//	fmt.Println("Ident name", fn.Name)
	}

	return variables
}

//Helper function for extracting actual variable value -- used in
func GetVarWrap(node ast.Node) interface{}{

	var varValue interface{}

	//Basic literal --> just the raw value (string literal, primitives, etc)
	if basicLit, ok := node.(*ast.BasicLit); ok{
		varValue = basicLit.Value
		fmt.Println("basic lit", basicLit)
	}

	if id, ok := node.(*ast.Ident); ok {
		fmt.Println("ident",id)
	}

	//TODO: handle func lits
	if funcLit, ok := node.(*ast.FuncLit); ok{
		fmt.Println("func lit",funcLit)
	}

	//TODO: func decl
	if funcDecl, ok := node.(*ast.FuncDecl); ok{
		fmt.Println("func decl", funcDecl)
	}

	return varValue
}




