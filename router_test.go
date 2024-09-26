package argrouter

import (
	"fmt"
	"testing"
)

func parseTestArgs(def testOptions, args ...string) (testArgs, testOptions, error) {
	r := NewRouter()
	var resultArgs testArgs
	var resultOptions testOptions
	Route(r, "test", func(x testOptions, y testArgs) error {
		resultArgs, resultOptions = y, x
		return nil
	}, def)
	_, err := r.Run(args)
	return resultArgs, resultOptions, err
}

type testOptions struct {
	Option1       int     `opt:"option-1"`
	Option2       string  `opt:"option-2"`
	Option3       float64 `opt:"option-3"`
	Option4       bool    `opt:"option-4"`
	InvalidOption []int   `opt:"invalid"`
}

type testArgs struct {
	Arg1 string
	Arg2 int
}

func (ta testArgs) assertEq(t *testing.T, other testArgs) {
	if ta.Arg1 != other.Arg1 || ta.Arg2 != other.Arg2 {
		t.Logf("Parsed args do not equal:\n%v\n%v\n", ta, other)
		t.FailNow()
	}
}

func (ta testOptions) assertEq(t *testing.T, other testOptions) {
	if ta.Option1 != other.Option1 || ta.Option2 != other.Option2 {
		t.Logf("Parsed options do not equal:\n%v\n%v\n", ta, other)
		t.FailNow()
	}
}

func TestOK(t *testing.T) {
	type okPair struct {
		args            []string
		expectedArgs    testArgs
		expectedOptions testOptions
	}

	pairs := []okPair{
		{
			[]string{"test", "a", "3"},
			testArgs{Arg1: "a", Arg2: 3},
			testOptions{},
		},
		{
			[]string{"test", "string", "-72"},
			testArgs{Arg1: "string", Arg2: -72},
			testOptions{},
		},
		{
			[]string{"test", "-option-1", "9", "string", "-72"},
			testArgs{Arg1: "string", Arg2: -72},
			testOptions{Option1: 9},
		},
		{
			[]string{"test", "-option-2", "optional!", "string", "-72"},
			testArgs{Arg1: "string", Arg2: -72},
			testOptions{Option2: "optional!"},
		},
		{
			[]string{"test", "-option-1", "10", "-option-2", "optional2!", "string", "-72"},
			testArgs{Arg1: "string", Arg2: -72},
			testOptions{Option1: 10, Option2: "optional2!"},
		},
		{
			[]string{"test", "-option-3", "-5.3", "-option-4", "true", "string", "-72"},
			testArgs{Arg1: "string", Arg2: -72},
			testOptions{Option3: -5.3, Option4: true},
		},
	}
	for _, p := range pairs {
		resArgs, resOptions, err := parseTestArgs(testOptions{}, p.args...)
		if err != nil {
			fmt.Println("Args:", p.args)
			t.Error(err)
			t.FailNow()
		}
		resArgs.assertEq(t, p.expectedArgs)
		resOptions.assertEq(t, p.expectedOptions)
	}
}

func TestInvalid(t *testing.T) {
	invalidArgs := [][]string{
		{"test"},
		{"test", "a"},
		{"test", "11"},
		{"test", "a", "b"},
		{"test", "-option-1", "11", "b"},
		{"test", "-option-1", "a", "11", "b"},
		{"test", "-option-1"},
		{"test", "abc", "11", "b"},
		{"test2", "11", "b"},
		{"test", "-njidsnjfnksje", "11", "b"},
		{"test", "-invalid", "5", "11", "b"},
		{},
	}

	for _, args := range invalidArgs {
		_, _, err := parseTestArgs(testOptions{}, args...)
		if err == nil {
			t.Log(args)
			t.Error("Should have failed but didn't")
			t.FailNow()
		}
	}
}
