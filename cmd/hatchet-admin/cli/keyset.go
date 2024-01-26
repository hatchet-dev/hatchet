package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/internal/encryption"
)

var (
	cloudKMSCredentialsPath string
	cloudKMSKeyURI          string
)

var keysetCmd = &cobra.Command{
	Use:   "keyset",
	Short: "command for managing keysets.",
}

var keysetCreateMasterCmd = &cobra.Command{
	Use:   "create-master",
	Short: "create a new local master keyset.",
	Run: func(cmd *cobra.Command, args []string) {
		err := runCreateLocalMasterKeyset()

		if err != nil {
			fmt.Printf("Fatal: could not run seed command: %v\n", err)
			os.Exit(1)
		}
	},
}

var keysetCreateLocalJWTCmd = &cobra.Command{
	Use:   "create-local-jwt",
	Short: "create a new local JWT keyset.",
	Run: func(cmd *cobra.Command, args []string) {
		err := runCreateLocalJWTKeyset()

		if err != nil {
			fmt.Printf("Fatal: could not run seed command: %v\n", err)
			os.Exit(1)
		}
	},
}

var keysetCreateCloudKMSJWTCmd = &cobra.Command{
	Use:   "create-cloudkms-jwt",
	Short: "create a new JWT keyset encrypted by a remote CloudKMS repository.",
	Run: func(cmd *cobra.Command, args []string) {
		err := runCreateCloudKMSJWTKeyset()

		if err != nil {
			fmt.Printf("Fatal: could not run seed command: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(keysetCmd)
	keysetCmd.AddCommand(keysetCreateMasterCmd)
	keysetCmd.AddCommand(keysetCreateLocalJWTCmd)
	keysetCmd.AddCommand(keysetCreateCloudKMSJWTCmd)

	keysetCreateCloudKMSJWTCmd.PersistentFlags().StringVar(
		&cloudKMSCredentialsPath,
		"credentials",
		"",
		"path to the JSON credentials file for the CloudKMS repository",
	)

	keysetCreateCloudKMSJWTCmd.PersistentFlags().StringVar(
		&cloudKMSKeyURI,
		"key-uri",
		"",
		"URI of the key in the CloudKMS repository",
	)
}

func runCreateLocalMasterKeyset() error {
	masterKeyBytes, _, _, err := encryption.GenerateLocalKeys()

	if err != nil {
		return err
	}

	fmt.Println("Master Key Bytes:")
	fmt.Println(string(masterKeyBytes))

	return nil
}

func runCreateLocalJWTKeyset() error {
	_, privateEc256, publicEc256, err := encryption.GenerateLocalKeys()

	if err != nil {
		return err
	}

	fmt.Println("Private EC256 Keyset:")
	fmt.Println(string(privateEc256))

	fmt.Println("Public EC256 Keyset:")
	fmt.Println(string(publicEc256))

	return nil
}

func runCreateCloudKMSJWTKeyset() error {
	if cloudKMSCredentialsPath == "" {
		return fmt.Errorf("missing required flag --credentials")
	}

	if cloudKMSKeyURI == "" {
		return fmt.Errorf("missing required flag --key-uri")
	}

	credentials, err := os.ReadFile(cloudKMSCredentialsPath)

	if err != nil {
		return err
	}

	privateEc256, publicEc256, err := encryption.GenerateJWTKeysetsFromCloudKMS(cloudKMSKeyURI, credentials)

	if err != nil {
		return err
	}

	fmt.Println("Private EC256 Keyset:")
	fmt.Println(string(privateEc256))

	fmt.Println("Public EC256 Keyset:")
	fmt.Println(string(publicEc256))

	return nil
}
