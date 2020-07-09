package test

import "fmt"

// ----- Test cases for var_interface_test.go ------------

func func1(){
	a := 1
	func2(a)

	test := two()
	fmt.Println(test)
}

func func2(b int){
	fmt.Println(b)
}

func one(){
	a := two()
	three(a)
}
func two() int{
	return 2
}
func three(b int) {
	fmt.Print(b)
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