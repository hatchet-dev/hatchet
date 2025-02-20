package worker

import (
	"context"
	"fmt"
	"reflect"
)

func decodeArgsToInterface(fnType reflect.Type) (results interface{}, err error) {
	numIn := fnType.NumIn()
	if numIn != 1 && numIn != 2 {
		return nil, fmt.Errorf("fn must have exactly one or two arguments")
	}

	firstArg := fnType.In(0)

	if firstArg.Kind() != reflect.Interface || !firstArg.Implements(reflect.TypeOf((*context.Context)(nil)).Elem()) {
		return nil, fmt.Errorf("first argument must implement context.Context")
	}

	// second argument should be a pointer to a struct
	if numIn == 2 {
		secondArg := fnType.In(1)

		if secondArg.Kind() != reflect.Ptr {
			return nil, fmt.Errorf("second argument must be a pointer to a struct")
		}

		secondArgElem := secondArg.Elem()

		if secondArgElem.Kind() != reflect.Struct {
			return nil, fmt.Errorf("second argument must be a pointer to a struct")
		}

		return reflect.New(secondArgElem).Interface(), nil
	}

	return nil, nil
}

func decodeFnArgTypes(fnType reflect.Type) (result []reflect.Type, err error) {
	if fnType.Kind() != reflect.Func {
		return nil, fmt.Errorf("method must be a function")
	}

	numIn := fnType.NumIn()

	// if not a function with one or two arguments, return an error
	if numIn != 1 && numIn != 2 {
		return nil, fmt.Errorf("method must have one or two arguments")
	}

	// if first argument is not a context, return error
	firstArg := fnType.In(0)

	if firstArg.Kind() != reflect.Interface || !firstArg.Implements(reflect.TypeOf((*context.Context)(nil)).Elem()) {
		return nil, fmt.Errorf("first argument must be context.Context")
	}

	res := []reflect.Type{firstArg}

	if numIn == 2 {
		// if second argument is not a pointer to a struct, return error
		secondArg := fnType.In(1)

		if secondArg.Kind() != reflect.Ptr {
			return nil, fmt.Errorf("second argument must be a pointer to a struct")
		}

		secondArgElem := secondArg.Elem()

		if secondArgElem.Kind() != reflect.Struct {
			return nil, fmt.Errorf("second argument must be a pointer to a struct")
		}

		res = append(res, secondArg)
	}

	return res, nil
}

func decodeFnReturnTypes(fnType reflect.Type) (result []reflect.Type, err error) {
	if fnType.NumOut() > 2 {
		return nil, fmt.Errorf("fn cannot have more than 2 return values")
	}

	firstOut := fnType.Out(0)

	// if there are two args, the first one should be a pointer to a struct
	if fnType.NumOut() == 2 {
		if firstOut.Kind() != reflect.Ptr {
			return nil, fmt.Errorf("first argument must be a pointer to a struct when there are two return values")
		}

		firstOutElem := firstOut.Elem()

		if firstOutElem.Kind() != reflect.Struct {
			return nil, fmt.Errorf("first argument must be a pointer to a struct when there are two return values")
		}
	}

	lastOut := fnType.Out(fnType.NumOut() - 1)

	if lastOut.Kind() != reflect.Interface || !lastOut.Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		return nil, fmt.Errorf("last return value must be error")
	}

	if fnType.NumOut() == 1 {
		return []reflect.Type{firstOut}, nil
	}

	return []reflect.Type{firstOut, lastOut}, nil
}
