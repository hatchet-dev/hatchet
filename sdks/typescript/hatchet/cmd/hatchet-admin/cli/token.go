package cli

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/pkg/config/loader"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

var (
	tokenTenantId string
	tokenName     string
	expiresIn     time.Duration
)

var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "command for generating tokens.",
}

var tokenCreateAPICmd = &cobra.Command{
	Use:   "create",
	Short: "create a new API token.",
	Run: func(cmd *cobra.Command, args []string) {
		err := runCreateAPIToken(expiresIn)

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

	tokenCreateAPICmd.PersistentFlags().DurationVarP(
		&expiresIn,
		"expiresIn",
		"e",
		90*24*time.Hour,
		"Expiration duration for the API token",
	)

}

func runCreateAPIToken(expiresIn time.Duration) error {
	// read in the local config
	configLoader := loader.NewConfigLoader(configDirectory)

	cleanup, server, err := configLoader.CreateServerFromConfig("", func(scf *server.ServerConfigFile) {
		// disable rabbitmq since it's not needed to create the api token
		scf.MessageQueue.Enabled = false

		// disable security checks since we're not running the server
		scf.SecurityCheck.Enabled = false
	})

	if err != nil {
		return err
	}

	defer cleanup() // nolint:errcheck

	defer server.Disconnect() // nolint:errcheck

	expiresAt := time.Now().UTC().Add(expiresIn)

	tenantId := tokenTenantId

	if tenantId == "" {
		tenantId = server.Seed.DefaultTenantID
	}

	defaultTok, err := server.Auth.JWTManager.GenerateTenantToken(context.Background(), tenantId, tokenName, false, &expiresAt)

	if err != nil {
		return err
	}

	fmt.Println(defaultTok.Token)

	return nil
}
