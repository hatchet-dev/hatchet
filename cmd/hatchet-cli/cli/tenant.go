package cli

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/config/cli"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

var tenantMemberRoles = []rest.TenantMemberRole{
	rest.OWNER,
	rest.ADMIN,
	rest.MEMBER,
}

func parseTenantMemberRole(s string) (rest.TenantMemberRole, error) {
	r := rest.TenantMemberRole(strings.ToUpper(strings.TrimSpace(s)))
	for _, valid := range tenantMemberRoles {
		if r == valid {
			return r, nil
		}
	}
	names := make([]string, len(tenantMemberRoles))
	for i, valid := range tenantMemberRoles {
		names[i] = string(valid)
	}
	return "", fmt.Errorf("invalid role %q (must be one of: %s)", s, strings.Join(names, ", "))
}

func tenantMemberRoleOptions() []huh.Option[string] {
	options := make([]huh.Option[string], 0, len(tenantMemberRoles))
	for _, r := range tenantMemberRoles {
		options = append(options, huh.NewOption(string(r), string(r)))
	}
	return options
}

func parseUUIDArg(s, what string) openapi_types.UUID {
	parsed, err := uuid.Parse(s)
	if err != nil {
		cli.Logger.Fatalf("invalid %s %q: %v", what, s, err)
	}
	return parsed
}

var tenantCmd = &cobra.Command{
	Use:   "tenant",
	Short: "Manage the current tenant",
	Long:  `Commands for viewing and updating the current tenant, its members, and invites.`,
	Run:   func(cmd *cobra.Command, args []string) { _ = cmd.Help() },
}

var tenantGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get the current tenant",
	Long:  `Get details about the current tenant.`,
	Run: func(cmd *cobra.Command, args []string) {
		_, hatchetClient := clientFromCmd(cmd)
		tenantUUID := clientTenantUUID(hatchetClient)

		resp, err := hatchetClient.API().TenantGetWithResponse(cmd.Context(), tenantUUID)
		if err != nil {
			cli.Logger.Fatalf("failed to get tenant: %v", err)
		}
		if resp.JSON200 == nil {
			cli.Logger.Fatalf("unexpected response from API (status %d)", resp.StatusCode())
		}

		if isJSONOutput(cmd) {
			printJSON(resp.JSON200)
			return
		}

		tenant := resp.JSON200
		fmt.Println(styles.SuccessMessage(fmt.Sprintf("Tenant: %s", tenant.Name)))
		fmt.Printf("  ID:      %s\n", tenant.Metadata.Id)
		fmt.Printf("  Slug:    %s\n", tenant.Slug)
		fmt.Printf("  Version: %s\n", tenant.Version)
		if tenant.Environment != nil {
			fmt.Printf("  Environment: %s\n", *tenant.Environment)
		}
		if tenant.Region != nil {
			fmt.Printf("  Region:  %s\n", *tenant.Region)
		}
		if tenant.AnalyticsOptOut != nil {
			fmt.Printf("  Analytics opt-out: %t\n", *tenant.AnalyticsOptOut)
		}
		if tenant.AlertMemberEmails != nil {
			fmt.Printf("  Alert member emails: %t\n", *tenant.AlertMemberEmails)
		}
	},
}

var tenantUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update the current tenant",
	Long:  `Update settings on the current tenant. Only flags that are explicitly set are sent to the API.`,
	Example: `  hatchet tenant update --name my-tenant
  hatchet tenant update --analytics-opt-out --max-alerting-frequency 1h`,
	Run: func(cmd *cobra.Command, args []string) {
		_, hatchetClient := clientFromCmd(cmd)
		tenantUUID := clientTenantUUID(hatchetClient)

		body := rest.TenantUpdateJSONRequestBody{}
		changed := false

		if cmd.Flags().Changed("name") {
			name, _ := cmd.Flags().GetString("name")
			body.Name = &name
			changed = true
		}
		if cmd.Flags().Changed("analytics-opt-out") {
			v, _ := cmd.Flags().GetBool("analytics-opt-out")
			body.AnalyticsOptOut = &v
			changed = true
		}
		if cmd.Flags().Changed("alert-member-emails") {
			v, _ := cmd.Flags().GetBool("alert-member-emails")
			body.AlertMemberEmails = &v
			changed = true
		}
		if cmd.Flags().Changed("enable-expiring-token-alerts") {
			v, _ := cmd.Flags().GetBool("enable-expiring-token-alerts")
			body.EnableExpiringTokenAlerts = &v
			changed = true
		}
		if cmd.Flags().Changed("enable-resource-limit-alerts") {
			v, _ := cmd.Flags().GetBool("enable-resource-limit-alerts")
			body.EnableTenantResourceLimitAlerts = &v
			changed = true
		}
		if cmd.Flags().Changed("enable-run-failure-alerts") {
			v, _ := cmd.Flags().GetBool("enable-run-failure-alerts")
			body.EnableWorkflowRunFailureAlerts = &v
			changed = true
		}
		if cmd.Flags().Changed("max-alerting-frequency") {
			v, _ := cmd.Flags().GetString("max-alerting-frequency")
			body.MaxAlertingFrequency = &v
			changed = true
		}
		if cmd.Flags().Changed("version") {
			v, _ := cmd.Flags().GetString("version")
			version := rest.TenantVersion(strings.ToUpper(strings.TrimSpace(v)))
			if version != rest.V0 && version != rest.V1 {
				cli.Logger.Fatalf("invalid version %q (must be one of: V0, V1)", v)
			}
			body.Version = &version
			changed = true
		}

		if !changed {
			cli.Logger.Fatal("no fields to update; set at least one flag")
		}

		resp, err := hatchetClient.API().TenantUpdateWithResponse(cmd.Context(), tenantUUID, body)
		if err != nil {
			cli.Logger.Fatalf("failed to update tenant: %v", err)
		}
		if resp.JSON200 == nil {
			cli.Logger.Fatalf("unexpected response from API (status %d)", resp.StatusCode())
		}

		if isJSONOutput(cmd) {
			printJSON(resp.JSON200)
		} else {
			fmt.Println(styles.SuccessMessage(fmt.Sprintf("Updated tenant: %s", resp.JSON200.Name)))
		}
	},
}

var tenantResourcePolicyCmd = &cobra.Command{
	Use:   "resource-policy",
	Short: "Get the tenant resource policy",
	Long:  `Get the resource policy (limits) for the current tenant. Outputs raw JSON.`,
	Run: func(cmd *cobra.Command, args []string) {
		_, hatchetClient := clientFromCmd(cmd)
		tenantUUID := clientTenantUUID(hatchetClient)

		resp, err := hatchetClient.API().TenantResourcePolicyGetWithResponse(cmd.Context(), tenantUUID)
		if err != nil {
			cli.Logger.Fatalf("failed to get resource policy: %v", err)
		}
		if resp.JSON200 == nil {
			cli.Logger.Fatalf("unexpected response from API (status %d)", resp.StatusCode())
		}

		printJSON(resp.JSON200)
	},
}

var tenantQueueMetricsCmd = &cobra.Command{
	Use:   "queue-metrics",
	Short: "Get tenant queue metrics",
	Long:  `Get queue metrics for the current tenant. Use --step-runs for step run queue metrics instead. Outputs raw JSON.`,
	Run: func(cmd *cobra.Command, args []string) {
		_, hatchetClient := clientFromCmd(cmd)
		tenantUUID := clientTenantUUID(hatchetClient)
		ctx := cmd.Context()

		stepRuns, _ := cmd.Flags().GetBool("step-runs")
		if stepRuns {
			resp, err := hatchetClient.API().TenantGetStepRunQueueMetricsWithResponse(ctx, tenantUUID)
			if err != nil {
				cli.Logger.Fatalf("failed to get step run queue metrics: %v", err)
			}
			if resp.JSON200 == nil {
				cli.Logger.Fatalf("unexpected response from API (status %d)", resp.StatusCode())
			}
			printJSON(resp.JSON200)
			return
		}

		params := &rest.TenantGetQueueMetricsParams{}
		if cmd.Flags().Changed("workflow-ids") {
			workflowIDs, _ := cmd.Flags().GetStringSlice("workflow-ids")
			workflows := make([]rest.WorkflowID, len(workflowIDs))
			copy(workflows, workflowIDs)
			params.Workflows = &workflows
		}
		if cmd.Flags().Changed("additional-metadata") {
			additionalMetadata, _ := cmd.Flags().GetStringSlice("additional-metadata")
			params.AdditionalMetadata = &additionalMetadata
		}

		resp, err := hatchetClient.API().TenantGetQueueMetricsWithResponse(ctx, tenantUUID, params)
		if err != nil {
			cli.Logger.Fatalf("failed to get queue metrics: %v", err)
		}
		if resp.JSON200 == nil {
			cli.Logger.Fatalf("unexpected response from API (status %d)", resp.StatusCode())
		}

		printJSON(resp.JSON200)
	},
}

