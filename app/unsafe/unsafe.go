package unsafe

import (
	"errors"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
)

type SomeInterface interface {
	GetMessage() string
}

type SomeStruct struct {
	Count int
}

func Unsafe(x int, msg string) ([]string, error) {

	if x <= 15 {

		y := x + 5

		if y < 10 {
			log.Info().Msg("y is small")
		} else {
			log.Info().Msg("y is large")
		}

		var array [10]string

		if x > -1 {
			if x <= 11 {
				array[x-1] = parseMessage(msg)
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

func getStruct(y int) SomeInterface {
	if y > 75 {
		return nil
	} else {
		return &SomeStruct{0}
	}
}

func getArray(message string, size int) []string {
	var strArr [10]string
	if size <= 11 {
		strArr[size-1] = parseMessage(message)
	}

	return strArr[:]
}

func warning() {
	warning2()
}

func warning2() {
	log.Warn().Msg("Program may not work correctly")
}

func (s *SomeStruct) GetMessage() string {
	return fmt.Sprintf("The count is %v", s.Count)
}

func parseMessage(msg string) string {
	return strings.Trim(msg, " ")
}
