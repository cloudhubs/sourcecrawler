package test

import "fmt"

func func1(){
	a := 1
	func2(a)
}

func func2(b int){
	fmt.Println(b)
}

/*
(FnWrapper for func1){
    Variables: []VariableWrapper{{Name: a, Value: 1},}

  Code: {
    a := 1
    func2(a)
    (FnWrapper for func2){
        Variables: []VariableWrapper{{Name: b, Value: Outer.Variables[0]},}
      fmt.Println(b)
    }
  }
}
 */