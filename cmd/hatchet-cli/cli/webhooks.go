package cli

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/config/cli"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/tui"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

var webhookSourceNames = []rest.V1WebhookSourceName{
	rest.GENERIC,
	rest.GITHUB,
	rest.LINEAR,
	rest.SLACK,
	rest.STRIPE,
	rest.SVIX,
}

var webhookHMACAlgorithms = []rest.V1WebhookHMACAlgorithm{
	rest.MD5,
	rest.SHA1,
	rest.SHA256,
	rest.SHA512,
}

var webhookHMACEncodings = []rest.V1WebhookHMACEncoding{
	rest.BASE64,
	rest.BASE64URL,
	rest.HEX,
}

var webhooksCmd = &cobra.Command{
	Use:     "webhooks",
	Aliases: []string{"webhook"},
	Short:   "Manage webhooks",
	Long:    `Commands for listing, inspecting, creating, updating, and deleting webhooks.`,
	Run:     func(cmd *cobra.Command, args []string) { _ = cmd.Help() },
}

var webhooksListCmd = &cobra.Command{
	Use:   "list",
	Short: "List webhooks",
	Long:  `List webhooks. Without --output json, launches the interactive TUI. With --output json, outputs raw JSON.`,
	Example: `  # Launch interactive TUI (default)
  hatchet webhooks list --profile local

  # JSON output
  hatchet webhooks list -o json`,
	Run: func(cmd *cobra.Command, args []string) {
		isJSON := isJSONOutput(cmd)
		selectedProfile, hatchetClient := clientFromCmd(cmd)

		if !isJSON {
			tuiM := newTUIModel(selectedProfile, hatchetClient)
			tuiM.currentViewType = ViewTypeWebhooks
			tuiM.currentView = tui.NewWebhooksView(tuiM.ctx)
			p := tea.NewProgram(tuiM, tea.WithAltScreen())
			if _, err := p.Run(); err != nil {
				cli.Logger.Fatalf("error running TUI: %v", err)
			}
			return
		}

		ctx := cmd.Context()
		tenantUUID := clientTenantUUID(hatchetClient)
		limit := int64(50)
		offset := int64(0)
		resp, err := hatchetClient.API().V1WebhookListWithResponse(ctx, tenantUUID, &rest.V1WebhookListParams{
			Limit:  &limit,
			Offset: &offset,
		})
		if err != nil {
			cli.Logger.Fatalf("failed to list webhooks: %v", err)
		}
		if resp.JSON200 == nil {
			cli.Logger.Fatalf("unexpected response from API (status %d)", resp.StatusCode())
		}

		printJSON(resp.JSON200)
	},
}

