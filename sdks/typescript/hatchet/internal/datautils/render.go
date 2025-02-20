package datautils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"text/template"
)

// RenderTemplateFields recursively processes the input map, rendering any string fields using the data map.
func RenderTemplateFields(data map[string]interface{}, input map[string]interface{}) (map[string]interface{}, error) {
	output := map[string]interface{}{}

	for key, val := range input {
		switch v := val.(type) {
		case string:
			tmpl, err := template.New(key).Parse(v)

			if err != nil {
				return nil, fmt.Errorf("error creating template for key %s: %v", key, err)
			}

			var tpl bytes.Buffer
			err = tmpl.Execute(&tpl, data)

			if err != nil {
				return nil, fmt.Errorf("error executing template for key %s: %v", key, err)
			}

			res := tpl.String()

			// if the string can be unmarshalled into a map[string]interface{}, do so
			resMap := map[string]interface{}{}

			if err := json.Unmarshal(tpl.Bytes(), &resMap); err == nil {
				output[key] = resMap

				// if the key is "object", the entire input is replaced with the rendered value
				if key == "object" {
					// note we do not recursively render the new input, as it may contain untrusted data.
					return resMap, nil
				}
			} else {
				output[key] = res
			}
		case map[string]interface{}:
			// if we hit a nested map[string]interface{}, render those recursively
			recOut, err := RenderTemplateFields(data, v)
			if err != nil {
				return nil, err
			}

			output[key] = recOut
		default:
			if reflect.TypeOf(v).Kind() == reflect.Map {
				// If it's a map but not map[string]interface{}, return an error
				return nil, fmt.Errorf("encountered a map that is not map[string]interface{}: %s", key)
			}

			// otherwise, just copy the value over
			output[key] = v
		}
	}

	return output, nil
}
