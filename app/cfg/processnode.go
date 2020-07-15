package cfg

import (
	"fmt"
	"go/ast"
	"go/token"
	"sourcecrawler/app/logsource"
	"strings"

	"golang.org/x/tools/go/cfg"
)

//Method to get condition, nil if not a conditional (specific to block wrapper) - used in traverse function
func (b *BlockWrapper) GetCondition() string{

	var condition string = ""
	//Return block or panic, fatal, etc
	if len(b.Succs) == 0 {
		return ""
	}
	//Normal block
	if len(b.Succs) == 1 {
		return ""
	}
	//Conditional block
	if len(b.Succs) == 2 && b.Block != nil && len(b.Block.Nodes) > 0 {
		condNode := b.Block.Nodes[len(b.Block.Nodes)-1] //conditional is last node in a block

		ast.Inspect(condNode, func(currNode ast.Node) bool {

			//Get the expression string for the condition TODO: note -> extract actual expression node to be used in SMT solver
			if exprNode, ok := condNode.(ast.Expr); ok {
				condition = GetExprStr(exprNode)
				//fmt.Println("Expression node to be used in SMT solver", exprNode)

				//Init pathInstance if not initialized
				if PathInstance == nil{
					CreateNewPath()
				}

				//Add exprNode to list of expressions if not already in
				if _, ok := PathInstance.Expressions[exprNode]; ok{
					//fmt.Println(exprNode, " exists already")
				}else {
					PathInstance.Expressions[exprNode] = "exists"
				}
			}

			return true
		})
	}

	return condition
}

//Process the AST node to extract function literals (can be called in traverse or parse time)
func GetFuncLits(node ast.Node){
	varMap := make(map[string]string) //switch to map[variable]variable later
	fmt.Println(varMap)

	ast.Inspect(node, func(currNode ast.Node) bool {
		if callNode, ok := node.(*ast.CallExpr); ok{
			for _, expr := range callNode.Args{
				switch fnLit := expr.(type){
				case *ast.FuncLit:
					fmt.Printf("func lit type %s, lit body %s", fnLit.Type, fnLit.Body)
				case *ast.Ident:
					fmt.Println("Ident", fnLit.Name, fnLit.Obj)
				}
			}
		}

		return true
	})

}

//Gets all the variables within a block -
func GetVariables(curr Wrapper) []ast.Node {
	filter := make(map[string]ast.Node)
	varList := []ast.Node{}

	switch curr := curr.(type) {
	case *FnWrapper:
	case *BlockWrapper:
		//Process all nodes in a block for possible variables
		for _, node := range curr.Block.Nodes {
			//fmt.Println("hm", reflect.TypeOf(node))
			ast.Inspect(node, func(currNode ast.Node) bool {
				//fmt.Println("hm2", reflect.TypeOf(node))
				//If a node is an assignStmt or ValueSpec, it should most likely be a variable
				switch node := node.(type) {
				case *ast.ValueSpec, *ast.AssignStmt, *ast.IncDecStmt, *ast.ExprStmt, *ast.Ident:
					//Gets variable name
					name := GetVarName(curr, node)

					//filter out duplicates
					_, ok := filter[name]
					if ok && name != "" {
						//fmt.Println(name, " is already in the list")
					} else {
						filter[name] = node
					}
				}
				return true
			})
		}

		//Convert into list of variable nodes
		for _, node := range filter {
			varList = append(varList, node)
		}
	}

	return varList
}

//Processes function node information (still needs to track variables per function)
func GetFuncInfo(curr Wrapper, node ast.Node) (string, map[string]ast.Node) {

	funcName := ""
	funcVars := make(map[string]ast.Node)

	ast.Inspect(node, func(currNode ast.Node) bool {
		switch node := node.(type) {
		case *ast.FuncDecl: //Check parameters
			funcName = node.Name.Name

			if node.Type.Params != nil {
				//Get parameters
				for _, params := range node.Type.Params.List {
					if params != nil {
						//fmt.Println("Parameter", params.Names)
						//fmt.Println("Param type", fmt.Sprint(params.Type))
					}
				}
			}

			//Go through body for statements
			for _, statement := range node.Body.List {
				switch stmt := statement.(type) {
				case *ast.AssignStmt: //for variables
					//fmt.Println("Assign stmt found", stmt)
					//fmt.Println("Var name", GetVarName(stmt))

					//If var is already in the map, skip
					varName := GetVarName(curr, stmt)
					if _, ok := funcVars[varName]; ok {

					} else {
						funcVars[varName] = stmt
					}

				case *ast.ReturnStmt: //Not sure if this is even needed
					//fmt.Println("Return stmt", stmt)
				default:
					//fmt.Println("default", stmt)
				}
			}

		case *ast.FuncLit:
			//TOOD: handle literals
		}
		return true
	})

	return funcName, funcVars
}

