package epsagon

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/epsagon/epsagon-go/protocol"
	"reflect"
)

func errorHandler(e error) genericHandler {
	return func(ctx context.Context, payload json.RawMessage) (interface{}, error) {
		AddException(&protocol.Exception{
			Type:    "wrapper",
			Message: fmt.Sprintf("Error in wrapper: %v", e),
			Time:    GetTimestamp(),
		})
		return nil, e
	}
}

// validateArguments returns an error if the handler's arguments are
// not compatible with aws lambda handlers
// the boolean return value is wether or not the handler accepts context.Context
// in its first argument.
func validateArguments(handler reflect.Type) (bool, error) {
	handlerTakesContext := false
	if handler.NumIn() > 2 {
		return false, fmt.Errorf("handlers may not take more than two arguments, but handler takes %d", handler.NumIn())
	} else if handler.NumIn() > 0 {
		contextType := reflect.TypeOf((*context.Context)(nil)).Elem()
		argumentType := handler.In(0)
		handlerTakesContext = argumentType.Implements(contextType)
		if handler.NumIn() > 1 && !handlerTakesContext {
			return false, fmt.Errorf("handler takes two arguments, but the first is not Context. got %s", argumentType.Kind())
		}
	}

	return handlerTakesContext, nil
}

func validateReturns(handler reflect.Type) error {
	errorType := reflect.TypeOf((*error)(nil)).Elem()
	if handler.NumOut() > 2 {
		return fmt.Errorf("handler may not return more than two values")
	} else if handler.NumOut() > 1 {
		if !handler.Out(1).Implements(errorType) {
			return fmt.Errorf("handler returns two values, but the second does not implement error")
		}
	} else if handler.NumOut() == 1 {
		if !handler.Out(0).Implements(errorType) {
			return fmt.Errorf("handler returns a single value, but it does not implement error")
		}
	}
	return nil
}

func makeGenericHandler(handlerSymbol interface{}) genericHandler {
	if handlerSymbol == nil {
		return errorHandler(fmt.Errorf("handler is nil"))
	}
	handler := reflect.ValueOf(handlerSymbol)
	handlerType := reflect.TypeOf(handlerSymbol)
	if handlerType.Kind() != reflect.Func {
		return errorHandler(fmt.Errorf("handler kind %s is not %s", handlerType.Kind(), reflect.Func))
	}

	takesContext, err := validateArguments(handlerType)
	if err != nil {
		return errorHandler(err)
	}

	if err := validateReturns(handlerType); err != nil {
		return errorHandler(err)
	}

	return func(ctx context.Context, payload json.RawMessage) (interface{}, error) {
		// construct arguments
		var args []reflect.Value
		if takesContext {
			args = append(args, reflect.ValueOf(ctx))
		}
		if (handlerType.NumIn() == 1 && !takesContext) || handlerType.NumIn() == 2 {
			argType := handlerType.In(handlerType.NumIn() - 1)
			arg := reflect.New(argType)

			if err := json.Unmarshal(payload, arg.Interface()); err != nil {
				AddException(&protocol.Exception{
					Type:    "wrapper",
					Message: fmt.Sprintf("Error in wrapper: failed to convert arguments: %v", err),
					Time:    GetTimestamp(),
				})
				return nil, err
			}

			args = append(args, arg.Elem())
		}

		response := handler.Call(args)

		// convert return values into (interface{}, error)
		var err error
		if len(response) > 0 {
			if errVal, ok := response[len(response)-1].Interface().(error); ok {
				err = errVal
			}
		}
		var val interface{}
		if len(response) > 1 {
			val = response[0].Interface()
		}

		return val, err
	}
}
