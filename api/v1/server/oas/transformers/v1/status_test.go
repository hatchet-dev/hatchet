package transformers

import (
	"testing"

	"github.com/hatchet-dev/hatchet/pkg/enums"
)

func TestMapOlapStatusCoversEveryDatabaseStatus(t *testing.T) {
	for _, s := range enums.AllV1ReadableStatusOlapValues() {
		status, evicted := mapOlapStatus(s)
		if !status.Valid() {
			t.Errorf("database status %s maps to %q, which is not a V1TaskStatus", s, status)
		}
		if evicted != (s == enums.V1ReadableStatusOlapEVICTED) {
			t.Errorf("database status %s: evicted = %v", s, evicted)
		}
	}
}
