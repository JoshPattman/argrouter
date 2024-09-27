package argrouter

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

func NoHelp(string) {}

func PrintHelp(s string) {
	fmt.Println(s)
}

type Router struct {
	runFuncs     []func([]string) (bool, string, error, error)
	helpFunction func(string)
}

func (r *Router) Run(args []string) (string, error) {
	// Start by ordering our runfuncs so we always check the longest first
	sort.Slice(r.runFuncs, func(i, j int) bool {
		_, iName, _, _ := r.runFuncs[i](nil)
		_, jName, _, _ := r.runFuncs[j](nil)
		return len(strings.Fields(iName)) > len(strings.Fields(jName))
	})

	// For each run func, try to run it
	for _, rf := range r.runFuncs {
		if ok, cmdName, parseErr, funcErr := rf(args); parseErr != nil {
			return cmdName, errors.Join(fmt.Errorf("failed to parse arguments for command '%s'", cmdName), parseErr)
		} else if funcErr != nil {
			return cmdName, errors.Join(fmt.Errorf("failed to run command '%s'", cmdName), funcErr)
		} else if ok {
			return cmdName, nil
		}
	}
	return "", fmt.Errorf("could not find matching command for the arguments '%s'", args)
}

func (r *Router) RunOS() (string, error) {
	return r.Run(os.Args[1:])
}

func NewRouter(helpFunction func(string)) *Router {
	return &Router{runFuncs: make([]func([]string) (bool, string, error, error), 0), helpFunction: helpFunction}
}

func Route[T, U any](r *Router, command string, handle func(options T, args U) error, defaultOptions T, helpString string) {
	r.runFuncs = append(r.runFuncs, func(argsStr []string) (attempted bool, name string, parseErr error, runErr error) {
		commandFeilds := strings.Fields(command)
		commandName := strings.Join(commandFeilds, " ")
		if argsStr == nil || len(commandFeilds) > len(argsStr) {
			return false, commandName, nil, nil
		}
		for i := range commandFeilds {
			if commandFeilds[i] != argsStr[i] {
				return false, commandName, nil, nil
			}
		}
		remainingArgs := argsStr[len(commandFeilds):]
		// Parse optional args
		options := defaultOptions
		help, pairs, remainingArgs, err := kvPairs(remainingArgs)
		if err != nil {
			return true, commandName, err, nil
		}
		if help {
			r.helpFunction(helpString)
			return true, commandName, nil, nil
		}
		for k, v := range pairs {
			err := parseIntoOptStruct(v, &options, k)
			if err != nil {
				return true, commandName, err, nil
			}
		}

		// Parse numbered args
		var args U
		expectedNum := reflect.ValueOf(args).NumField()
		if len(remainingArgs) != expectedNum {
			return true, commandName, fmt.Errorf("expected %d args but got %d", expectedNum, len(remainingArgs)), nil
		}
		for i := range expectedNum {
			err := parseIntoNumberStruct(remainingArgs[i], &args, i)
			if err != nil {
				return true, commandName, err, nil
			}
		}

		// Handle
		err = handle(options, args)
		return true, commandName, nil, err
	})
}

// Returns help?, kv pairs, remaining args, err
func kvPairs(args []string) (bool, map[string]string, []string, error) {
	waitingForDash := true
	lastName := ""
	result := make(map[string]string)

	for len(args) > 0 {
		if waitingForDash {
			if strings.HasPrefix(args[0], "-") {
				if args[0] == "-h" {
					return true, nil, nil, nil
				}
				lastName = strings.TrimPrefix(args[0], "-")
				waitingForDash = false
			} else {
				return false, result, args, nil
			}
		} else {
			result[lastName] = args[0]
			waitingForDash = true
			lastName = ""
		}
		args = args[1:]
	}

	if !waitingForDash {
		return false, nil, nil, fmt.Errorf("argument %s was provided no value", lastName)
	}
	return false, result, []string{}, nil
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
