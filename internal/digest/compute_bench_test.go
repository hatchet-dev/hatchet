package digest

import (
	"encoding/json"
	"testing"
)

type benchmarkStruct struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Jobs        []benchmarkJob         `json:"jobs"`
	Tags        []string               `json:"tags"`
	Metadata    map[string]interface{} `json:"metadata"`
}

type benchmarkJob struct {
	ID       string   `json:"id"`
	Action   string   `json:"action"`
	Timeout  string   `json:"timeout"`
	Parents  []string `json:"parents"`
	Retries  int      `json:"retries"`
	Priority int32    `json:"priority"`
}

func createBenchmarkData() *benchmarkStruct {
	return &benchmarkStruct{
		Name:        "test-workflow",
		Description: "A test workflow with multiple jobs",
		Jobs: []benchmarkJob{
			{
				ID:       "job-1",
				Action:   "action-1",
				Timeout:  "30s",
				Parents:  []string{},
				Retries:  3,
				Priority: 1,
			},
			{
				ID:       "job-2",
				Action:   "action-2",
				Timeout:  "60s",
				Parents:  []string{"job-1"},
				Retries:  5,
				Priority: 2,
			},
			{
				ID:       "job-3",
				Action:   "action-3",
				Timeout:  "120s",
				Parents:  []string{"job-1", "job-2"},
				Retries:  2,
				Priority: 3,
			},
		},
		Tags: []string{"tag1", "tag2", "tag3"},
		Metadata: map[string]interface{}{
			"version": "1.0",
			"author":  "test",
			"env":     "production",
		},
	}
}

func BenchmarkDigestValues_Old(b *testing.B) {
	data := createBenchmarkData()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		jsonBytes, err := json.Marshal(data)
		if err != nil {
			b.Fatal(err)
		}

		var dataMap map[string]interface{}
		err = json.Unmarshal(jsonBytes, &dataMap)
		if err != nil {
			b.Fatal(err)
		}

		_, err = DigestValues(dataMap)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDigestBytes_New(b *testing.B) {
	data := createBenchmarkData()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		jsonBytes, err := json.Marshal(data)
		if err != nil {
			b.Fatal(err)
		}

		_, err = DigestBytes(jsonBytes)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDigestValues_Small(b *testing.B) {
	dataMap := map[string]interface{}{
		"key": "value",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := DigestValues(dataMap)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDigestBytes_Small(b *testing.B) {
	data := map[string]interface{}{
		"key": "value",
	}
	jsonBytes, _ := json.Marshal(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := DigestBytes(jsonBytes)
		if err != nil {
			b.Fatal(err)
		}
	}
}