//Extract logging statements from a cfg block
func ExtractLogRegex(block *cfg.Block) (regexes []string) {

	//For each node inside the block, check if it contains logging stmts
	for _, currNode := range block.Nodes {
		ast.Inspect(currNode, func(node ast.Node) bool {
			if call, ok := node.(*ast.CallExpr); ok {
				if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
					if logsource.IsFromLog(sel) {
						//get log regex from the node
						for _, arg := range call.Args {
							switch logNode := arg.(type) {
							case *ast.BasicLit:
								regexStr := logsource.CreateRegex(logNode.Value)
								regexes = append(regexes, regexStr)
							}
							//Currently an runtime bug with catching Msgf ->
						}
					}
				}
			}
			return true
		})
	}

	return regexes
}

//Expression String
func GetExprStr(expr ast.Expr) string {
	if expr == nil {
		return ""
	}

	//Return expression based on the type
	switch condition := expr.(type) {
	case *ast.BasicLit:
		return condition.Value
	case *ast.Ident:
		return condition.Name
	case *ast.BinaryExpr:
		leftStr, rightStr := "", ""
		leftStr = GetExprStr(condition.X)
		rightStr = GetExprStr(condition.Y)
		return fmt.Sprint(leftStr, condition.Op, rightStr)
	case *ast.UnaryExpr:
		op := condition.Op.String()
		str := GetExprStr(condition.X)
		return fmt.Sprint(op, str)
	case *ast.SelectorExpr:
		selector := ""
		if condition.Sel != nil {
			selector = condition.Sel.String()
		}
		str := GetExprStr(condition.X)
		return fmt.Sprintf("%s.%s", str, selector)
	case *ast.ParenExpr:
		return fmt.Sprintf("(%s)", GetExprStr(condition.X))
	case *ast.CallExpr:
		fn := GetExprStr(condition.Fun)
		args := make([]string, 0)
		for _, arg := range condition.Args {
			args = append(args, GetExprStr(arg))
		}
		if condition.Ellipsis != token.NoPos {
			args[len(args)-1] = fmt.Sprintf("%s...", args[len(args)-1])
		}

		var builder strings.Builder
		_, _ = builder.WriteString(fmt.Sprintf("%s(", fn))
		for i, arg := range args {
			var s string
			if i == len(args)-1 {
				s = fmt.Sprintf("%s)", arg)
			} else {
				s = fmt.Sprintf("%s, ", arg)
			}
			_, _ = builder.WriteString(s)
		}
		if len(args) == 0 {
			_, _ = builder.WriteString(")")
		}

		return builder.String()
	case *ast.IndexExpr:
		expr := GetExprStr(condition.X)
		ndx := GetExprStr(condition.Index)
		return fmt.Sprintf("%s[%s]", expr, ndx)
	case *ast.KeyValueExpr:
		key := GetExprStr(condition.Key)
		value := GetExprStr(condition.Value)
		return fmt.Sprint(key, ":", value)
	case *ast.SliceExpr: // not sure about this one
		expr := GetExprStr(condition.X)
		low := GetExprStr(condition.Low)
		high := GetExprStr(condition.High)
		if condition.Slice3 {
			max := GetExprStr(condition.Max)
			return fmt.Sprintf("%s[%s : %s : %s]", expr, low, high, max)
		}
		return fmt.Sprintf("%s[%s : %s]", expr, low, high)
	case *ast.StarExpr:
		expr := GetExprStr(condition.X)
		return fmt.Sprintf("*%s", expr)
	case *ast.TypeAssertExpr:
		expr := GetExprStr(condition.X)
		typecast := GetExprStr(condition.Type)
		return fmt.Sprintf("%s(%s)", typecast, expr)
		//case *ast.FuncType:
		//	params := fnCfg.getFuncParams(condition.Params, "")
		//	rets := fnCfg.getFuncReturns(condition.Results)
		//	b := strings.Builder{}
		//	b.Write([]byte("func("))
		//	i := 0
		//	for _, param := range params {
		//		b.Write([]byte(fmt.Sprintf("%s", param.VarName)))
		//		if i < len(params)-1 {
		//			b.Write([]byte(", "))
		//		}
		//	}
		//	b.Write([]byte(")"))
		//	for i, ret := range rets {
		//		b.Write([]byte(fmt.Sprintf("%s %s", ret.Name, ret.ReturnType)))
		//		if i < len(params)-1 {
		//			b.Write([]byte(", "))
		//		}
		//	}
		//	return b.String()
	}
	return ""
}

