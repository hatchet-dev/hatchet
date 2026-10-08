package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/pkg/config/loader"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

var (
	userEmail         string
	userPasswordStdin bool
)

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "command for managing users.",
}

var userSetPasswordCmd = &cobra.Command{
	Use:   "set-password",
	Short: "set-password replaces a user's password with one read from stdin, hashed by this build.",
	Run: func(cmd *cobra.Command, args []string) {
		if !userPasswordStdin {
			log.Printf("Fatal: pass --password-stdin and provide the new password on stdin")
			os.Exit(1)
		}

		if err := runSetPassword(loader.NewConfigLoader(configDirectory), userEmail, os.Stdin); err != nil {
			log.Printf("Fatal: could not set password: %v", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(userCmd)
	userCmd.AddCommand(userSetPasswordCmd)

	userSetPasswordCmd.PersistentFlags().StringVar(&userEmail, "email", "", "email of the user")
	userSetPasswordCmd.PersistentFlags().BoolVar(&userPasswordStdin, "password-stdin", false, "read the new password from stdin")

	_ = userSetPasswordCmd.MarkPersistentFlagRequired("email")
}

func runSetPassword(cf *loader.ConfigLoader, email string, in io.Reader) error {
	raw, err := bufio.NewReader(in).ReadString('\n')

	if err != nil && err != io.EOF {
		return fmt.Errorf("could not read password from stdin: %w", err)
	}

	password := strings.TrimRight(raw, "\r\n")

	if password == "" {
		return fmt.Errorf("password on stdin is empty")
	}

	dc, err := cf.InitDataLayer()

	if err != nil {
		return err
	}

	defer dc.Disconnect() // nolint: errcheck

	ctx := context.Background()

	user, err := dc.V1.User().GetUserByEmail(ctx, email)

	if err != nil {
		return fmt.Errorf("could not find user %s: %w", email, err)
	}

	hash, err := v1.HashPassword(password)

	if err != nil {
		return err
	}

	if _, err := dc.V1.User().UpdateUser(ctx, user.ID, &v1.UpdateUserOpts{Password: hash}); err != nil {
		return fmt.Errorf("could not update user %s: %w", email, err)
	}

	sessions, err := dc.V1.UserSession().DeleteByUserId(ctx, user.ID, nil)

	if err != nil {
		return fmt.Errorf("password updated but could not revoke sessions for %s: %w", email, err)
	}

	fmt.Printf("password updated for %s, %d session(s) revoked\n", email, len(sessions))

	return nil
}
