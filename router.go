package argrouter

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
)

type Router struct {
	runFuncs []func([]string) (bool, string, error)
}

func (r *Router) Run(args []string) error {
	for _, rf := range r.runFuncs {
		if ok, cmdName, err := rf(args); err != nil {
			return fmt.Errorf("failed to parse arguments for command '%s': %v", cmdName, err)
		} else if ok {
			return nil
		}
	}
	return fmt.Errorf("could not find matching command for the arguments '%s'", args)
}

func (r *Router) RunOS() error {
	return r.Run(os.Args[1:])
}

func NewRouter() *Router {
	return &Router{}
}

func Route[T, U any](r *Router, command string, handle func(options T, args U), defaultOptions T) {
	r.runFuncs = append(r.runFuncs, func(argsStr []string) (bool, string, error) {
		commandFeilds := strings.Fields(command)
		if len(commandFeilds) > len(argsStr) {
			return false, "", nil
		}
		for i := range commandFeilds {
			if commandFeilds[i] != argsStr[i] {
				return false, "", nil
			}
		}
		remainingArgs := argsStr[len(commandFeilds):]
		commandName := strings.Join(commandFeilds, " ")
		// Parse optional args
		options := defaultOptions
		pairs, remainingArgs, err := kvPairs(remainingArgs)
		if err != nil {
			return true, commandName, err
		}
		for k, v := range pairs {
			err := parseIntoOptStruct(v, &options, k)
			if err != nil {
				return true, commandName, err
			}
		}

		// Parse numbered args
		var args U
		expectedNum := reflect.ValueOf(args).NumField()
		if len(remainingArgs) != expectedNum {
			return true, commandName, fmt.Errorf("expected %d args but got %d", expectedNum, len(remainingArgs))
		}
		for i := range expectedNum {
			err := parseIntoNumberStruct(remainingArgs[i], &args, i)
			if err != nil {
				return true, commandName, err
			}
		}

		// Handle
		handle(options, args)
		return true, commandName, nil
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
		return nil, nil, fmt.Errorf("argument %s was provided no value", lastName)
	}
	return result, []string{}, nil
}

// parseIntoStruct parses the provided string into the struct field that matches the given tag name and value.
func parseIntoOptStruct(input string, ptrToStruct interface{}, tagValue string) error {
	structValue := reflect.ValueOf(ptrToStruct).Elem()
	structType := structValue.Type()

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		fieldValue := structValue.Field(i)
		if tag := field.Tag.Get("opt"); tag == tagValue {
			toSet, err := parseToValue(input, fieldValue.Kind())
			if err != nil {
				return err
			}
			fieldValue.Set(reflect.ValueOf(toSet))
			return nil
		}
	}

	return fmt.Errorf("invalid option '%s'", tagValue)
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

func parseToValue(input string, to reflect.Kind) (val any, err error) {
	switch to {
	case reflect.String:
		val, err = input, nil
	case reflect.Int:
		i, err2 := strconv.ParseInt(input, 10, 64)
		val, err = int(i), err2
	case reflect.Float64:
		val, err = strconv.ParseFloat(input, 64)
	case reflect.Bool:
		val, err = strconv.ParseBool(input)
	default:
		val, err = nil, fmt.Errorf("unsupported type: %s", to)
	}

	if err != nil {
		err = fmt.Errorf("failed to parse '%v' to a %v: %v", input, to, err)
	}
	return
}
