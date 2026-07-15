package rbac

import (
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
	"go.yaml.in/yaml/v3"
)

type BearerPolicy struct {
	Operations BearerOperations
}

type BearerOperations struct {
	Denied  []string
	Allowed []string
}

func LoadBearerPolicy(yamlBytes []byte) (*BearerPolicy, error) {
	var policy BearerPolicy

	if err := yaml.Unmarshal(yamlBytes, &policy); err != nil {
		return nil, err
	}

	return &policy, nil
}

func (p *BearerPolicy) IsDenied(operationId string) bool {
	return OperationIn(operationId, p.Operations.Denied)
}

func (p *BearerPolicy) ValidateSpec(spec openapi3.T) error {
	specOperations := map[string]struct{}{}

	for _, pathItem := range spec.Paths.Map() {
		for _, op := range pathItem.Operations() {
			specOperations[op.OperationID] = struct{}{}
		}
	}

	classified := map[string]int{}

	for _, operationId := range p.Operations.Denied {
		classified[operationId]++
	}

	for _, operationId := range p.Operations.Allowed {
		classified[operationId]++
	}

	for operationId, count := range classified {
		if count > 1 {
			return &RBACError{
				Message: fmt.Sprintf("%s is listed more than once in bearer.yaml", operationId),
			}
		}

		if _, ok := specOperations[operationId]; !ok {
			return &RBACError{
				Message: fmt.Sprintf("%s exists in bearer.yaml but not in specs", operationId),
			}
		}
	}

	for operationId := range specOperations {
		if _, ok := classified[operationId]; !ok {
			return &RBACError{
				Message: fmt.Sprintf(
					"%s exists in openapi specs but not in bearer.yaml: add it to operations.denied if it needs a user session, or to operations.allowed if api tokens may call it",
					operationId,
				),
			}
		}
	}

	return nil
}
