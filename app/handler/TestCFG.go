package handler

import (
	"fmt"

	"log"
)

//Testing CFG - can remove
func doMath() int{
	sum := 5 + 5
	return sum
}

//Testing CFG - can remove
func testConditional() bool{
	sum := 5
	if sum > 6{
		return true
	} else{
		return false
	}
}

//
func testPanic(){
	panic("Throwing error")
}

func dummy() int{
	return 6
}

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
		fmt.Println("Error")
		log.Println("abc")
	}
}
