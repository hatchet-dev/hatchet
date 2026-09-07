//go:build !e2e && !load && !rampup && !integration

package serverlessoperator

import (
	"google.golang.org/protobuf/encoding/protojson"

	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

func protojsonMarshal(wf *v1.CreateWorkflowVersionRequest) ([]byte, error) {
	return protojson.Marshal(wf)
}
