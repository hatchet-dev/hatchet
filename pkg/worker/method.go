package worker

import (
	"context"
	"fmt"
	"reflect"
)

func getFnFromMethod(method any) (result actionFunc, err error) {
	// if not a function type, return error
	methodType := reflect.TypeOf(method)

	if methodType.Kind() != reflect.Func {
		return nil, fmt.Errorf("method must be a function")
	}

	// if not a function with two arguments, return error
	if methodType.NumIn() != 2 {
		return nil, fmt.Errorf("method must have exactly two arguments")
	}

	// if first argument is not a context, return error
	firstArg := methodType.In(0)

	if firstArg.Kind() != reflect.Interface || !firstArg.Implements(reflect.TypeOf((*context.Context)(nil)).Elem()) {
		return nil, fmt.Errorf("first argument must be context.Context")
	}

	// if second argument is not a pointer to a struct, return error
	secondArg := methodType.In(1)

	if secondArg.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("second argument must be a pointer to a struct")
	}

	secondArgElem := secondArg.Elem()

	if secondArgElem.Kind() != reflect.Struct {
		return nil, fmt.Errorf("second argument must be a pointer to a struct")
	}

	// if function does not return two values, return error
	if methodType.NumOut() == 2 {
		// if first return value is not a pointer to a struct, return error
		firstReturn := methodType.Out(0)

		if firstReturn.Kind() != reflect.Ptr {
			return nil, fmt.Errorf("first return value must be a pointer to a struct")
		}

		firstReturnElem := firstReturn.Elem()

		if firstReturnElem.Kind() != reflect.Struct {
			return nil, fmt.Errorf("first return value must be a pointer to a struct")
		}
	}

	// if last return value is not an error, return error
	lastReturn := methodType.Out(methodType.NumOut() - 1)

	if lastReturn.Kind() != reflect.Interface || !lastReturn.Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		return nil, fmt.Errorf("second return value must be of type error")
	}

	return func(args ...interface{}) []interface{} {
		// Ensure args length is correct
		if len(args) != 2 {
			return []interface{}{nil, fmt.Errorf("expected exactly two arguments, got %d", len(args))}
		}

		// Call the method with reflection
		values := reflect.ValueOf(method).Call([]reflect.Value{
			reflect.ValueOf(args[0]),
			reflect.ValueOf(args[1]),
		})

		// Return the results as an interface slice
		res := []interface{}{}

		for i := range values {
			res = append(res, values[i].Interface())
		}

		return res
	}, nil
}
