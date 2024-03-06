package schema

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/invopop/jsonschema"
)

func SchemaBytesFromBytes(d []byte) ([]byte, error) {
	var m map[string]interface{}
	if err := json.Unmarshal(d, &m); err != nil {
		return nil, err
	}
	return SchemaBytesFromMap(m)
}

func SchemaBytesFromMap(m map[string]interface{}) ([]byte, error) {
	goType := parse(m)

	// create instance of reflect type
	t := reflect.New(goType).Elem()

	s := jsonschema.Reflect(t.Interface())

	return json.Marshal(s)
}

// parse recursively generates a reflect.Type from the given data.
func parse(data interface{}) reflect.Type {
	switch v := data.(type) {
	case map[string]interface{}:
		return parseObject(v)
	case []interface{}:
		return parseArray(v)
	case string:
		return reflect.TypeOf("")
	case float64:
		// Check if it can be an int.
		if v == float64(int(v)) {
			return reflect.TypeOf(0)
		}
		return reflect.TypeOf(0.0)
	case bool:
		return reflect.TypeOf(false)
	case nil:
		return reflect.TypeOf(new(interface{})).Elem()
	default:
		fmt.Printf("Unhandled type: %T\n", v)
		return reflect.TypeOf(new(interface{})).Elem()
	}
}

// parseObject handles JSON objects by creating a struct type with appropriately typed fields.
func parseObject(obj map[string]interface{}) reflect.Type {
	var fields []reflect.StructField
	count := 0

	for key, val := range obj {
		fieldType := parse(val)
		defaultValue := formatDefaultValue(val)

		tag := fmt.Sprintf(`json:"%s" jsonschema:"default=%s"`, key, defaultValue)

		if defaultValue == "" {
			tag = fmt.Sprintf(`json:"%s"`, key)
		}

		field := reflect.StructField{
			Name: fmt.Sprintf("Field%d", count),
			Type: fieldType,
			Tag:  reflect.StructTag(tag),
		}
		fields = append(fields, field)
		count++
	}
	return reflect.StructOf(fields)
}

func formatDefaultValue(val interface{}) string {
	switch v := val.(type) {
	case string:
		return v
	case float64, bool:
		return fmt.Sprintf("%v", v)
	case nil:
		return "null"
	default:
		return ""
	}
}

// parseArray handles JSON arrays by determining the type of its elements.
func parseArray(arr []interface{}) reflect.Type {
	if len(arr) == 0 {
		return reflect.SliceOf(reflect.TypeOf(""))
	}
	elemType := parse(arr[0])
	return reflect.SliceOf(elemType)
}
