package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/config/cli"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

var celCmd = &cobra.Command{
	Use:   "cel",
	Short: "Work with CEL expressions",
	Long:  `Commands for debugging CEL expressions against sample payloads.`,
	Run:   func(cmd *cobra.Command, args []string) { _ = cmd.Help() },
}

var celDebugCmd = &cobra.Command{
	Use:   "debug",
	Short: "Debug a CEL expression",
	Long:  `Evaluate a CEL expression against a sample input payload, optional additional metadata, and an optional filter payload.`,
	Example: `  hatchet cel debug -e "input.value > 5" --input '{"value": 10}'

  hatchet cel debug -e "metadata.env == 'prod'" --input '{}' --metadata '{"env": "prod"}'

  hatchet cel debug -e "payload.enabled" --input '{}' --filter-payload '{"enabled": true}' -o json`,
	Run: func(cmd *cobra.Command, args []string) {
		isJSON := isJSONOutput(cmd)
		_, hatchetClient := clientFromCmd(cmd)
		tenantUUID := clientTenantUUID(hatchetClient)

		expression, _ := cmd.Flags().GetString("expression")
		inputStr, _ := cmd.Flags().GetString("input")
		metadataStr, _ := cmd.Flags().GetString("metadata")
		filterPayloadStr, _ := cmd.Flags().GetString("filter-payload")

		input, err := parseJSONObjectFlag("input", inputStr)
		if err != nil {
			cli.Logger.Fatalf("%v", err)
		}

		body := rest.V1CELDebugRequest{
			Expression: expression,
			Input:      input,
		}
		if metadataStr != "" {
			metadata, mErr := parseJSONObjectFlag("metadata", metadataStr)
			if mErr != nil {
				cli.Logger.Fatalf("%v", mErr)
			}
			body.AdditionalMetadata = &metadata
		}
		if filterPayloadStr != "" {
			filterPayload, fErr := parseJSONObjectFlag("filter-payload", filterPayloadStr)
			if fErr != nil {
				cli.Logger.Fatalf("%v", fErr)
			}
			body.FilterPayload = &filterPayload
		}

		resp, err := hatchetClient.API().V1CelDebugWithResponse(cmd.Context(), tenantUUID, body)
		if err != nil {
			cli.Logger.Fatalf("failed to debug CEL expression: %v", err)
		}
		if resp.JSON200 == nil {
			cli.Logger.Fatalf("failed to debug CEL expression: %s (status %d)", apiErrorMessage(resp.JSON400, resp.JSON403), resp.StatusCode())
		}

		if isJSON {
			printJSON(resp.JSON200)
			return
		}

		result := resp.JSON200
		if result.Status == rest.V1CELDebugResponseStatusSUCCESS {
			output := "<none>"
			if result.Output != nil {
				output = fmt.Sprintf("%t", *result.Output)
			}
			fmt.Println(styles.SuccessMessage(fmt.Sprintf("Expression evaluated successfully, output: %s", output)))
		} else {
			errMsg := "unknown error"
			if result.Error != nil {
				errMsg = *result.Error
			}
			cli.Logger.Fatalf("expression evaluation failed: %s", errMsg)
		}
	},
}

func parseJSONObjectFlag(flagName, value string) (map[string]interface{}, error) {
	if value == "" {
		return map[string]interface{}{}, nil
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(value), &parsed); err != nil {
		return nil, fmt.Errorf("invalid JSON for --%s: %w", flagName, err)
	}
	return parsed, nil
}

func init() {
	rootCmd.AddCommand(celCmd)
	celCmd.AddCommand(celDebugCmd)

	celCmd.PersistentFlags().StringP("profile", "p", "", "Profile to use for connecting to Hatchet (default: prompts for selection)")
	celCmd.PersistentFlags().StringP("output", "o", "", "Output format: json")

	celDebugCmd.Flags().StringP("expression", "e", "", "CEL expression to evaluate")
	celDebugCmd.Flags().String("input", "", "Sample input payload as a JSON object")
	celDebugCmd.Flags().String("metadata", "", "Additional metadata as a JSON object")
	celDebugCmd.Flags().String("filter-payload", "", "Filter payload as a JSON object")
	_ = celDebugCmd.MarkFlagRequired("expression")
}
