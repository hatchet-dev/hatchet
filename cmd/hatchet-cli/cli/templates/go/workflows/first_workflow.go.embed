package workflows

import (
	"fmt"
	"strings"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type SimpleInput struct {
	Message string `json:"message"`
}

type SimpleOutput struct {
	TransformedMessage string `json:"transformed_message"`
}

func FirstWorkflow(c *hatchet.Client) *hatchet.StandaloneTask {
	return c.NewStandaloneTask("first-workflow", func(ctx hatchet.Context, input SimpleInput) (SimpleOutput, error) {
		fmt.Println("ToLower task called")

		return SimpleOutput{
			TransformedMessage: strings.ToLower(input.Message),
		}, nil
	})
}
