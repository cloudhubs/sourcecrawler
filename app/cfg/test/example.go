package test

import "fmt"

func exampleMain() {
	x := 1

	if x < 2 {
		foo(x)
		if x == 2 {
			x := 3
			y := bar(x, x)
			y++
			if y == 3 {
				fmt.Println("This is a log")
			}
		}
	}
}

func foo(x int) int {
	if x > 5 {
		x := 3
		if x < 4 {
			x++
		}
	}
	return x
}

func bar(x, y int) int {
	if y > 1000 {
		x *= 2
		if x > y {
			x += y
		}
	}
	return x
}
