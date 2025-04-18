package stepruns

import (
	"encoding/json"
	"fmt"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/populator"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/internal/schema"
)

func (t *StepRunService) StepRunGetSchema(ctx echo.Context, request gen.StepRunGetSchemaRequestObject) (gen.StepRunGetSchemaResponseObject, error) {
	populator := populator.FromContext(ctx)

	stepRun, err := populator.GetStepRun()
	if err != nil {
		return nil, err
	}

	var res map[string]interface{}

	input := stepRun.Input

	if input != nil {
		schemaBytes, err := schema.SchemaBytesFromBytes(input)

		if err != nil {
			return nil, fmt.Errorf("could not get schema bytes: %w", err)
		}

		err = json.Unmarshal(schemaBytes, &res)

		if err != nil {
			return nil, fmt.Errorf("could not unmarshal schema: %w", err)
		}
	}

	return gen.StepRunGetSchema200JSONResponse(res), nil
}