var tenantTaskStatsCmd = &cobra.Command{
	Use:   "task-stats",
	Short: "Get tenant task stats",
	Long:  `Get task statistics for the current tenant. Outputs raw JSON.`,
	Run: func(cmd *cobra.Command, args []string) {
		_, hatchetClient := clientFromCmd(cmd)
		tenantUUID := clientTenantUUID(hatchetClient)

		params := &rest.TenantGetTaskStatsParams{}
		if cmd.Flags().Changed("task-names") {
			taskNames, _ := cmd.Flags().GetStringSlice("task-names")
			params.TaskNames = &taskNames
		}

		resp, err := hatchetClient.API().TenantGetTaskStatsWithResponse(cmd.Context(), tenantUUID, params)
		if err != nil {
			cli.Logger.Fatalf("failed to get task stats: %v", err)
		}
		if resp.JSON200 == nil {
			cli.Logger.Fatalf("unexpected response from API (status %d)", resp.StatusCode())
		}

		printJSON(resp.JSON200)
	},
}

var tenantPrometheusMetricsCmd = &cobra.Command{
	Use:   "prometheus-metrics",
	Short: "Get tenant Prometheus metrics",
	Long:  `Get Prometheus metrics for the current tenant in the Prometheus text exposition format.`,
	Run: func(cmd *cobra.Command, args []string) {
		_, hatchetClient := clientFromCmd(cmd)
		tenantUUID := clientTenantUUID(hatchetClient)

		resp, err := hatchetClient.API().TenantGetPrometheusMetricsWithResponse(cmd.Context(), tenantUUID)
		if err != nil {
			cli.Logger.Fatalf("failed to get prometheus metrics: %v", err)
		}
		if resp.StatusCode() != 200 {
			cli.Logger.Fatalf("unexpected response from API (status %d)", resp.StatusCode())
		}

		fmt.Print(string(resp.Body))
	},
}

var tenantFeatureFlagCmd = &cobra.Command{
	Use:   "feature-flag <key>",
	Short: "Evaluate a feature flag",
	Long:  `Evaluate a feature flag for the current tenant. Outputs raw JSON.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		_, hatchetClient := clientFromCmd(cmd)
		tenantUUID := clientTenantUUID(hatchetClient)

		enabledIfNoPosthog, _ := cmd.Flags().GetBool("enabled-if-no-posthog")
		params := &rest.TenantFeatureFlagEvaluateParams{
			FeatureFlagId:        rest.FeatureFlagId(args[0]),
			IsEnabledIfNoPosthog: enabledIfNoPosthog,
		}

		resp, err := hatchetClient.API().TenantFeatureFlagEvaluateWithResponse(cmd.Context(), tenantUUID, params)
		if err != nil {
			cli.Logger.Fatalf("failed to evaluate feature flag: %v", err)
		}
		if resp.JSON200 == nil {
			cli.Logger.Fatalf("unexpected response from API (status %d)", resp.StatusCode())
		}

		printJSON(resp.JSON200)
	},
}

var tenantMembersCmd = &cobra.Command{
	Use:     "members",
	Aliases: []string{"member"},
	Short:   "Manage tenant members",
	Long:    `Commands for listing, updating, and removing tenant members.`,
	Run:     func(cmd *cobra.Command, args []string) { _ = cmd.Help() },
}

var tenantMembersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tenant members",
	Long:  `List members of the current tenant.`,
	Run: func(cmd *cobra.Command, args []string) {
		_, hatchetClient := clientFromCmd(cmd)
		tenantUUID := clientTenantUUID(hatchetClient)

		resp, err := hatchetClient.API().TenantMemberListWithResponse(cmd.Context(), tenantUUID)
		if err != nil {
			cli.Logger.Fatalf("failed to list tenant members: %v", err)
		}
		if resp.JSON200 == nil {
			cli.Logger.Fatalf("unexpected response from API (status %d)", resp.StatusCode())
		}

		if isJSONOutput(cmd) {
			printJSON(resp.JSON200)
			return
		}

		if resp.JSON200.Rows == nil || len(*resp.JSON200.Rows) == 0 {
			fmt.Println("No members found.")
			return
		}

		fmt.Printf("%-40s  %-8s  %s\n", "EMAIL", "ROLE", "ID")
		for _, member := range *resp.JSON200.Rows {
			fmt.Printf("%-40s  %-8s  %s\n", member.User.Email, member.Role, member.Metadata.Id)
		}
	},
}

var tenantMembersUpdateCmd = &cobra.Command{
	Use:   "update <member-id>",
	Short: "Update a tenant member's role",
	Long:  `Update the role of a tenant member by member ID.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		_, hatchetClient := clientFromCmd(cmd)
		tenantUUID := clientTenantUUID(hatchetClient)
		memberUUID := parseUUIDArg(args[0], "member id")

		roleStr, _ := cmd.Flags().GetString("role")
		if roleStr == "" {
			cli.Logger.Fatal("--role is required")
		}
		role, err := parseTenantMemberRole(roleStr)
		if err != nil {
			cli.Logger.Fatalf("%v", err)
		}

		resp, err := hatchetClient.API().TenantMemberUpdateWithResponse(cmd.Context(), tenantUUID, memberUUID, rest.TenantMemberUpdateJSONRequestBody{
			Role: role,
		})
		if err != nil {
			cli.Logger.Fatalf("failed to update tenant member: %v", err)
		}
		if resp.JSON200 == nil {
			cli.Logger.Fatalf("unexpected response from API (status %d)", resp.StatusCode())
		}

		if isJSONOutput(cmd) {
			printJSON(resp.JSON200)
		} else {
			fmt.Println(styles.SuccessMessage(fmt.Sprintf("Updated member %s to role %s", resp.JSON200.User.Email, resp.JSON200.Role)))
		}
	},
}

