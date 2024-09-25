package argrouter

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
)

type Nil struct{}

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

func Route[T, U any](r *Router, command string, handle func(options T, args U), defaultOptions T) {
	r.runFuncs = append(r.runFuncs, func(argsStr []string) (bool, error) {
		commandFeilds := strings.Fields(command)
		if len(commandFeilds) > len(argsStr) {
			return false, nil
		}
		for i := range commandFeilds {
			if commandFeilds[i] != argsStr[i] {
				return false, nil
			}
		}
		remainingArgs := argsStr[len(commandFeilds):]
		// Parse optional args
		options := defaultOptions
		pairs, remainingArgs, err := kvPairs(remainingArgs)
		if err != nil {
			return true, err
		}
		for k, v := range pairs {
			err := parseIntoNameStruct(v, &options, "opt", k)
			if err != nil {
				return true, err
			}
		}

		// Parse numbered args
		var args U
		expectedNum := reflect.ValueOf(args).NumField()
		if len(remainingArgs) != expectedNum {
			return true, fmt.Errorf("expected %d args but got %d", expectedNum, len(remainingArgs))
		}
		for i := range expectedNum {
			err := parseIntoNumberStruct(remainingArgs[i], &args, i)
			if err != nil {
				return true, err
			}
		}

		// Handle
		handle(options, args)
		return true, nil
	})
}

func kvPairs(args []string) (map[string]string, []string, error) {
	waitingForDash := true
	lastName := ""
	result := make(map[string]string)

	for len(args) > 0 {
		if waitingForDash {
			if strings.HasPrefix(args[0], "-") {
				lastName = strings.TrimPrefix(args[0], "-")
				waitingForDash = false
			} else {
				return result, args, nil
			}
		} else {
			result[lastName] = args[0]
			waitingForDash = true
			lastName = ""
		}
		args = args[1:]
	}

	if !waitingForDash {
		return nil, nil, fmt.Errorf("argument %s got no value", lastName)
	}
	return result, []string{}, nil
}

// parseIntoStruct parses the provided string into the struct field that matches the given tag name and value.
func parseIntoNameStruct(input string, ptrToStruct interface{}, tagName, tagValue string) error {
	structValue := reflect.ValueOf(ptrToStruct).Elem()
	structType := structValue.Type()

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		fieldValue := structValue.Field(i)
		if tag := field.Tag.Get(tagName); tag == tagValue {
			toSet, err := parseToValue(input, fieldValue.Kind())
			if err != nil {
				return err
			}
			fieldValue.Set(reflect.ValueOf(toSet))
			return nil
		}
	}

	return fmt.Errorf("no field with tag %s=%s found", tagName, tagValue)
}

// parseIntoStruct parses the provided string into the struct field that matches the given tag name and value.
func parseIntoNumberStruct(input string, ptrToStruct interface{}, i int) error {
	structValue := reflect.ValueOf(ptrToStruct).Elem()

	fieldValue := structValue.Field(i)
	toSet, err := parseToValue(input, fieldValue.Kind())
	if err != nil {
		return err
	}
	fieldValue.Set(reflect.ValueOf(toSet))
	return nil
}

func parseToValue(input string, to reflect.Kind) (any, error) {
	switch to {
	case reflect.String:
		return input, nil
	case reflect.Int:
		i, err := strconv.ParseInt(input, 10, 64)
		return int(i), err
	case reflect.Float64:
		return strconv.ParseFloat(input, 64)
	case reflect.Bool:
		return strconv.ParseBool(input)
	default:
		return nil, fmt.Errorf("unsupported type: %s", to)
	}
}
