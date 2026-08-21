package cli

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/config/cli"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

var alertingCmd = &cobra.Command{
	Use:     "alerting",
	Aliases: []string{"alerts"},
	Short:   "Manage tenant alerting",
	Long:    `Commands for viewing alerting settings and managing email groups, Slack webhooks, and SNS integrations.`,
	Run:     func(cmd *cobra.Command, args []string) { _ = cmd.Help() },
}

var alertingSettingsCmd = &cobra.Command{
	Use:   "settings",
	Short: "Show tenant alerting settings",
	Long:  `Show the tenant alerting settings. Outputs raw JSON.`,
	Run: func(cmd *cobra.Command, args []string) {
		_, hatchetClient := clientFromCmd(cmd)
		tenantUUID := clientTenantUUID(hatchetClient)

		resp, err := hatchetClient.API().TenantAlertingSettingsGetWithResponse(cmd.Context(), tenantUUID)
		if err != nil {
			cli.Logger.Fatalf("failed to get alerting settings: %v", err)
		}
		if resp.JSON200 == nil {
			cli.Logger.Fatalf("unexpected response from API (status %d)", resp.StatusCode())
		}

		printJSON(resp.JSON200)
	},
}

var alertingEmailGroupsCmd = &cobra.Command{
	Use:     "email-groups",
	Aliases: []string{"email-group", "eg"},
	Short:   "Manage alert email groups",
	Long:    `Commands for listing, creating, updating, and deleting alert email groups.`,
	Run:     func(cmd *cobra.Command, args []string) { _ = cmd.Help() },
}

var alertingEmailGroupsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List alert email groups",
	Long:  `List alert email groups. Outputs raw JSON.`,
	Run: func(cmd *cobra.Command, args []string) {
		_, hatchetClient := clientFromCmd(cmd)
		tenantUUID := clientTenantUUID(hatchetClient)

		resp, err := hatchetClient.API().AlertEmailGroupListWithResponse(cmd.Context(), tenantUUID)
		if err != nil {
			cli.Logger.Fatalf("failed to list alert email groups: %v", err)
		}
		if resp.JSON200 == nil {
			cli.Logger.Fatalf("unexpected response from API (status %d)", resp.StatusCode())
		}

		printJSON(resp.JSON200)
	},
}

var alertingEmailGroupsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an alert email group",
	Long:  `Create an alert email group. In --output json mode, --emails is required. Otherwise, launches an interactive form when omitted.`,
	Example: `  # Interactive mode
  hatchet alerting email-groups create --profile local

  # Non-interactive
  hatchet alerting email-groups create --emails a@example.com,b@example.com -o json`,
	Run: func(cmd *cobra.Command, args []string) {
		isJSON := isJSONOutput(cmd)
		_, hatchetClient := clientFromCmd(cmd)

		emails, _ := cmd.Flags().GetStringSlice("emails")
		if !isJSON && len(emails) == 0 {
			emails = promptEmails()
		}
		if len(emails) == 0 {
			cli.Logger.Fatal("--emails is required")
		}

		tenantUUID := clientTenantUUID(hatchetClient)
		resp, err := hatchetClient.API().AlertEmailGroupCreateWithResponse(cmd.Context(), tenantUUID, rest.CreateTenantAlertEmailGroupRequest{
			Emails: emails,
		})
		if err != nil {
			cli.Logger.Fatalf("failed to create alert email group: %v", err)
		}
		if resp.JSON201 == nil {
			cli.Logger.Fatalf("unexpected response from API (status %d)", resp.StatusCode())
		}

		if isJSON {
			printJSON(resp.JSON201)
		} else {
			fmt.Println(styles.SuccessMessage(fmt.Sprintf("Created alert email group %s (%s)", resp.JSON201.Metadata.Id, strings.Join(resp.JSON201.Emails, ", "))))
		}
	},
}