var tenantMembersDeleteCmd = &cobra.Command{
	Use:   "delete <member-id>",
	Short: "Remove a tenant member",
	Long:  `Remove a member from the current tenant by member ID. Use --yes to skip confirmation.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		isJSON := isJSONOutput(cmd)
		yes, _ := cmd.Flags().GetBool("yes")
		_, hatchetClient := clientFromCmd(cmd)
		tenantUUID := clientTenantUUID(hatchetClient)
		memberUUID := parseUUIDArg(args[0], "member id")

		if !isJSON && !yes {
			if !confirmAction(fmt.Sprintf("Remove member '%s' from the tenant?", args[0])) {
				fmt.Println("Aborted.")
				return
			}
		}

		resp, err := hatchetClient.API().TenantMemberDeleteWithResponse(cmd.Context(), tenantUUID, memberUUID)
		if err != nil {
			cli.Logger.Fatalf("failed to remove tenant member: %v", err)
		}
		if resp.StatusCode() >= 400 {
			cli.Logger.Fatalf("failed to remove tenant member (status %d)", resp.StatusCode())
		}

		if isJSON {
			printJSON(map[string]interface{}{"deleted": true, "memberId": args[0]})
		} else {
			fmt.Println(styles.SuccessMessage(fmt.Sprintf("Removed member: %s", args[0])))
		}
	},
}

var tenantInvitesCmd = &cobra.Command{
	Use:     "invites",
	Aliases: []string{"invite"},
	Short:   "Manage tenant invites",
	Long:    `Commands for listing, creating, updating, and deleting tenant invites.`,
	Run:     func(cmd *cobra.Command, args []string) { _ = cmd.Help() },
}

var tenantInvitesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tenant invites",
	Long:  `List pending invites for the current tenant.`,
	Run: func(cmd *cobra.Command, args []string) {
		_, hatchetClient := clientFromCmd(cmd)
		tenantUUID := clientTenantUUID(hatchetClient)

		resp, err := hatchetClient.API().TenantInviteListWithResponse(cmd.Context(), tenantUUID)
		if err != nil {
			cli.Logger.Fatalf("failed to list tenant invites: %v", err)
		}
		if resp.JSON200 == nil {
			cli.Logger.Fatalf("unexpected response from API (status %d)", resp.StatusCode())
		}

		if isJSONOutput(cmd) {
			printJSON(resp.JSON200)
			return
		}

		if resp.JSON200.Rows == nil || len(*resp.JSON200.Rows) == 0 {
			fmt.Println("No invites found.")
			return
		}

		fmt.Printf("%-40s  %-8s  %-36s  %s\n", "EMAIL", "ROLE", "ID", "EXPIRES")
		for _, invite := range *resp.JSON200.Rows {
			fmt.Printf("%-40s  %-8s  %-36s  %s\n", invite.Email, invite.Role, invite.Metadata.Id, invite.Expires.Format("2006-01-02 15:04:05"))
		}
	},
}

var tenantInvitesCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a tenant invite",
	Long:  `Invite a user to the current tenant by email. In --output json mode, --email and --role are required. Otherwise, launches an interactive form for missing values.`,
	Example: `  # Interactive mode
  hatchet tenant invites create

  # Non-interactive
  hatchet tenant invites create --email user@example.com --role MEMBER -o json`,
	Run: func(cmd *cobra.Command, args []string) {
		isJSON := isJSONOutput(cmd)
		_, hatchetClient := clientFromCmd(cmd)
		tenantUUID := clientTenantUUID(hatchetClient)

		email, _ := cmd.Flags().GetString("email")
		roleStr, _ := cmd.Flags().GetString("role")

		if !isJSON && (email == "" || roleStr == "") {
			if roleStr == "" {
				roleStr = string(rest.MEMBER)
			}
			form := huh.NewForm(
				huh.NewGroup(
					huh.NewInput().
						Title("Email").
						Value(&email).
						Placeholder("user@example.com").
						Validate(func(s string) error {
							if !strings.Contains(s, "@") {
								return fmt.Errorf("must be a valid email")
							}
							return nil
						}),
				),
				huh.NewGroup(
					huh.NewSelect[string]().
						Title("Role").
						Options(tenantMemberRoleOptions()...).
						Value(&roleStr),
				),
			).WithTheme(styles.HatchetTheme())
			if err := form.Run(); err != nil {
				cli.Logger.Fatalf("form cancelled: %v", err)
			}
		}

		if email == "" {
			cli.Logger.Fatal("--email is required")
		}
		if roleStr == "" {
			cli.Logger.Fatal("--role is required")
		}
		role, err := parseTenantMemberRole(roleStr)
		if err != nil {
			cli.Logger.Fatalf("%v", err)
		}

		resp, err := hatchetClient.API().TenantInviteCreateWithResponse(cmd.Context(), tenantUUID, rest.TenantInviteCreateJSONRequestBody{
			Email: email,
			Role:  role,
		})
		if err != nil {
			cli.Logger.Fatalf("failed to create tenant invite: %v", err)
		}
		if resp.JSON201 == nil {
			cli.Logger.Fatalf("unexpected response from API (status %d)", resp.StatusCode())
		}

		if isJSON {
			printJSON(resp.JSON201)
		} else {
			fmt.Println(styles.SuccessMessage(fmt.Sprintf("Invited %s as %s", resp.JSON201.Email, resp.JSON201.Role)))
		}
	},
}

var tenantInvitesUpdateCmd = &cobra.Command{
	Use:   "update <invite-id>",
	Short: "Update a tenant invite's role",
	Long:  `Update the role of a pending tenant invite by invite ID.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		_, hatchetClient := clientFromCmd(cmd)
		tenantUUID := clientTenantUUID(hatchetClient)
		inviteUUID := parseUUIDArg(args[0], "invite id")

		roleStr, _ := cmd.Flags().GetString("role")
		if roleStr == "" {
			cli.Logger.Fatal("--role is required")
		}
		role, err := parseTenantMemberRole(roleStr)
		if err != nil {
			cli.Logger.Fatalf("%v", err)
		}

		resp, err := hatchetClient.API().TenantInviteUpdateWithResponse(cmd.Context(), tenantUUID, inviteUUID, rest.TenantInviteUpdateJSONRequestBody{
			Role: role,
		})
		if err != nil {
			cli.Logger.Fatalf("failed to update tenant invite: %v", err)
		}
		if resp.JSON200 == nil {
			cli.Logger.Fatalf("unexpected response from API (status %d)", resp.StatusCode())
		}

		if isJSONOutput(cmd) {
			printJSON(resp.JSON200)
		} else {
			fmt.Println(styles.SuccessMessage(fmt.Sprintf("Updated invite for %s to role %s", resp.JSON200.Email, resp.JSON200.Role)))
		}
	},
}

