package ifelse

import (
	"errors"
	"os"
	"strconv"

	"github.com/rs/zerolog/log"
)

func main() {

	num, _ := strconv.Atoi(os.Args[1])

	num *= 2
	if num > 4 {
		log.Log().Msgf("%d > 4", num)
	} else {
		log.Log().Msgf("%d <= 4", num)
	}

	x := num * 4
	if x < 9 {
		log.Log().Msgf("%d < 9", num)
	}

	num -= 2
	if num < 0 {
		log.Log().Msgf("%d is negative", num)
	} else if num < 10 {
		log.Warn().Msgf("%d has 1 digit", num)
		panic(errors.New("has 1 digit"))
	} else {
		log.Log().Msgf("%d has multiple digits", num)
	}
}
