package cfg

import (
	"fmt"
	"sourcecrawler/app/model"
)

//---------- Labeling feature for Must/May-haves (rewrite) --------------
func LabelCFG(curr Wrapper, logs []model.LogType, root Wrapper){

	//Label root
	if curr == root || len(curr.GetParents()) == 0{
		fmt.Println("At topmost wrapper")
		curr.SetLabel(Must)
		return
	}


	wrapper := curr
	for len(wrapper.GetParents()) > 0{
		if wrapper.GetLabel() == NoLabel{
			switch wrap := wrapper.(type){
			case *FnWrapper:
				//If it's a function we can assume it runs
				//Check fn.node as same name on stack
				wrap.SetLabel(Must)
			case *BlockWrapper: //BlockWrapper can represent a condition, but could be a statement, etc
				//Check if it's a condition, if not set as must
				if wrap.GetCondition() == ""{
					wrap.SetLabel(Must)
				}
				//Two parents = branches joining
				//Check each node to see if it is from log
			}
		}
	}
}
