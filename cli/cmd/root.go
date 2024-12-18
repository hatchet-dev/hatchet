/*
Copyright © 2024 Hatchet Technologies Inc. <support@hatchet.run>
*/
package cfg

import (
	"context"
	"errors"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Version: "0.0.1",
	Use:     "hatchet",
	Short:   "Hatchet-CLI (hatchet) for local development and cloud deployment of workflows",
	Long: `Hatchet-CLI (hatchet) for local development and cloud deployment of workflows.

For more information, visit https://docs.hatchet.run`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			return err
		}
		viper.AutomaticEnv()
		viper.SetEnvPrefix("hatchet")

		if _, err := os.Stat(viper.GetString(".hatchet")); errors.Is(err, os.ErrNotExist) {
			return errors.New(err.Error() + ": please run init to configure hatchet\n")
		}
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	rootCmd.AddCommand(initialize())

	_, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	//Define root flags
	// rootCmd.PersistentFlags().String(cfgPath, dir+cfgDir+cfgFile, "location of the hatchet config file")

	return rootCmd.ExecuteContext(context.Background())
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cli.yaml)")
}
