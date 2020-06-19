package handler

import (
	"fmt"

	"github.com/rs/zerolog/log"
)

type SomeInterface interface {
	GetMessage() string
}

type SomeStruct struct {
	Count int
}

func Unsafe(nilAssignment, badIndex bool) []string {
	if nilAssignment && badIndex {
		warning()
	}

	obj := getStruct(nilAssignment)
	log.Info().Msgf("We are executing with message %v", obj.GetMessage())

	var array []string
	if badIndex {
		array = getArray(obj.GetMessage(), 6)
	} else {
		array = getArray(obj.GetMessage(), 3)
	}
	return array
}

func getStruct(nilAssignment bool) SomeInterface {
	if nilAssignment {
		return nil
	} else {
		return &SomeStruct{0}
	}
}

func getArray(message string, index int) []string {
	var strArr [5]string
	strArr[index] = message
	return strArr[:]
}

func warning() {
	log.Warn().Msg("Program may not fail correctly")
}

func (s *SomeStruct) GetMessage() string {
	return fmt.Sprintf("The count is %v", s.Count)
}
