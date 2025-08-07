package cli

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage Hatchet configuration",
}

var setTokenCmd = &cobra.Command{
	Use:   "set-token",
	Short: "Set the API token",
	Run:   setToken,
	Args:  cobra.ExactArgs(1),
}

func init() {
	configCmd.AddCommand(setTokenCmd)

	rootCmd.AddCommand(configCmd)
}

func setToken(cmd *cobra.Command, args []string) {
	token := args[0]

	if token == "" {
		log.Fatalf("Token cannot be empty")
	}

	viper.Set("token", token)
	err := viper.WriteConfig()
	if err != nil {
		log.Fatalf("Error setting token: %v", err)
	}

	fmt.Println("Token set successfully")
}
