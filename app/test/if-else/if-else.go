package ifelse

import (
	"errors"
	"os"
	"strconv"

	"github.com/rs/zerolog/log"
)

func main() {

	num, _ := strconv.Atoi(os.Args[1])

	y := num % 2
	if y == 0 {
		log.Log().Msgf("%d is even", num)
	} else {
		log.Log().Msgf("%d is odd", num)
	}

	x := num % 4
	if x == 0 {
		log.Log().Msgf("%d is divisible by 4", num)
	}

	if num < 0 {
		log.Log().Msgf("%d is negative", num)
	} else if num < 10 {
		log.Warn().Msgf("%d has 1 digit", num)
		panic(errors.New("has 1 digit"))
	} else {
		log.Log().Msgf("%d has multiple digits", num)
	}
}
