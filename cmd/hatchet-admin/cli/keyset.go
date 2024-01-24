package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/internal/encryption"
)

// keysetCmd seeds the database with initial data
var keysetCmd = &cobra.Command{
	Use:   "keyset",
	Short: "command for managing keysets.",
}

var keysetCreateLocalCmd = &cobra.Command{
	Use:   "create-local",
	Short: "create a new local keyset.",
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		err = runCreateLocalKeyset()

		if err != nil {
			fmt.Printf("Fatal: could not run seed command: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(keysetCmd)
	keysetCmd.AddCommand(keysetCreateLocalCmd)
}

func runCreateLocalKeyset() error {
	masterKeyBytes, privateEc256, publicEc256, err := encryption.GenerateLocalKeys()

	if err != nil {
		return err
	}

	fmt.Println("Master Key Bytes:")
	fmt.Println(string(masterKeyBytes))

	fmt.Println("Private EC256 Keyset:")
	fmt.Println(string(privateEc256))

	fmt.Println("Public EC256 Keyset:")
	fmt.Println(string(publicEc256))

	return nil
}
