package handler

import "log"

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
	log.Panicf("Throwing error")
}
