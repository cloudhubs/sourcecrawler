package cfg

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"sourcecrawler/app/logsource"
	"strconv"
	"strings"

	"github.com/mitchellh/go-z3"
	"golang.org/x/tools/go/cfg"
)

// Deletes the keys of any assignments that aren't user input (were assigned a value)
func FilterToUserInput(block Wrapper, nodes []ast.Node, assignments map[string]*z3.AST) {
	isAssigned := make(map[*ast.Object]bool)

	for _, node := range nodes {
		ast.Inspect(node, func(id ast.Node) bool {
			if id, ok := id.(*ast.Ident); ok && id.Obj != nil && id.Obj.Decl != nil {
				switch id.Obj.Decl.(type) {
				case *ast.AssignStmt:
					assigned := hasAssignment(nodes, id)
					if _, ok := assignments[id.Name]; ok && assigned {
						delete(assignments, id.Name)
					}
				case *ast.Field:
					if _, ok := assignments[id.Name]; ok {
						filterFnArgsUserInput(block, nodes, isAssigned, id)
						if assigned, ok := isAssigned[id.Obj]; ok {
							if assigned {
								delete(assignments, id.Name)
							}
						}
					}
				}
			}
			return true
		})
		// switch node := node.(type) {
		// case *ast.AssignStmt:
		// 	for _, l := range node.Lhs {
		// 		if id, ok := l.(*ast.Ident); ok {
		// 			if _, ok := assignments[id.Name]; ok {
		// 				delete(assignments, id.Name)
		// 			}
		// 		}
		// 	}
		// default:

		// }
	}
}

func hasAssignment(nodes []ast.Node, id *ast.Ident) bool {
	isAssigned := false
	for _, node := range nodes {
		ast.Inspect(node, func(node ast.Node) bool {
			switch node := node.(type) {
			case *ast.AssignStmt:
				for _, l := range node.Lhs {
					if foundID, ok := l.(*ast.Ident); ok && id == foundID {
						isAssigned = true
						return false
					}
				}
			}
			return true
		})
	}
	return isAssigned
}

func filterFnArgsUserInput(block Wrapper, constraints []ast.Node, isAssigned map[*ast.Object]bool, id *ast.Ident) {
	if block == nil || id.Obj == nil || id.Obj.Decl == nil {
		return
	}

	switch block := block.(type) {
	case *FnWrapper:
		var fnType *ast.FuncType
		switch fn := block.Fn.(type) {
		case *ast.FuncDecl:
			fnType = fn.Type
		case *ast.FuncLit:
			fnType = fn.Type
		}

		if fnType != nil || fnType.Params == nil {
			return
		}

		foundArg := false
		for _, field := range fnType.Params.List {
			if f, ok := id.Obj.Decl.(*ast.Field); ok && field == f {
				foundArg = true
				arg := block.ParamsToArgs[id.Obj]
				if argID, ok := arg.(*ast.Ident); ok && id.Obj != nil {
					if _, ok := isAssigned[argID.Obj]; !ok {
						// return isUserInput[id.Obj]
						assigned := hasAssignment(constraints, argID)
						isAssigned[argID.Obj] = assigned
					}
					isAssigned[id.Obj] = isAssigned[argID.Obj]
					// return isUserInput[id.Obj]
				}
				return
			}
		}

		if !foundArg {
			filterFnArgsUserInput(block.GetOuterWrapper(), constraints, isAssigned, id)
		}
	case *BlockWrapper:
		filterFnArgsUserInput(block.GetOuterWrapper(), constraints, isAssigned, id)
	}
	// return false
}