var alertingEmailGroupsUpdateCmd = &cobra.Command{
	Use:   "update <group-id>",
	Short: "Update an alert email group",
	Long:  `Update the emails of an alert email group by ID.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		isJSON := isJSONOutput(cmd)
		_, hatchetClient := clientFromCmd(cmd)

		groupUUID, err := uuid.Parse(args[0])
		if err != nil {
			cli.Logger.Fatalf("invalid email group ID %q: %v", args[0], err)
		}

		emails, _ := cmd.Flags().GetStringSlice("emails")
		if !isJSON && len(emails) == 0 {
			emails = promptEmails()
		}
		if len(emails) == 0 {
			cli.Logger.Fatal("--emails is required")
		}

		resp, err := hatchetClient.API().AlertEmailGroupUpdateWithResponse(cmd.Context(), groupUUID, rest.UpdateTenantAlertEmailGroupRequest{
			Emails: emails,
		})
		if err != nil {
			cli.Logger.Fatalf("failed to update alert email group: %v", err)
		}
		if resp.JSON200 == nil {
			cli.Logger.Fatalf("unexpected response from API (status %d)", resp.StatusCode())
		}

		if isJSON {
			printJSON(resp.JSON200)
		} else {
			fmt.Println(styles.SuccessMessage(fmt.Sprintf("Updated alert email group %s (%s)", args[0], strings.Join(resp.JSON200.Emails, ", "))))
		}
	},
}

var alertingEmailGroupsDeleteCmd = &cobra.Command{
	Use:   "delete <group-id>",
	Short: "Delete an alert email group",
	Long:  `Delete an alert email group by ID. Use --yes to skip confirmation.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		deleteAlertingResource(cmd, args[0], "alert email group", func(id uuid.UUID) (interface{ StatusCode() int }, error) {
			return hatchetAPIFromCmd(cmd).AlertEmailGroupDeleteWithResponse(cmd.Context(), id)
		})
	},
}

var alertingSlackCmd = &cobra.Command{
	Use:   "slack",
	Short: "Manage Slack webhooks for alerts",
	Long: `Commands for listing and deleting Slack webhooks used for alerting.

Slack webhooks are created via the OAuth flow in the Hatchet dashboard and cannot be created with an API token.`,
	Run: func(cmd *cobra.Command, args []string) { _ = cmd.Help() },
}

var alertingSlackListCmd = &cobra.Command{
	Use:   "list",
	Short: "List Slack webhooks",
	Long:  `List Slack webhooks. Outputs raw JSON.`,
	Run: func(cmd *cobra.Command, args []string) {
		_, hatchetClient := clientFromCmd(cmd)
		tenantUUID := clientTenantUUID(hatchetClient)

		resp, err := hatchetClient.API().SlackWebhookListWithResponse(cmd.Context(), tenantUUID)
		if err != nil {
			cli.Logger.Fatalf("failed to list Slack webhooks: %v", err)
		}
		if resp.JSON200 == nil {
			cli.Logger.Fatalf("unexpected response from API (status %d)", resp.StatusCode())
		}

		printJSON(resp.JSON200)
	},
}

var alertingSlackDeleteCmd = &cobra.Command{
	Use:   "delete <webhook-id>",
	Short: "Delete a Slack webhook",
	Long:  `Delete a Slack webhook by ID. Use --yes to skip confirmation.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		deleteAlertingResource(cmd, args[0], "Slack webhook", func(id uuid.UUID) (interface{ StatusCode() int }, error) {
			return hatchetAPIFromCmd(cmd).SlackWebhookDeleteWithResponse(cmd.Context(), id)
		})
	},
}

var alertingSnsCmd = &cobra.Command{
	Use:   "sns",
	Short: "Manage SNS integrations for alerts",
	Long:  `Commands for listing, creating, and deleting SNS integrations.`,
	Run:   func(cmd *cobra.Command, args []string) { _ = cmd.Help() },
}

var alertingSnsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List SNS integrations",
	Long:  `List SNS integrations. Outputs raw JSON.`,
	Run: func(cmd *cobra.Command, args []string) {
		_, hatchetClient := clientFromCmd(cmd)
		tenantUUID := clientTenantUUID(hatchetClient)

		resp, err := hatchetClient.API().SnsListWithResponse(cmd.Context(), tenantUUID)
		if err != nil {
			cli.Logger.Fatalf("failed to list SNS integrations: %v", err)
		}
		if resp.JSON200 == nil {
			cli.Logger.Fatalf("unexpected response from API (status %d)", resp.StatusCode())
		}

		printJSON(resp.JSON200)
	},
}

var alertingSnsCreateCmd = &cobra.Command{
	Use:     "create",
	Short:   "Create an SNS integration",
	Long:    `Create an SNS integration for the given SNS topic ARN.`,
	Example: `  hatchet alerting sns create --topic-arn arn:aws:sns:us-east-1:123456789012:my-topic`,
	Run: func(cmd *cobra.Command, args []string) {
		isJSON := isJSONOutput(cmd)
		_, hatchetClient := clientFromCmd(cmd)

		topicArn, _ := cmd.Flags().GetString("topic-arn")
		if topicArn == "" {
			cli.Logger.Fatal("--topic-arn is required")
		}

		tenantUUID := clientTenantUUID(hatchetClient)
		resp, err := hatchetClient.API().SnsCreateWithResponse(cmd.Context(), tenantUUID, rest.CreateSNSIntegrationRequest{
			TopicArn: topicArn,
		})
		if err != nil {
			cli.Logger.Fatalf("failed to create SNS integration: %v", err)
		}
		if resp.JSON201 == nil {
			cli.Logger.Fatalf("unexpected response from API (status %d)", resp.StatusCode())
		}

		if isJSON {
			printJSON(resp.JSON201)
			return
		}

		fmt.Println(styles.SuccessMessage(fmt.Sprintf("Created SNS integration for %s", resp.JSON201.TopicArn)))
		if resp.JSON201.IngestUrl != nil {
			fmt.Printf("Ingest URL: %s\n", *resp.JSON201.IngestUrl)
		}
	},
}

var alertingSnsDeleteCmd = &cobra.Command{
	Use:   "delete <sns-id>",
	Short: "Delete an SNS integration",
	Long:  `Delete an SNS integration by ID. Use --yes to skip confirmation.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		deleteAlertingResource(cmd, args[0], "SNS integration", func(id uuid.UUID) (interface{ StatusCode() int }, error) {
			return hatchetAPIFromCmd(cmd).SnsDeleteWithResponse(cmd.Context(), id)
		})
	},
}

