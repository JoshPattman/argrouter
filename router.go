package argrouter

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
)

type Router struct {
	runFuncs []func([]string) (bool, error)
}

func (r *Router) Run(args []string) error {
	for _, rf := range r.runFuncs {
		if ok, err := rf(args); err != nil {
			return err
		} else if ok {
			return nil
		}
	}
	return fmt.Errorf("could not find matching command")
}

func (r *Router) RunOS() error {
	return r.Run(os.Args[1:])
}

func NewRouter() *Router {
	return &Router{}
}

func Route[T any](r *Router, command string, handle func(args T), defaultVal T) {
	r.runFuncs = append(r.runFuncs, func(args []string) (bool, error) {
		val := defaultVal
		commandFeilds := strings.Fields(command)
		if len(commandFeilds) > len(args) {
			return false, nil
		}
		for i := range commandFeilds {
			if commandFeilds[i] != args[i] {
				return false, nil
			}
		}
		remainingArgs := args[len(commandFeilds):]

		// Parse optional args
		remainingArgs, err := trySetOptionalArgs(&val, remainingArgs)
		if err != nil {
			return true, err
		}

		// Parse numbered args
		if err := trySetArgs(&val, remainingArgs); err != nil {
			return true, err
		}
		handle(val)
		return true, nil
	})
}

func trySetOptionalArgs(into any, vals []string) ([]string, error) {
	i := 0
	awaitingName := true
	lastName := ""
	for i < len(vals) {
		if awaitingName {
			if strings.HasPrefix(vals[i], "-") {
				awaitingName = false
				lastName = strings.TrimPrefix(vals[i], "-")
			} else {
				return vals[i:], nil
			}
		} else {
			opt, err := findOption(into, lastName)
			if err != nil {
				return nil, err
			}
			if err := parseInto(vals[i], opt); err != nil {
				return nil, err
			}
			awaitingName = true
			lastName = ""
		}
		i++
	}
	if !awaitingName {
		return nil, fmt.Errorf("missing value for option %s", lastName)
	}
	return vals[i:], nil
}

func trySetArgs(into any, vals []string) error {
	intoType := reflect.TypeOf(into).Elem()
	intoPtr := reflect.ValueOf(into).Elem()
	ni := 0
	i := 0
	for i < intoPtr.NumField() {
		if intoType.Field(i).Tag.Get("opt") == "" {
			if ni >= len(vals) {
				return fmt.Errorf("not enough arguments")
			} else if err := parseInto(vals[ni], intoPtr.Field(i)); err != nil {
				return err
			}
			ni++
		}
		i++
	}
	if len(vals) > ni {
		return fmt.Errorf("too many arguments")
	}
	return nil
}

func parseInto(val string, into reflect.Value) error {
	switch into.Kind() {
	case reflect.String:
		into.SetString(val)
	case reflect.Int:
		i, err := strconv.Atoi(val)
		if err != nil {
			return err
		}
		into.SetInt(int64(i))
	default:
		return fmt.Errorf("unrecognised type to parse: %s", into.Kind().String())
	}
	return nil
}

func findOption(into any, option string) (reflect.Value, error) {
	intoType := reflect.TypeOf(into).Elem()
	intoPtr := reflect.ValueOf(into).Elem()

	for i := range intoType.NumField() {
		if intoType.Field(i).Tag.Get("opt") == option {
			return intoPtr.Field(i), nil
		}
	}
	return reflect.Value{}, fmt.Errorf("failed to find option '%s'", option)
}
