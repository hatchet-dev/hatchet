package datautils

import (
	"bytes"
	"fmt"
	"reflect"
	"text/template"
)

// RenderTemplateFields recursively processes the input map, rendering any string fields using the data map.
func RenderTemplateFields(data map[string]interface{}, input map[string]interface{}) error {
	for key, val := range input {
		switch v := val.(type) {
		case string:
			tmpl, err := template.New(key).Parse(v)

			if err != nil {
				return fmt.Errorf("error creating template for key %s: %v", key, err)
			}

			var tpl bytes.Buffer
			err = tmpl.Execute(&tpl, data)

			if err != nil {
				return fmt.Errorf("error executing template for key %s: %v", key, err)
			}
			input[key] = tpl.String()
		case map[string]interface{}:
			// if we hit a nested map[string]interface{}, render those recursively
			err := RenderTemplateFields(data, v)
			if err != nil {
				return err
			}
		default:
			if reflect.TypeOf(v).Kind() == reflect.Map {
				// If it's a map but not map[string]interface{}, return an error
				return fmt.Errorf("encountered a map that is not map[string]interface{}: %s", key)
			}
		}
	}

	return nil
}
