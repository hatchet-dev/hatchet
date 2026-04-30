package cli

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/pkg/config/loader"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

var (
	tokenTenantIdStr string
	tokenName        string
	expiresIn        time.Duration
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
		&tokenTenantIdStr,
		"tenant-id",
		"",
		"the tenant ID to associate with the token",
	)

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

	tenantId, err := tenantIDForTokenCreate(server.Seed.DefaultTenantID)
	if err != nil {
		return err
	}

	defaultTok, err := server.Auth.JWTManager.GenerateTenantToken(context.Background(), tenantId, tokenName, false, &expiresAt)

	if err != nil {
		return err
	}

	fmt.Println(defaultTok.Token)

	return nil
}

// tenantIDForTokenCreate returns the tenant UUID from --tenant-id when set, otherwise the seed default.
func tenantIDForTokenCreate(defaultTenantID string) (uuid.UUID, error) {
	if strings.TrimSpace(tokenTenantIdStr) != "" {
		id, err := uuid.Parse(strings.TrimSpace(tokenTenantIdStr))
		if err != nil {
			return uuid.Nil, fmt.Errorf("parse --tenant-id: %w", err)
		}
		return id, nil
	}
	return uuid.Parse(defaultTenantID)
}
