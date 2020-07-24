package cfg

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"sourcecrawler/app/helper"
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
				for i, l := range node.Lhs {
					if foundID, ok := l.(*ast.Ident); ok && id == foundID {
						switch r := node.Rhs[i].(type) {
						case *ast.SelectorExpr:
							if id, ok := r.X.(*ast.Ident); ok {
								if id.Name == "os" {
									args := false
									ast.Inspect(r.Sel, func(node ast.Node) bool {
										if argsID, ok := node.(*ast.Ident); ok {
											if argsID.Name == "Args" {
												args = true
												return false
											}
										}
										return true
									})
									if !args {
										isAssigned = true
									}
								}
							}
						}
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
				for _, id := range decl.Lhs {
					if id.(*ast.Ident).Obj == expr.Obj {
						var bf bytes.Buffer
						printer.Fprint(&bf, fset, id)
						return ctx.Const(ctx.Symbol(bf.String()), ctx.IntSort())
						// rhs := decl.Rhs[i]
						// switch rhs := rhs.(type) {
						// case *ast.UnaryExpr:
						// 	if rhs, ok := rhs.X.(*ast.BasicLit); ok {
						// 		switch rhs.Kind {
						// 		case token.INT:
						// 			var bf bytes.Buffer
						// 			printer.Fprint(&bf, fset, id)
						// 			return ctx.Const(ctx.Symbol(bf.String()), ctx.IntSort())
						// 		}
						// 	}
						// case *ast.BasicLit:
						// 	switch rhs.Kind {
						// 	case token.INT:
						// 		var bf bytes.Buffer
						// 		printer.Fprint(&bf, fset, id)
						// 		return ctx.Const(ctx.Symbol(bf.String()), ctx.IntSort())
						// 	}
						// }
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
	name := ""

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

//Extract logging statements from a cfg block
func ExtractLogRegex(block *cfg.Block) (regexes []string) {

	//For each node inside the block, check if it contains logging stmts
	for _, currNode := range block.Nodes {
		ast.Inspect(currNode, func(node ast.Node) bool {
			if call, ok := node.(*ast.CallExpr); ok {
				if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
					if helper.IsFromLog(sel) {
						//get log regex from the node
						for _, arg := range call.Args {
							switch logNode := arg.(type) {
							case *ast.BasicLit:
								regexStr := helper.CreateRegex(logNode.Value)
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
