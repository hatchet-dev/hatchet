package worker

import (
	"context"
	"fmt"
	"reflect"
)

func decodeArgsToInterface(fnType reflect.Type) (result interface{}, err error) {
	if fnType.NumIn() != 2 {
		return nil, fmt.Errorf("fn must have exactly one argument")
	}

	firstArg := fnType.In(0)

	if firstArg.Kind() != reflect.Interface || !firstArg.Implements(reflect.TypeOf((*context.Context)(nil)).Elem()) {
		return nil, fmt.Errorf("first argument must be context.Context")
	}

	// second argument should be a pointer to a struct
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
