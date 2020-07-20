package test

import (
	"fmt"
	"errors"
)

func Unsafe(x int, msg string) ([]string, error) {

	if x <= 15 {

		y := x + 5

		if y < 10 {
			fmt.Println("y is small")
		} else {
			fmt.Println("y is large")
		}

		var array [10]string

		if x > -1 {
			if x <= 11 {
				array[x-1] = "bad"
			}
		}

		if x < -1 {
			fmt.Println("x is negative")
		}

		return array[:], nil
	} else {
		return nil, errors.New("x too big")
	}
}