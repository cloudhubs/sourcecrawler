package unsafe

import (
	"errors"
	"fmt"

	"github.com/rs/zerolog/log"
)

type SomeInterface interface {
	GetMessage() string
}

type SomeStruct struct {
	Count int
}

func Unsafe(x int, msg string) ([]string, error) {
	if x > 20 {
		warning()
		return nil, errors.New("x too big")
	} else if x < -1 {
		warning()
		return nil, errors.New("x too small")
	}

	obj := getStruct(x)
	log.Info().Msgf("We are executing with message %v", obj.GetMessage())

	array := getArray(obj.GetMessage(), x)

	return array, nil
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
	for i := 0; i < size; i++ {
		strArr[i] = message
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