func hatchetAPIFromCmd(cmd *cobra.Command) *rest.ClientWithResponses {
	_, hatchetClient := clientFromCmd(cmd)
	return hatchetClient.API()
}

func deleteAlertingResource(cmd *cobra.Command, rawID, kind string, del func(uuid.UUID) (interface{ StatusCode() int }, error)) {
	isJSON := isJSONOutput(cmd)
	yes, _ := cmd.Flags().GetBool("yes")

	id, err := uuid.Parse(rawID)
	if err != nil {
		cli.Logger.Fatalf("invalid %s ID %q: %v", kind, rawID, err)
	}

	if !isJSON && !yes {
		if !confirmAction(fmt.Sprintf("Delete %s '%s'?", kind, rawID)) {
			fmt.Println("Aborted.")
			return
		}
	}

	resp, err := del(id)
	if err != nil {
		cli.Logger.Fatalf("failed to delete %s: %v", kind, err)
	}
	if resp.StatusCode() >= 400 {
		cli.Logger.Fatalf("failed to delete %s (status %d)", kind, resp.StatusCode())
	}

	if isJSON {
		printJSON(map[string]interface{}{"deleted": true, "id": rawID})
	} else {
		fmt.Println(styles.SuccessMessage(fmt.Sprintf("Deleted %s: %s", kind, rawID)))
	}
}

func promptEmails() []string {
	var raw string
	form := huh.NewForm(huh.NewGroup(
		huh.NewInput().
			Title("Emails (comma-separated)").
			Value(&raw).
			Placeholder("a@example.com,b@example.com").
			Validate(func(s string) error {
				if strings.TrimSpace(s) == "" {
					return fmt.Errorf("at least one email is required")
				}
				return nil
			}),
	)).WithTheme(styles.HatchetTheme())
	if err := form.Run(); err != nil {
		cli.Logger.Fatalf("form cancelled: %v", err)
	}

	var emails []string
	for _, e := range strings.Split(raw, ",") {
		if trimmed := strings.TrimSpace(e); trimmed != "" {
			emails = append(emails, trimmed)
		}
	}
	return emails
}

func init() {
	rootCmd.AddCommand(alertingCmd)
	alertingCmd.AddCommand(alertingSettingsCmd, alertingEmailGroupsCmd, alertingSlackCmd, alertingSnsCmd)
	alertingEmailGroupsCmd.AddCommand(alertingEmailGroupsListCmd, alertingEmailGroupsCreateCmd, alertingEmailGroupsUpdateCmd, alertingEmailGroupsDeleteCmd)
	alertingSlackCmd.AddCommand(alertingSlackListCmd, alertingSlackDeleteCmd)
	alertingSnsCmd.AddCommand(alertingSnsListCmd, alertingSnsCreateCmd, alertingSnsDeleteCmd)

	alertingCmd.PersistentFlags().StringP("profile", "p", "", "Profile to use for connecting to Hatchet (default: prompts for selection)")
	alertingCmd.PersistentFlags().StringP("output", "o", "", "Output format: json")

	alertingEmailGroupsCreateCmd.Flags().StringSlice("emails", nil, "Comma-separated list of emails for the group")
	alertingEmailGroupsUpdateCmd.Flags().StringSlice("emails", nil, "Comma-separated list of emails for the group")
	alertingEmailGroupsDeleteCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")

	alertingSlackDeleteCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")

	alertingSnsCreateCmd.Flags().String("topic-arn", "", "The Amazon Resource Name (ARN) of the SNS topic")
	alertingSnsDeleteCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")
}
