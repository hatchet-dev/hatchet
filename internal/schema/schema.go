package schema

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"

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
	for key, val := range obj {
		fieldType := parse(val)
		defaultValue := formatDefaultValue(val)
		field := reflect.StructField{
			Name: toExportedName(key),
			Type: fieldType,
			Tag:  reflect.StructTag(fmt.Sprintf(`json:"%s" jsonschema:"default=%s"`, key, defaultValue)),
		}
		fields = append(fields, field)
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
		// Complex types like arrays or objects are not supported as default values in jsonschema tags.
		return ""
	}
}

// parseArray handles JSON arrays by determining the type of its elements.
func parseArray(arr []interface{}) reflect.Type {
	if len(arr) == 0 {
		return reflect.SliceOf(reflect.TypeOf(new(interface{})).Elem())
	}
	elemType := parse(arr[0])
	return reflect.SliceOf(elemType)
}

var nameRegex = regexp.MustCompile(`(\b|-|_|\.)[a-z]`)

var invalidCharRegex = regexp.MustCompile(`[^a-zA-Z0-9_]`)

// toExportedName converts a JSON key into an exported Go field name.
func toExportedName(key string) string {
	res := nameRegex.ReplaceAllStringFunc(key, func(t string) string {
		if len(t) == 1 {
			return strings.ToUpper(t)
		}

		return strings.ToUpper(string(t[1]))
	})

	return invalidCharRegex.ReplaceAllString(res, "")
}
