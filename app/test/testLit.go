package test

import (
	"fmt"
)

// ----- Test cases for var_interface_test.go ------------

func testCond(x int) {
	if x <= 15 {

		y := 25

		if y < 10 {
			fmt.Println("y is small")
		} else {
			fmt.Println("y is large")
		}

		var array [10]string

		if x > -1 {
			if x <= 11 {
				// array[x-1] = parseMessage(msg)
				array[0] = "hello"
			}
		}

		if x < -1 {
			fmt.Println("x is negative")
		}
	} else {
		fmt.Println("outer Else")
	}

}

// func testCond(x int) {

// 	y := 6

// 	if x+y > 10 {
// 		fmt.Println(x, y)
// 		var num1 int = 10
// 		num2 := x
// 		fmt.Println(num1, num2)
// 		log.Log().Msg("Logging mezssage")
// 	} else {

// 		if x < 8 {
// 			fmt.Println("test else if")
// 		}
// 		elseVar := 15
// 		fmt.Println("else", elseVar)
// 	}

// }

//func testCond(x int){
//
//	if x > 10{
//		fmt.Println(x)
//		var num1 int = 10
//		num2 := x
//		fmt.Println(num1, num2)
//	}else if x > 5{
//		elseVar := 15
//		fmt.Println("else", elseVar)
//	}
//
//}

func func1() {
	a := 1
	func2(a)

	test := two()
	fmt.Println(test)
}

func func2(b int) {
	fmt.Println(b)
}

func one() {
	a := two()
	three(a)
}
func two() int {
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