var tenantInvitesDeleteCmd = &cobra.Command{
	Use:   "delete <invite-id>",
	Short: "Delete a tenant invite",
	Long:  `Delete a pending tenant invite by invite ID. Use --yes to skip confirmation.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		isJSON := isJSONOutput(cmd)
		yes, _ := cmd.Flags().GetBool("yes")
		_, hatchetClient := clientFromCmd(cmd)
		tenantUUID := clientTenantUUID(hatchetClient)
		inviteUUID := parseUUIDArg(args[0], "invite id")

		if !isJSON && !yes {
			if !confirmAction(fmt.Sprintf("Delete invite '%s'?", args[0])) {
				fmt.Println("Aborted.")
				return
			}
		}

		resp, err := hatchetClient.API().TenantInviteDeleteWithResponse(cmd.Context(), tenantUUID, inviteUUID)
		if err != nil {
			cli.Logger.Fatalf("failed to delete tenant invite: %v", err)
		}
		if resp.StatusCode() >= 400 {
			cli.Logger.Fatalf("failed to delete tenant invite (status %d)", resp.StatusCode())
		}

		if isJSON {
			printJSON(map[string]interface{}{"deleted": true, "inviteId": args[0]})
		} else {
			fmt.Println(styles.SuccessMessage(fmt.Sprintf("Deleted invite: %s", args[0])))
		}
	},
}

func init() {
	rootCmd.AddCommand(tenantCmd)
	tenantCmd.AddCommand(
		tenantGetCmd,
		tenantUpdateCmd,
		tenantResourcePolicyCmd,
		tenantQueueMetricsCmd,
		tenantTaskStatsCmd,
		tenantPrometheusMetricsCmd,
		tenantFeatureFlagCmd,
		tenantMembersCmd,
		tenantInvitesCmd,
	)
	tenantMembersCmd.AddCommand(tenantMembersListCmd, tenantMembersUpdateCmd, tenantMembersDeleteCmd)
	tenantInvitesCmd.AddCommand(tenantInvitesListCmd, tenantInvitesCreateCmd, tenantInvitesUpdateCmd, tenantInvitesDeleteCmd)

	tenantCmd.PersistentFlags().StringP("profile", "p", "", "Profile to use for connecting to Hatchet (default: prompts for selection)")
	tenantCmd.PersistentFlags().StringP("output", "o", "", "Output format: json")

	tenantUpdateCmd.Flags().String("name", "", "Name of the tenant")
	tenantUpdateCmd.Flags().Bool("analytics-opt-out", false, "Opt out of analytics")
	tenantUpdateCmd.Flags().Bool("alert-member-emails", false, "Alert tenant members via email")
	tenantUpdateCmd.Flags().Bool("enable-expiring-token-alerts", false, "Enable alerts when tokens are approaching expiration")
	tenantUpdateCmd.Flags().Bool("enable-resource-limit-alerts", false, "Enable alerts when tenant resources are approaching limits")
	tenantUpdateCmd.Flags().Bool("enable-run-failure-alerts", false, "Enable alerts when workflow runs fail")
	tenantUpdateCmd.Flags().String("max-alerting-frequency", "", "Max frequency at which to alert (e.g. 1h)")
	tenantUpdateCmd.Flags().String("version", "", "Tenant engine version: V0 or V1")

	tenantQueueMetricsCmd.Flags().Bool("step-runs", false, "Get step run queue metrics instead")
	tenantQueueMetricsCmd.Flags().StringSlice("workflow-ids", nil, "Workflow IDs to filter by")
	tenantQueueMetricsCmd.Flags().StringSlice("additional-metadata", nil, "Metadata key:value pairs to filter by")

	tenantTaskStatsCmd.Flags().StringSlice("task-names", nil, "Task names that must appear in the response")

	tenantFeatureFlagCmd.Flags().Bool("enabled-if-no-posthog", false, "Treat the flag as enabled if PostHog is disabled or unavailable")

	tenantMembersUpdateCmd.Flags().String("role", "", "New role: OWNER, ADMIN, or MEMBER")
	tenantMembersDeleteCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")

	tenantInvitesCreateCmd.Flags().String("email", "", "Email of the user to invite")
	tenantInvitesCreateCmd.Flags().String("role", "", "Role for the invited user: OWNER, ADMIN, or MEMBER")
	tenantInvitesUpdateCmd.Flags().String("role", "", "New role: OWNER, ADMIN, or MEMBER")
	tenantInvitesDeleteCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")
}
