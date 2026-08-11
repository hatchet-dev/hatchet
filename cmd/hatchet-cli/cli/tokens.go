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

var tokensCmd = &cobra.Command{
	Use:     "tokens",
	Aliases: []string{"token", "api-tokens"},
	Short:   "Manage API tokens",
	Long:    `Commands for listing, creating, and revoking API tokens.`,
	Run:     func(cmd *cobra.Command, args []string) { _ = cmd.Help() },
}

var tokensListCmd = &cobra.Command{
	Use:   "list",
	Short: "List API tokens",
	Long:  `List API tokens for the current tenant.`,
	Example: `  # List tokens
  hatchet tokens list --profile local

  # JSON output
  hatchet tokens list -o json`,
	Run: func(cmd *cobra.Command, args []string) {
		isJSON := isJSONOutput(cmd)
		_, hatchetClient := clientFromCmd(cmd)
		ctx := cmd.Context()
		tenantUUID := clientTenantUUID(hatchetClient)

		resp, err := hatchetClient.API().ApiTokenListWithResponse(ctx, tenantUUID)
		if err != nil {
			cli.Logger.Fatalf("failed to list API tokens: %v", err)
		}
		if resp.JSON200 == nil {
			cli.Logger.Fatalf("unexpected response from API (status %d)", resp.StatusCode())
		}

		if isJSON {
			printJSON(resp.JSON200)
			return
		}

		if resp.JSON200.Rows == nil || len(*resp.JSON200.Rows) == 0 {
			fmt.Println("No API tokens found.")
			return
		}
		for _, token := range *resp.JSON200.Rows {
			fmt.Printf("%s\t%s\texpires %s\n", token.Name, token.Metadata.Id, token.ExpiresAt.Format("2006-01-02 15:04:05 MST"))
		}
	},
}

var tokensCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an API token",
	Long:  `Create a new API token. The token value is only shown once, so store it securely.`,
	Example: `  # Interactive mode
  hatchet tokens create --profile local

  # Non-interactive
  hatchet tokens create --name my-token --expires-in 720h -o json`,
	Run: func(cmd *cobra.Command, args []string) {
		isJSON := isJSONOutput(cmd)
		_, hatchetClient := clientFromCmd(cmd)

		name, _ := cmd.Flags().GetString("name")
		expiresIn, _ := cmd.Flags().GetString("expires-in")

		if !isJSON && name == "" {
			form := huh.NewForm(huh.NewGroup(
				huh.NewInput().
					Title("Token name").
					Value(&name).
					Placeholder("my-token").
					Validate(func(s string) error {
						if strings.TrimSpace(s) == "" {
							return fmt.Errorf("name is required")
						}
						return nil
					}),
			)).WithTheme(styles.HatchetTheme())
			if err := form.Run(); err != nil {
				cli.Logger.Fatalf("form cancelled: %v", err)
			}
		}

		if name == "" {
			cli.Logger.Fatal("--name is required")
		}

		body := rest.CreateAPITokenRequest{
			Name: name,
		}
		if expiresIn != "" {
			body.ExpiresIn = &expiresIn
		}

		ctx := cmd.Context()
		tenantUUID := clientTenantUUID(hatchetClient)
		resp, err := hatchetClient.API().ApiTokenCreateWithResponse(ctx, tenantUUID, body)
		if err != nil {
			cli.Logger.Fatalf("failed to create API token: %v", err)
		}
		if resp.JSON200 == nil {
			cli.Logger.Fatalf("unexpected response from API (status %d)", resp.StatusCode())
		}

		if isJSON {
			printJSON(resp.JSON200)
			return
		}

		fmt.Println(styles.SuccessMessage(fmt.Sprintf("Created API token: %s", name)))
		fmt.Println("This token is only shown once. Store it securely:")
		fmt.Println(resp.JSON200.Token)
	},
}

var tokensRevokeCmd = &cobra.Command{
	Use:   "revoke <token-id>",
	Short: "Revoke an API token",
	Long:  `Revoke an API token by ID. Use --yes to skip confirmation.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		isJSON := isJSONOutput(cmd)
		yes, _ := cmd.Flags().GetBool("yes")
		_, hatchetClient := clientFromCmd(cmd)

		tokenUUID, err := uuid.Parse(args[0])
		if err != nil {
			cli.Logger.Fatalf("invalid token ID %q: %v", args[0], err)
		}

		if !isJSON && !yes {
			if !confirmAction(fmt.Sprintf("Revoke API token '%s'?", args[0])) {
				fmt.Println("Aborted.")
				return
			}
		}

		resp, err := hatchetClient.API().ApiTokenUpdateRevokeWithResponse(cmd.Context(), tokenUUID)
		if err != nil {
			cli.Logger.Fatalf("failed to revoke API token: %v", err)
		}
		if resp.StatusCode() >= 400 {
			cli.Logger.Fatalf("failed to revoke API token (status %d)", resp.StatusCode())
		}

		if isJSON {
			printJSON(map[string]interface{}{"revoked": true})
		} else {
			fmt.Println(styles.SuccessMessage(fmt.Sprintf("Revoked API token: %s", args[0])))
		}
	},
}

func init() {
	rootCmd.AddCommand(tokensCmd)
	tokensCmd.AddCommand(tokensListCmd, tokensCreateCmd, tokensRevokeCmd)

	tokensCmd.PersistentFlags().StringP("profile", "p", "", "Profile to use for connecting to Hatchet (default: prompts for selection)")
	tokensCmd.PersistentFlags().StringP("output", "o", "", "Output format: json")

	tokensCreateCmd.Flags().StringP("name", "n", "", "Name for the API token")
	tokensCreateCmd.Flags().String("expires-in", "", "Duration for which the token is valid (e.g. 720h)")

	tokensRevokeCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")
}
