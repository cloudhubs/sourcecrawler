package test

import (
	"fmt"
	"log"
)

func Simple(num int) int {
	x := 10

	if x < 15{ 
		y := 200
		log.Println("Logging msg", x, y)
		if y < 300{
			log.Panic("Exception HERE!") //Path 1 -- should be a MUST
		}else{
			fmt.Println("Safe from panic. Deep breath") //Path 2 -- May
		}
	}else{
		x = 500     //Path 3 - May
	}

	return x
}