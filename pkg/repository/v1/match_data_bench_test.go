package v1

import (
	"encoding/json"
	"testing"
)

func BenchmarkTriggerDataValue_Old(b *testing.B) {
	oldImpl := func(values []interface{}) map[string]interface{} {
		for _, v := range values {
			vBytes, err := json.Marshal(v)
			if err != nil {
				continue
			}

			data := map[string]interface{}{}
			err = json.Unmarshal(vBytes, &data)
			if err != nil {
				continue
			}

			return data
		}
		return nil
	}

	testData := []interface{}{
		map[string]interface{}{
			"key1": "value1",
			"key2": 123,
			"key3": true,
			"nested": map[string]interface{}{
				"inner": "value",
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := oldImpl(testData)
		if result == nil {
			b.Fatal("expected result")
		}
	}
}

func BenchmarkTriggerDataValue_New(b *testing.B) {
	newImpl := func(values []interface{}) map[string]interface{} {
		for _, v := range values {
			if data, ok := v.(map[string]interface{}); ok {
				return data
			}
		}
		return nil
	}

	testData := []interface{}{
		map[string]interface{}{
			"key1": "value1",
			"key2": 123,
			"key3": true,
			"nested": map[string]interface{}{
				"inner": "value",
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := newImpl(testData)
		if result == nil {
			b.Fatal("expected result")
		}
	}
}

func BenchmarkDataValueAsTaskOutputEvent_Old(b *testing.B) {
	oldImpl := func(values []interface{}) *TaskOutputEvent {
		for _, v := range values {
			vBytes, err := json.Marshal(v)
			if err != nil {
				continue
			}

			event := &TaskOutputEvent{}
			err = json.Unmarshal(vBytes, event)
			if err != nil {
				continue
			}

			return event
		}
		return nil
	}

	testData := []interface{}{
		map[string]interface{}{
			"taskId":      int64(123),
			"output":      []byte(`{"result": "success"}`),
			"error":       "",
			"startedAt":   "2024-01-01T00:00:00Z",
			"finishedAt":  "2024-01-01T00:01:00Z",
			"timeoutAt":   "2024-01-01T00:10:00Z",
			"cancelledAt": "",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := oldImpl(testData)
		if result == nil {
			b.Fatal("expected result")
		}
	}
}

func BenchmarkDataValueAsTaskOutputEvent_New(b *testing.B) {
	newImpl := func(values []interface{}) *TaskOutputEvent {
		for _, v := range values {
			if event, ok := v.(*TaskOutputEvent); ok {
				return event
			}
			if eventMap, ok := v.(map[string]interface{}); ok {
				event := &TaskOutputEvent{}
				vBytes, err := json.Marshal(eventMap)
				if err != nil {
					continue
				}
				if err := json.Unmarshal(vBytes, event); err != nil {
					continue
				}
				return event
			}
		}
		return nil
	}

	testData := []interface{}{
		map[string]interface{}{
			"taskId":      int64(123),
			"output":      []byte(`{"result": "success"}`),
			"error":       "",
			"startedAt":   "2024-01-01T00:00:00Z",
			"finishedAt":  "2024-01-01T00:01:00Z",
			"timeoutAt":   "2024-01-01T00:10:00Z",
			"cancelledAt": "",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := newImpl(testData)
		if result == nil {
			b.Fatal("expected result")
		}
	}
}

func BenchmarkDataValueAsTaskOutputEvent_DirectType(b *testing.B) {
	newImpl := func(values []interface{}) *TaskOutputEvent {
		for _, v := range values {
			if event, ok := v.(*TaskOutputEvent); ok {
				return event
			}
			if eventMap, ok := v.(map[string]interface{}); ok {
				event := &TaskOutputEvent{}
				vBytes, err := json.Marshal(eventMap)
				if err != nil {
					continue
				}
				if err := json.Unmarshal(vBytes, event); err != nil {
					continue
				}
				return event
			}
		}
		return nil
	}

	testData := []interface{}{
		&TaskOutputEvent{
			TaskId: 123,
			Output: []byte(`{"result": "success"}`),
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := newImpl(testData)
		if result == nil {
			b.Fatal("expected result")
		}
	}
}
