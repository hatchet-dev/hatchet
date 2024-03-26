package cli

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/internal/config/loader"
)

var (
	tokenTenantId string
	tokenName     string
)

var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "command for generating tokens.",
}

var tokenCreateAPICmd = &cobra.Command{
	Use:   "create",
	Short: "create a new API token.",
	Run: func(cmd *cobra.Command, args []string) {
		err := runCreateAPIToken()

		if err != nil {
			log.Printf("Fatal: could not run [token create] command: %v", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(tokenCmd)
	tokenCmd.AddCommand(tokenCreateAPICmd)

	tokenCreateAPICmd.PersistentFlags().StringVar(
		&tokenTenantId,
		"tenant-id",
		"",
		"the tenant ID to associate with the token",
	)

	// require the tenant ID
	tokenCreateAPICmd.MarkPersistentFlagRequired("tenant-id") // nolint: errcheck

	tokenCreateAPICmd.PersistentFlags().StringVar(
		&tokenName,
		"name",
		"default",
		"the name of the token",
	)
}

func runCreateAPIToken() error {
	// read in the local config
	configLoader := loader.NewConfigLoader(configDirectory)

	cleanup, serverConf, err := configLoader.LoadServerConfig()
	if err != nil {
		return err
	}

	defer cleanup() // nolint:errcheck

	defer serverConf.Disconnect() // nolint:errcheck

	defaultTok, err := serverConf.Auth.JWTManager.GenerateTenantToken(tokenTenantId, tokenName)

	if err != nil {
		return err
	}

	fmt.Println(defaultTok)

	return nil
}