//Get name pointed to by the expr node
func GetPointedName(curr Wrapper, node ast.Expr) string {
	if outer, ok := curr.GetOuterWrapper().(*FnWrapper); ok {
		switch expr := node.(type) {
		case *ast.StarExpr:
			return GetPointedName(outer, expr.X)
		case *ast.Ident:
			fmt.Println("hello", outer, node)
			if expr.Obj != nil {
				if v, ok := outer.ParamsToArgs[expr.Obj]; ok {
					fmt.Println("--", v)
					return GetPointedName(outer, v)
				}
			}
		}
	}
	return fmt.Sprint(node)
}

//Helper function to get var name (handles both assign and declaration vars)
func GetVarName(curr Wrapper, node ast.Node) string {
	var name string = ""

	//fmt.Println("var", node, reflect.TypeOf(node))

	switch node := node.(type) {
	case *ast.AssignStmt:
		if len(node.Lhs) > 0 {
			name = GetPointedName(curr, node.Lhs[0])
			// name = fmt.Sprint(node.Lhs[0])
		}
		//for _, lhsExpr := range node.Lhs {
		//	switch expr := lhsExpr.(type) {
		//	case *ast.SelectorExpr:
		//		if expr.Sel.Name != "" {
		//			name = expr.Sel.Name
		//		}
		//	}
		//}
	case *ast.ValueSpec:
		//Set variable name
		if len(node.Names) > 0 {
			if _, ok := node.Type.(*ast.StarExpr); ok {
				name = GetPointedName(curr, node.Names[0])
			} else {
				name = node.Names[0].Name
			}
		}
	case *ast.IncDecStmt:
		name = GetPointedName(curr, node.X)
		//fmt.Println(name)
	case *ast.ExprStmt:
		name = GetPointedName(curr, node.X)
		//fmt.Println(name)
	}

	return name
}

func GetVarExpr(curr Wrapper, assignNode *ast.AssignStmt) string {

	var exprValue string
	exprOp := assignNode.Tok.String()
	varName := GetVarName(curr, assignNode)

	//Checks if rhs if a variable gets a value from a function or literal
	for _, rhsExpr := range assignNode.Rhs {
		switch expr := rhsExpr.(type) {
		//Basic literals indicate the var shouldn't have been returned from a function and a real value
		case *ast.BasicLit:
			if exprOp == ":=" {
				exprValue = varName + " " + exprOp + " " + expr.Value
			} else if exprOp == "=" {
				exprValue = "var " + varName + " " + expr.Kind.String() + " " + exprOp + " " + expr.Value
			}
			//isFromFunction = false
			//isReal = true

		case *ast.CompositeLit: //Indicates a variable being assigned a struct/slice/array (real value?)
			//fmt.Println("Is composite literal", expr.Type)

			//Grabbing the struct/slice assignment from the composite literal
			//litPos := expr.Type.Pos()
			//tempFile := fnCfg.fset.Position(litPos).Filename //TODO: need fset
			//lineNum := fnCfg.fset.Position(litPos).Line
			//file, err := os.Open(tempFile)
			//if err != nil {
			//	fmt.Println("Error opening file")
			//}

			//Read file at specific line to get function name
			//cnt := 1
			//var rightValue string = ""
			//scanner := bufio.NewScanner(file)
			//for scanner.Scan() {
			//	if cnt == lineNum {
			//		rightValue = scanner.Text()
			//		break
			//	}
			//	cnt++
			//}

			//Get the right side value assignment
			//rightValue = rightValue[strings.Index(rightValue, "=")+2 : strings.Index(rightValue, "{")]
			//
			//isFromFunction = false
			//isReal = false
			exprValue = varName + " " + exprOp + " " + "composite lit"

		default: //If it isn't a literal, it will be a symbolic value (from variable or from function)
			//litPos := expr.Pos()
			//tempFile := fnCfg.fset.Position(litPos).Filename
			//lineNum := fnCfg.fset.Position(litPos).Line
			//file, err := os.Open(tempFile)
			//if err != nil {
			//	fmt.Println("Error opening file")
			//}
			//
			////Read file at specific line to get function name
			//cnt := 1
			//var rightValue string = ""
			//scanner := bufio.NewScanner(file)
			//for scanner.Scan() {
			//	if cnt == lineNum {
			//		rightValue = scanner.Text()
			//		break
			//	}
			//	cnt++
			//}
			//
			////Sets the variable expression
			//isFromFunction, exprValue = isFunctionAssignment(rightValue)
			//isReal = false
			exprValue = varName + " " + exprOp + " default"
		}
	}

	return exprValue
}
