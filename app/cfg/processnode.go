package cfg

import (
	"fmt"
	"go/ast"
	"golang.org/x/tools/go/cfg"
	"sourcecrawler/app/logsource"
)

//Method to get condition, nil if not a conditional (specific to block wrapper)
func (b *BlockWrapper) GetCondition() string{

	var condition string = ""
	//Return block or panic, fatal, etc
	if len(b.Succs) == 0{
		return ""
	}
	//Normal block
	if len(b.Succs) == 1 {
		return ""
	}
	//Conditional block (TODO: assuming the last node in the block node list is the conditional node)
	if len(b.Succs) == 2 && b.Block != nil && len(b.Block.Nodes) > 0{
		condNode := b.Block.Nodes[len(b.Block.Nodes)-1] //Last node in the block's list of nodes
		ast.Inspect(condNode, func(currNode ast.Node) bool {
			if ifNode, ok := condNode.(*ast.IfStmt); ok{
				condition = fmt.Sprint(ifNode.Cond)
			}

			if forNode, ok := condNode.(*ast.ForStmt); ok{
				fmt.Println("For loop condition", forNode.Cond)
			}
			return true
		})
	}

	return condition
}

//Process variable nodes
func ProcessVariables(node *ast.Node) []*ast.Node{
	var varList []*ast.Node = []*ast.Node{}



	return varList
}

//Processes function node information
func getFuncInfo(node *ast.Node){

}

//Extract logging statements from a cfg block
func ExtractLogRegex(block *cfg.Block) (regexes []string){

	//For each node inside the block, check if it contains logging stmts
	for _, currNode := range block.Nodes{
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