var webhooksGetCmd = &cobra.Command{
	Use:   "get <webhook-name>",
	Short: "Get a webhook",
	Long:  `Get details about a single webhook by name. Outputs raw JSON.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		_, hatchetClient := clientFromCmd(cmd)
		tenantUUID := clientTenantUUID(hatchetClient)

		resp, err := hatchetClient.API().V1WebhookGetWithResponse(cmd.Context(), tenantUUID, name)
		if err != nil {
			cli.Logger.Fatalf("failed to get webhook: %v", err)
		}
		if resp.JSON200 == nil {
			cli.Logger.Fatalf("webhook %q not found (status %d)", name, resp.StatusCode())
		}

		printJSON(resp.JSON200)
	},
}

var webhooksCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a webhook",
	Long:  `Create an incoming webhook. All configuration is provided via flags.`,
	Example: `  # Basic auth webhook
  hatchet webhooks create --name my-hook --source GENERIC --event-key-expression "input.event" \
    --auth-type basic --username user --password pass

  # API key webhook
  hatchet webhooks create --name my-hook --source STRIPE --event-key-expression "input.type" \
    --auth-type api-key --api-key-header X-Api-Key --api-key secret

  # HMAC webhook
  hatchet webhooks create --name my-hook --source GITHUB --event-key-expression "input.action" \
    --auth-type hmac --hmac-signing-secret secret --hmac-signature-header X-Hub-Signature-256 \
    --hmac-algorithm SHA256 --hmac-encoding HEX`,
	Run: func(cmd *cobra.Command, args []string) {
		isJSON := isJSONOutput(cmd)
		_, hatchetClient := clientFromCmd(cmd)
		tenantUUID := clientTenantUUID(hatchetClient)

		name, _ := cmd.Flags().GetString("name")
		sourceStr, _ := cmd.Flags().GetString("source")
		eventKeyExpression, _ := cmd.Flags().GetString("event-key-expression")
		authTypeStr, _ := cmd.Flags().GetString("auth-type")

		if name == "" {
			cli.Logger.Fatal("--name is required")
		}
		if eventKeyExpression == "" {
			cli.Logger.Fatal("--event-key-expression is required")
		}

		source, err := parseWebhookSourceName(sourceStr)
		if err != nil {
			cli.Logger.Fatalf("%v", err)
		}

		var body rest.V1CreateWebhookRequest

		switch strings.ToLower(strings.ReplaceAll(strings.TrimSpace(authTypeStr), "_", "-")) {
		case "basic":
			username, _ := cmd.Flags().GetString("username")
			password, _ := cmd.Flags().GetString("password")
			if username == "" || password == "" {
				cli.Logger.Fatal("--username and --password are required for basic auth")
			}
			err = body.FromV1CreateWebhookRequestBasicAuth(rest.V1CreateWebhookRequestBasicAuth{
				AuthType:           rest.BASIC,
				Auth:               rest.V1WebhookBasicAuth{Username: username, Password: password},
				Name:               name,
				SourceName:         source,
				EventKeyExpression: eventKeyExpression,
			})
		case "api-key", "apikey":
			apiKeyHeader, _ := cmd.Flags().GetString("api-key-header")
			apiKey, _ := cmd.Flags().GetString("api-key")
			if apiKeyHeader == "" || apiKey == "" {
				cli.Logger.Fatal("--api-key-header and --api-key are required for api-key auth")
			}
			err = body.FromV1CreateWebhookRequestAPIKey(rest.V1CreateWebhookRequestAPIKey{
				AuthType:           rest.V1CreateWebhookRequestAPIKeyAuthTypeAPIKEY,
				Auth:               rest.V1WebhookAPIKeyAuth{HeaderName: apiKeyHeader, ApiKey: apiKey},
				Name:               name,
				SourceName:         source,
				EventKeyExpression: eventKeyExpression,
			})
		case "hmac":
			signingSecret, _ := cmd.Flags().GetString("hmac-signing-secret")
			signatureHeader, _ := cmd.Flags().GetString("hmac-signature-header")
			algorithmStr, _ := cmd.Flags().GetString("hmac-algorithm")
			encodingStr, _ := cmd.Flags().GetString("hmac-encoding")
			if signingSecret == "" || signatureHeader == "" {
				cli.Logger.Fatal("--hmac-signing-secret and --hmac-signature-header are required for hmac auth")
			}
			algorithm, algErr := parseWebhookHMACAlgorithm(algorithmStr)
			if algErr != nil {
				cli.Logger.Fatalf("%v", algErr)
			}
			encoding, encErr := parseWebhookHMACEncoding(encodingStr)
			if encErr != nil {
				cli.Logger.Fatalf("%v", encErr)
			}
			err = body.FromV1CreateWebhookRequestHMAC(rest.V1CreateWebhookRequestHMAC{
				AuthType: rest.HMAC,
				Auth: rest.V1WebhookHMACAuth{
					SigningSecret:       signingSecret,
					SignatureHeaderName: signatureHeader,
					Algorithm:           algorithm,
					Encoding:            encoding,
				},
				Name:               name,
				SourceName:         source,
				EventKeyExpression: eventKeyExpression,
			})
		default:
			cli.Logger.Fatalf("invalid auth type %q (must be one of: basic, api-key, hmac)", authTypeStr)
		}

		if err != nil {
			cli.Logger.Fatalf("failed to build webhook request: %v", err)
		}

		resp, err := hatchetClient.API().V1WebhookCreateWithResponse(cmd.Context(), tenantUUID, body)
		if err != nil {
			cli.Logger.Fatalf("failed to create webhook: %v", err)
		}
		if resp.JSON200 == nil {
			cli.Logger.Fatalf("failed to create webhook: %s (status %d)", apiErrorMessage(resp.JSON400, resp.JSON403, resp.JSON404), resp.StatusCode())
		}

		if !isJSON {
			fmt.Println(styles.SuccessMessage(fmt.Sprintf("Created webhook: %s", resp.JSON200.Name)))
		}
		printJSON(resp.JSON200)
	},
}

var webhooksUpdateCmd = &cobra.Command{
	Use:   "update <webhook-name>",
	Short: "Update a webhook",
	Long:  `Update a webhook by name. Only the provided flags are changed.`,
	Args:  cobra.ExactArgs(1),
	Example: `  hatchet webhooks update my-hook --event-key-expression "input.event_type"
  hatchet webhooks update my-hook --scope-expression "input.tenant" --return-event-as-response-payload`,
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		isJSON := isJSONOutput(cmd)
		_, hatchetClient := clientFromCmd(cmd)
		tenantUUID := clientTenantUUID(hatchetClient)

		body := rest.V1UpdateWebhookRequest{}
		changed := false

		if cmd.Flags().Changed("event-key-expression") {
			v, _ := cmd.Flags().GetString("event-key-expression")
			body.EventKeyExpression = &v
			changed = true
		}
		if cmd.Flags().Changed("scope-expression") {
			v, _ := cmd.Flags().GetString("scope-expression")
			body.ScopeExpression = &v
			changed = true
		}
		if cmd.Flags().Changed("return-event-as-response-payload") {
			v, _ := cmd.Flags().GetBool("return-event-as-response-payload")
			body.ReturnEventAsResponsePayload = &v
			changed = true
		}
		if cmd.Flags().Changed("static-payload") {
			v, _ := cmd.Flags().GetString("static-payload")
			payload, err := parseJSONObjectFlag("static-payload", v)
			if err != nil {
				cli.Logger.Fatalf("%v", err)
			}
			body.StaticPayload = &payload
			changed = true
		}

		if !changed {
			cli.Logger.Fatal("no fields to update; provide at least one flag")
		}

		resp, err := hatchetClient.API().V1WebhookUpdateWithResponse(cmd.Context(), tenantUUID, name, body)
		if err != nil {
			cli.Logger.Fatalf("failed to update webhook: %v", err)
		}
		if resp.JSON200 == nil {
			cli.Logger.Fatalf("failed to update webhook: %s (status %d)", apiErrorMessage(resp.JSON400, resp.JSON403, resp.JSON404), resp.StatusCode())
		}

		if isJSON {
			printJSON(resp.JSON200)
		} else {
			fmt.Println(styles.SuccessMessage(fmt.Sprintf("Updated webhook: %s", resp.JSON200.Name)))
		}
	},
}

var webhooksDeleteCmd = &cobra.Command{
	Use:   "delete <webhook-name>",
	Short: "Delete a webhook",
	Long:  `Delete a webhook by name. Use --yes to skip confirmation.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		isJSON := isJSONOutput(cmd)
		yes, _ := cmd.Flags().GetBool("yes")
		_, hatchetClient := clientFromCmd(cmd)
		tenantUUID := clientTenantUUID(hatchetClient)

		if !isJSON && !yes {
			if !confirmAction(fmt.Sprintf("Delete webhook '%s'?", name)) {
				fmt.Println("Aborted.")
				return
			}
		}

		resp, err := hatchetClient.API().V1WebhookDeleteWithResponse(cmd.Context(), tenantUUID, name)
		if err != nil {
			cli.Logger.Fatalf("failed to delete webhook: %v", err)
		}
		if resp.StatusCode() >= 400 {
			cli.Logger.Fatalf("failed to delete webhook: %s (status %d)", apiErrorMessage(resp.JSON400, resp.JSON403, resp.JSON404), resp.StatusCode())
		}

		if isJSON {
			printJSON(map[string]interface{}{"deleted": true, "name": name})
		} else {
			fmt.Println(styles.SuccessMessage(fmt.Sprintf("Deleted webhook: %s", name)))
		}
	},
}

