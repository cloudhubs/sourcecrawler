package handler

import (
	"fmt"

	"github.com/rs/zerolog/log"
)

//Testing CFG - can remove
func doMath() int {
	sum := 5 + 5
	return sum
}

//Testing CFG - can remove
func testConditional() bool {
	sum := 5
	if sum > 6 {
		return true
	} else {
		return false
	}
}


//test
func testPanic() {
	log.Panic().Msg("Testing zerolog panic")
}

func callPanic(){
	panic("THROWING ERROR")
}

func callOtherPanic(num int){
	panic(num)
}

func main() {
	go foo() // not handled at the moment
	x := 1
	x++
	y := 1
	x = x + y
	defer func() { x = x + y }() // not handled at the moment
	a, b := foo(), bar()
	c, d, e := foo(), foo(), bar()
	f, g, h := 3, foo(), 1
	fmt.Println(a, b, c, d, e, f, g, h)
}

func dummy() int { return 3 }

func test() {
	dummy()
	x := dummy()
	if x < 5 {
		if 2 < dummy() {
			x++
		} else {
			x--
		}
		if 1 > dummy() {
			x++
		} else {
			x--
		}
	} else {
		log.Info().Msgf("something %v")
	}
}

func foo() int {
	fmt.Println("foo")
	return 1
}

func bar() int {
	fmt.Println("bar")
	return 1
}

func ifelseTest() {
	var x int
	if 4 < foo() {
		x++
	} else if dummy()+bar() < 9 {
		x++
	} else if bar() < 2 {
		x++
	} else {
		x++
	}
}

func initif() {
	if x := foo(); x < 3 {
		fmt.Println("something", x)
	}
}

func forloop() {
	for x := foo(); x < 10; x++ {
		fmt.Println("kek")
	}
}

func boolfn() bool {
	return false
}

func initcall() {
	if boolfn() {
		foo()
	} else if !boolfn() {
		bar()
	} else {
		fmt.Println("something")
	}
}

func testCondPanic(num int){
	if num > 10{
		callPanic()
	}else{
		callOtherPanic(num)
	}
}

func testNodeProp(str string) string {
	createChild(str)

	return str
}

func createChild(str string){
	if str == "a"{
		createChildOfChild()
	}else{
		//child node
		createChildOfChild()
	}
}

func createChildOfChild(){

}
