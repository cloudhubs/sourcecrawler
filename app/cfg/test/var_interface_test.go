package test

import (
	"sourcecrawler/app/cfg"
	"testing"
)

type varTestCase struct {
	Name string
}

func TestInterface(t *testing.T) {
	cases := []func() varTestCase{
		func() varTestCase {
			//construct a graph to represent a cfg

			one := cfg.FnVariableWrapper{
				Name: "one",
			}
			two := cfg.FnVariableWrapper{
				Name: "two",
			}
			fail := cfg.FnVariableWrapper{
				Name: "fail",
			}

			one.SetValue("test")
			two.SetValue(&cfg.FnWrapper{})
			fail.SetValue(2)
			return varTestCase{
				Name: "setValue",
			}

		},
	}

	for _, testCase := range cases {
		test := testCase()
		t.Run(test.Name, func(t *testing.T) {

		})
	}
}