func ConvertExprToZ3(ctx *z3.Context, expr ast.Node, fset *token.FileSet) *z3.AST {
	if ctx == nil || expr == nil {
		// fmt.Println("returning nil")
		return nil
	}
	// fmt.Println("checking", expr, reflect.TypeOf(expr))
	switch expr := expr.(type) {
	case *ast.AssignStmt:
		var e *z3.AST
		for i, l := range expr.Lhs {
			r := expr.Rhs[i]
			lhs := ConvertExprToZ3(ctx, l, fset)
			rhs := ConvertExprToZ3(ctx, r, fset)
			if lhs != nil && rhs != nil {
				if e == nil {
					e = lhs.Eq(rhs)
				} else {
					e = e.And(lhs.Eq(rhs))
				}
			}
		}
		return e
	case *ast.BasicLit:
		switch expr.Kind {
		case token.INT:
			v, err := strconv.Atoi(expr.Value)
			// fmt.Println("literal value", v)
			if err == nil {
				return ctx.Int(v, ctx.IntSort())
			}
		}
		return nil
	case *ast.Ident:
		if expr.Obj != nil {
			// fmt.Println("nonnil obj")
			switch decl := expr.Obj.Decl.(type) {
			case *ast.Field:
				// fmt.Println("good field")
				switch t := decl.Type.(type) {
				case *ast.Ident:
					// fmt.Println("is ident")
					var sort *z3.Sort
					if strings.Contains(t.Name, "int") {
						sort = ctx.IntSort()
					} else if t.Name == "bool" {
						sort = ctx.BoolSort()
					}
					// fmt.Println("sort?", sort)
					//exprStr(expr)
					var bf bytes.Buffer
					printer.Fprint(&bf, fset, expr)
					//fmt.Println(bf.String())
					return ctx.Const(ctx.Symbol(bf.String()), sort)
					//case *ast.StarExpr:
					//case *ast.SelectorExpr:
				}
			case *ast.AssignStmt:
				for i, id := range decl.Lhs {
					if id.(*ast.Ident).Obj == expr.Obj {
						rhs := decl.Rhs[i]
						switch rhs := rhs.(type) {
						case *ast.UnaryExpr:
							if rhs, ok := rhs.X.(*ast.BasicLit); ok {
								switch rhs.Kind {
								case token.INT:
									var bf bytes.Buffer
									printer.Fprint(&bf, fset, id)
									return ctx.Const(ctx.Symbol(bf.String()), ctx.IntSort())
								}
							}
						case *ast.BasicLit:
							switch rhs.Kind {
							case token.INT:
								var bf bytes.Buffer
								printer.Fprint(&bf, fset, id)
								return ctx.Const(ctx.Symbol(bf.String()), ctx.IntSort())
							}
						}
					}
				}
			}
		}
		return nil
	case *ast.UnaryExpr:
		inner := ConvertExprToZ3(ctx, expr.X, fset)
		switch expr.Op {
		case token.NOT:
			return inner.Not()
		case token.ILLEGAL:
			return inner
		case token.SUB:
			return ctx.Int(0, ctx.IntSort()).Sub(inner)
		}
		return inner
	case *ast.BinaryExpr:
		left := ConvertExprToZ3(ctx, expr.X, fset)
		right := ConvertExprToZ3(ctx, expr.Y, fset)
		if left == nil || right == nil {
			// fmt.Println("can't combine", left, right)
			return nil
		}
		// fmt.Println("combining", left, right)
		switch expr.Op {
		case token.ADD:
			return left.Add(right)
		case token.SUB:
			return left.Sub(right)
		case token.MUL:
			return left.Mul(right)
		case token.LAND:
			return left.And(right)
		case token.LOR:
			return left.Or(right)
		case token.EQL:
			return left.Eq(right)
		case token.NEQ:
			//if int, Eq().Not()
			if _, err := strconv.Atoi(right.String()); err == nil {
				return left.Eq(right).Not()
			}
			//if variable, Distinct()
			return left.Distinct(right)
		case token.LSS:
			return left.Lt(right)
		case token.GTR:
			return left.Gt(right)
		case token.LEQ:
			return left.Le(right)
		case token.GEQ:
			return left.Ge(right)
		case token.XOR:
			return left.Xor(right)
		}
	case *ast.ParenExpr:
		return ConvertExprToZ3(ctx, expr.X, fset)
	case *ast.SelectorExpr:
		oldName := expr.Sel.Name
		//exprStr(expr.X)
		expr.Sel.Name = fmt.Sprintf("%v.%v", fmt.Sprint(expr.X), oldName)
		ident := ConvertExprToZ3(ctx, expr.Sel, fset)
		expr.Sel.Name = oldName
		return ident
	case *ast.StarExpr:
		return ConvertExprToZ3(ctx, expr.X, fset)
	}
	return nil
}

func SSAconversion(expr ast.Expr, ssaInts map[string]int) {
	ast.Inspect(expr, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.Ident:
			if i, ok := ssaInts[node.Name]; ok {
				if !strings.Contains("0123456789", string(node.Name[0])) {
					node.Name = fmt.Sprint(i, node.Name)
				}
			}
		}
		return true
	})
}