func parseWebhookSourceName(s string) (rest.V1WebhookSourceName, error) {
	v := rest.V1WebhookSourceName(strings.ToUpper(strings.TrimSpace(s)))
	for _, valid := range webhookSourceNames {
		if v == valid {
			return v, nil
		}
	}
	names := make([]string, len(webhookSourceNames))
	for i, valid := range webhookSourceNames {
		names[i] = string(valid)
	}
	return "", fmt.Errorf("invalid source %q (must be one of: %s)", s, strings.Join(names, ", "))
}

func parseWebhookHMACAlgorithm(s string) (rest.V1WebhookHMACAlgorithm, error) {
	v := rest.V1WebhookHMACAlgorithm(strings.ToUpper(strings.TrimSpace(s)))
	for _, valid := range webhookHMACAlgorithms {
		if v == valid {
			return v, nil
		}
	}
	names := make([]string, len(webhookHMACAlgorithms))
	for i, valid := range webhookHMACAlgorithms {
		names[i] = string(valid)
	}
	return "", fmt.Errorf("invalid hmac algorithm %q (must be one of: %s)", s, strings.Join(names, ", "))
}

func parseWebhookHMACEncoding(s string) (rest.V1WebhookHMACEncoding, error) {
	v := rest.V1WebhookHMACEncoding(strings.ToUpper(strings.TrimSpace(s)))
	for _, valid := range webhookHMACEncodings {
		if v == valid {
			return v, nil
		}
	}
	names := make([]string, len(webhookHMACEncodings))
	for i, valid := range webhookHMACEncodings {
		names[i] = string(valid)
	}
	return "", fmt.Errorf("invalid hmac encoding %q (must be one of: %s)", s, strings.Join(names, ", "))
}

