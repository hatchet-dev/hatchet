package cli

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/pkg/encryption"
)

var (
	encryptionKeyDir        string
	cloudKMSCredentialsPath string
	cloudKMSKeyURI          string
	keysetWithNoAuth        bool
)

var keysetCmd = &cobra.Command{
	Use:   "keyset",
	Short: "command for managing keysets.",
}

var keysetCreateLocalKeysetsCmd = &cobra.Command{
	Use:   "create-local-keys",
	Short: "create a new local master keyset and JWT public/private keyset.",
	Run: func(cmd *cobra.Command, args []string) {
		err := runCreateLocalKeysets()

		if err != nil {
			log.Printf("Fatal: could not run [keyset create-local-keys] command: %v", err)
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
			log.Printf("Fatal: could not run [keyset create-cloudkms-jwt] command: %v", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(keysetCmd)
	keysetCmd.AddCommand(keysetCreateLocalKeysetsCmd)
	keysetCmd.AddCommand(keysetCreateCloudKMSJWTCmd)

	keysetCmd.PersistentFlags().StringVar(
		&encryptionKeyDir,
		"key-dir",
		"",
		"if storing keys on disk, path to the directory where encryption keys should be stored",
	)

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

	keysetCreateLocalKeysetsCmd.PersistentFlags().BoolVar(
		&keysetWithNoAuth,
		"no-auth",
		false,
		"also generate a dedicated JWT keyset for local no-auth mode",
	)
}

func runCreateLocalKeysets() error {
	masterKeyBytes, privateEc256, publicEc256, insecurePublicHandleEc256, err := encryption.GenerateLocalKeys()

	if err != nil {
		return err
	}

	if encryptionKeyDir != "" {
		// we write these as .key files so that they're gitignored by default
		err = os.WriteFile(encryptionKeyDir+"/master.key", masterKeyBytes, 0600)

		if err != nil {
			return err
		}

		err = os.WriteFile(encryptionKeyDir+"/private_ec256.key", privateEc256, 0600)

		if err != nil {
			return err
		}

		err = os.WriteFile(encryptionKeyDir+"/public_ec256.key", publicEc256, 0600)

		if err != nil {
			return err
		}

		err = os.WriteFile(encryptionKeyDir+"/public_handle_unencrypted_ec256.key", insecurePublicHandleEc256, 0600)

		if err != nil {
			return err
		}

		if keysetWithNoAuth {
			if err := writeNoAuthKeyset(masterKeyBytes); err != nil {
				return err
			}
		}
	} else {
		fmt.Println("Master Key Bytes:")
		fmt.Println(string(masterKeyBytes))

		fmt.Println("Private EC256 Keyset:")
		fmt.Println(string(privateEc256))

		fmt.Println("Public EC256 Keyset:")
		fmt.Println(string(publicEc256))

		fmt.Println("Insecure Public Handle EC256 Keyset:")
		fmt.Println(string(insecurePublicHandleEc256))

		if keysetWithNoAuth {
			noAuthPrivate, noAuthPublic, _, genErr := encryption.GenerateJWTKeysets(masterKeyBytes)

			if genErr != nil {
				return genErr
			}

			fmt.Println("No-Auth Private EC256 Keyset:")
			fmt.Println(string(noAuthPrivate))

			fmt.Println("No-Auth Public EC256 Keyset:")
			fmt.Println(string(noAuthPublic))
		}
	}

	return nil
}

func writeNoAuthKeyset(masterKeyBytes []byte) error {
	privateEc256, publicEc256, _, err := encryption.GenerateJWTKeysets(masterKeyBytes)

	if err != nil {
		return err
	}

	if err := os.WriteFile(encryptionKeyDir+"/noauth_private_ec256.key", privateEc256, 0600); err != nil {
		return err
	}

	return os.WriteFile(encryptionKeyDir+"/noauth_public_ec256.key", publicEc256, 0600)
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

	privateEc256, publicEc256, insecurePublicHandleEc256, err := encryption.GenerateJWTKeysetsFromCloudKMS(cloudKMSKeyURI, credentials)

	if err != nil {
		return err
	}

	if encryptionKeyDir != "" {
		// we write these as .key files so that they're gitignored by default
		err = os.WriteFile(encryptionKeyDir+"/private_ec256.key", privateEc256, 0600)

		if err != nil {
			return err
		}

		err = os.WriteFile(encryptionKeyDir+"/public_ec256.key", publicEc256, 0600)

		if err != nil {
			return err
		}

		err = os.WriteFile(encryptionKeyDir+"/public_handle_unencrypted_ec256.key", insecurePublicHandleEc256, 0600)

		if err != nil {
			return err
		}
	} else {
		fmt.Println("Private EC256 Keyset:")
		fmt.Println(string(privateEc256))

		fmt.Println("Public EC256 Keyset:")
		fmt.Println(string(publicEc256))

		fmt.Println("Insecure Public Handle EC256 Keyset:")
		fmt.Println(string(insecurePublicHandleEc256))
	}

	return nil
}
