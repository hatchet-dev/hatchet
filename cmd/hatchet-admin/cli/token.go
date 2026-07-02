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

	"github.com/hatchet-dev/hatchet/pkg/auth/token"
	"github.com/hatchet-dev/hatchet/pkg/config/loader"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

var (
	tokenTenantIdStr string
	tokenName        string
	expiresIn        time.Duration
	tokenNoAuth      bool
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

	tokenCreateAPICmd.PersistentFlags().BoolVar(
		&tokenNoAuth,
		"no-auth",
		false,
		"sign the token with the no-auth keyset for the default tenant (local no-auth mode)",
	)
}

func runCreateAPIToken(expiresIn time.Duration) error {
	// read in the local config
	configLoader := loader.NewConfigLoader(configDirectory)

	var cf *server.ServerConfigFile

	cleanup, srv, err := configLoader.CreateServerFromConfig("", func(scf *server.ServerConfigFile) {
		// disable rabbitmq since it's not needed to create the api token
		scf.MessageQueue.Enabled = false

		// disable security checks since we're not running the server
		scf.SecurityCheck.Enabled = false

		cf = scf
	})

	if err != nil {
		return err
	}

	defer cleanup() // nolint:errcheck

	defer srv.Disconnect() // nolint:errcheck

	expiresAt := time.Now().UTC().Add(expiresIn)

	jwtManager := srv.Auth.JWTManager

	tenantId, err := tenantIDForTokenCreate(srv.Seed.DefaultTenantID)
	if err != nil {
		return err
	}

	if tokenNoAuth {
		// no-auth tokens are always scoped to the seed default tenant and signed by the no-auth keyset
		tenantId, err = uuid.Parse(srv.Seed.DefaultTenantID)
		if err != nil {
			return err
		}

		noAuthEncryptionSvc, encErr := loader.LoadNoAuthEncryptionSvc(cf)
		if encErr != nil {
			return encErr
		}

		jwtManager, err = token.NewJWTManager(noAuthEncryptionSvc, srv.V1.APIToken(), &token.TokenOpts{
			Issuer:               cf.Runtime.ServerURL,
			Audience:             cf.Runtime.ServerURL,
			GRPCBroadcastAddress: cf.Runtime.GRPCBroadcastAddress,
			ServerURL:            cf.Runtime.ServerURL,
		})
		if err != nil {
			return err
		}
	}

	defaultTok, err := jwtManager.GenerateTenantToken(context.Background(), tenantId, tokenName, false, &expiresAt)

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