func apiErrorMessage(errs ...*rest.APIErrors) string {
	for _, e := range errs {
		if e == nil {
			continue
		}
		msgs := make([]string, 0, len(e.Errors))
		for _, apiErr := range e.Errors {
			msgs = append(msgs, apiErr.Description)
		}
		if len(msgs) > 0 {
			return strings.Join(msgs, "; ")
		}
	}
	return "unexpected response from API"
}

func init() {
	rootCmd.AddCommand(webhooksCmd)
	webhooksCmd.AddCommand(webhooksListCmd, webhooksGetCmd, webhooksCreateCmd, webhooksUpdateCmd, webhooksDeleteCmd)

	webhooksCmd.PersistentFlags().StringP("profile", "p", "", "Profile to use for connecting to Hatchet (default: prompts for selection)")
	webhooksCmd.PersistentFlags().StringP("output", "o", "", "Output format: json (skips interactive TUI)")

	webhooksCreateCmd.Flags().StringP("name", "n", "", "Name of the webhook")
	webhooksCreateCmd.Flags().StringP("source", "s", string(rest.GENERIC), "Webhook source: GENERIC, GITHUB, LINEAR, SLACK, STRIPE, SVIX")
	webhooksCreateCmd.Flags().String("event-key-expression", "", "CEL expression used to create the event key from the webhook payload")
	webhooksCreateCmd.Flags().String("auth-type", "", "Authentication type: basic, api-key, hmac")
	webhooksCreateCmd.Flags().String("username", "", "Username for basic auth")
	webhooksCreateCmd.Flags().String("password", "", "Password for basic auth")
	webhooksCreateCmd.Flags().String("api-key-header", "", "Header name for api-key auth")
	webhooksCreateCmd.Flags().String("api-key", "", "API key for api-key auth")
	webhooksCreateCmd.Flags().String("hmac-signing-secret", "", "Signing secret for hmac auth")
	webhooksCreateCmd.Flags().String("hmac-signature-header", "", "Signature header name for hmac auth")
	webhooksCreateCmd.Flags().String("hmac-algorithm", string(rest.SHA256), "HMAC algorithm: MD5, SHA1, SHA256, SHA512")
	webhooksCreateCmd.Flags().String("hmac-encoding", string(rest.HEX), "HMAC encoding: BASE64, BASE64URL, HEX")

	webhooksUpdateCmd.Flags().String("event-key-expression", "", "CEL expression used to create the event key from the webhook payload")
	webhooksUpdateCmd.Flags().String("scope-expression", "", "CEL expression used to filter the correct workflow to trigger")
	webhooksUpdateCmd.Flags().Bool("return-event-as-response-payload", false, "Return the triggered event as the response payload")
	webhooksUpdateCmd.Flags().String("static-payload", "", "Static payload to send with the webhook, as a JSON object")

	webhooksDeleteCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")
}