// Converts shorthand assignment forms (or IncDec) to their
// lengthier regular token.ASSIGN counterpart.
//
// Note: The identifier is copied because otherwise the
// left and right hand side would always share the exact same
// identifier which we would not want.
func RessignmentConversion(node ast.Node) *ast.AssignStmt {
	stmt := new(ast.AssignStmt)
	stmt.Tok = token.ASSIGN

	var tok token.Token

	switch node := node.(type) {
	case *ast.AssignStmt:
		for _, r := range node.Rhs {
			call := false
			ast.Inspect(r, func(node ast.Node) bool {
				if _, ok := node.(*ast.CallExpr); ok {
					call = true
					return false
				}
				return true
			})
			if call {
				return nil
			}
		}

		switch node.Tok {
		case token.ADD_ASSIGN: // +=
			tok = token.ADD
		case token.SUB_ASSIGN: // -=
			tok = token.SUB
		case token.MUL_ASSIGN: // *=
			tok = token.MUL
		case token.QUO_ASSIGN: // /=
			tok = token.SUB
		case token.REM_ASSIGN: // %=
			tok = token.REM
		default:
			return node
		}

		stmt.TokPos = node.TokPos
		for i, l := range node.Lhs {
			stmt.Lhs = append(stmt.Lhs, l)
			var id *ast.Ident
			if node, ok := l.(*ast.Ident); ok {
				id = &ast.Ident{
					Name:    node.Name,
					NamePos: node.NamePos,
					Obj:     node.Obj,
				}
			}
			bin := &ast.BinaryExpr{
				X:     id,
				OpPos: node.TokPos,
				Op:    tok,
				Y:     node.Rhs[i],
			}
			stmt.Rhs = append(stmt.Rhs, bin)
		}
	case *ast.IncDecStmt:
		switch node.Tok {
		case token.INC: // ++
			tok = token.ADD
		case token.DEC: // --
			tok = token.SUB
		default:
			return nil
		}

		stmt.TokPos = node.TokPos
		stmt.Lhs = append(stmt.Lhs, node.X)
		var id *ast.Ident
		if node, ok := node.X.(*ast.Ident); ok {
			id = &ast.Ident{
				Name:    node.Name,
				NamePos: node.NamePos,
				Obj:     node.Obj,
			}
		}
		bin := &ast.BinaryExpr{
			X:     id,
			OpPos: node.TokPos,
			Op:    tok,
			Y:     &ast.BasicLit{Value: "1", Kind: token.INT, ValuePos: node.TokPos},
		}
		stmt.Rhs = append(stmt.Rhs, bin)
	}

	return stmt
}

//Method to get condition, nil if not a conditional (specific to block wrapper) - used in traverse function
func (b *BlockWrapper) GetCondition() ast.Node {
	//Conditional block
	if len(b.Succs) == 2 && b.Block != nil && len(b.Block.Nodes) > 0 {
		//conditional is last node in a block
		return b.Block.Nodes[len(b.Block.Nodes)-1]
	}

	return nil
}

//Gets all the variables within a block -
func GetVariables(curr Wrapper, filter map[string]ast.Node) []ast.Node {
	varList := []ast.Node{}

	switch curr := curr.(type) {
	case *FnWrapper:
	case *BlockWrapper:
		//Process all nodes in a block for possible variables
		for _, node := range curr.Block.Nodes {
			//If a node is an assignStmt or ValueSpec, it should most likely be a variable
			//Gets variable name
			name, node := GetVar(curr, node)

			//filter out duplicates
			_, ok := filter[name]
			if ok && name != "" {
				// name is already in the list
			} else {
				filter[name] = node
			}
		}

		//Convert into list of variable nodes
		for _, node := range filter {
			varList = append(varList, node)
		}
	}

	// fmt.Println("vars", varList)

	return varList
}

//Helper function to get var name (handles both assign and declaration vars)
func GetVar(curr Wrapper, node ast.Node) (string, ast.Node) {
	var name string = ""

	// fmt.Println("var", node, reflect.TypeOf(node))

	switch n := node.(type) {
	case *ast.AssignStmt:
		if len(n.Lhs) > 0 {
			name, node = GetPointedName(curr, n.Lhs[0])

			name = name + " " + n.Tok.String() + " " + fmt.Sprint(n.Rhs[0])
			// fmt.Println("new name", name)
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
		if len(n.Names) > 0 {
			if _, ok := n.Type.(*ast.StarExpr); ok {
				name, node = GetPointedName(curr, n.Names[0])
			} else {
				name = n.Names[0].Name
			}
		}
	case *ast.IncDecStmt:
		name, node = GetPointedName(curr, n.X)
	// case *ast.ExprStmt:
	// name, node = GetVar(curr, n.X)
	case *ast.Ident:
		name, node = GetPointedName(curr, n)
	}

	return name, node
}

//Get name pointed to by the expr node
func GetPointedName(curr Wrapper, node ast.Expr) (string, ast.Node) {

	if outer, ok := curr.GetOuterWrapper().(*FnWrapper); ok {
		switch expr := node.(type) {
		case *ast.SelectorExpr:
			lhs, _ := GetPointedName(outer, expr.X)
			rhs, _ := GetPointedName(outer, expr.Sel)
			return fmt.Sprintf("%v.%v", lhs, rhs), node
		case *ast.StarExpr:
			return GetPointedName(outer, expr.X)
		case *ast.Ident:
			if expr.Obj != nil {
				if field, ok := expr.Obj.Decl.(*ast.Field); ok {
					// Check if the variable should return the name it points to or
					// otherwise, return the current node's string
					switch field.Type.(type) {
					case *ast.StarExpr, *ast.FuncType, *ast.StructType:
						if v, ok := outer.ParamsToArgs[expr.Obj]; ok {
							return GetPointedName(outer, v)
						}
					}
				}

			}
		}
	}
	return fmt.Sprint(node), node
}

//Processes function node information
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
					varName, _ := GetVar(curr, stmt)
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

func GetVarExpr(curr Wrapper, assignNode *ast.AssignStmt) string {

	var exprValue string
	exprOp := assignNode.Tok.String()
	varName, _ := GetVar(curr, assignNode)

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
