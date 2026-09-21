package cli

import (
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"

	"github.com/hatchet-dev/hatchet/pkg/config/cli"
	"github.com/hatchet-dev/hatchet/pkg/config/cli/profilestore"
	"github.com/hatchet-dev/hatchet/pkg/config/loader/loaderutils"
)

var (
	HomeDir   string
	CLIConfig *cli.CLIConfig
	Profiles  *profilestore.Store
	Logger    *log.Logger
)

func init() {
	var err error
	HomeDir, err = os.UserHomeDir()

	if err != nil {
		log.Fatalf("could not get home directory: %v\n", err)
	}

	hatchetDir := filepath.Join(HomeDir, ".hatchet")

	if _, err := os.Stat(hatchetDir); os.IsNotExist(err) {
		os.Mkdir(hatchetDir, 0700) // nolint: errcheck
	} else if err != nil {
		log.Fatalf("could not create hatchet directory: %v\n", err)
	}

	// Load CLI config file
	cliConfigFilePath := filepath.Join(hatchetDir, "config.yaml")

	var cliConfigFileBytes []byte

	if _, err := os.Stat(cliConfigFilePath); err == nil {
		cliConfigFileBytes, err = os.ReadFile(cliConfigFilePath)

		if err != nil {
			log.Fatalf("could not read cli config file: %v\n", err)
		}
	} else if os.IsNotExist(err) {
		// if the file does not exist, create an empty config file
		err := os.WriteFile(cliConfigFilePath, []byte{}, 0600)

		if err != nil {
			log.Fatalf("could not create cli config file: %v\n", err)
		}
	}

	cliConfig, err := loadCLIConfigFile(cliConfigFileBytes)

	if err != nil {
		log.Fatalf("could not load cli config file: %v\n", err)
	}

	CLIConfig = cliConfig

	var logFormatter = log.TextFormatter

	if cliConfig.Logger.Format == "json" {
		logFormatter = log.JSONFormatter
	}

	Logger = log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: true,
		Prefix:          cliConfig.Logger.Prefix,
		Formatter:       logFormatter, // TODO: allow multiple formatters
	})

	// create an empty profiles file if it doesn't exist
	profilesFilePath := filepath.Join(hatchetDir, cliConfig.ProfileFileName)

	if _, err := os.Stat(profilesFilePath); os.IsNotExist(err) {
		err := os.WriteFile(profilesFilePath, []byte{}, 0600)

		if err != nil {
			log.Fatalf("could not create profiles file: %v\n", err)
		}
	} else if err != nil {
		log.Fatalf("could not read profiles file: %v\n", err)
	}

	// open the profile store (the shared public implementation)
	Profiles, err = profilestore.NewStore(hatchetDir, cliConfig.ProfileFileName)

	if err != nil {
		log.Fatalf("could not load profiles config file: %v\n", err)
	}
}

// loadCLIConfigFile loads the CLI config file via viper
func loadCLIConfigFile(files ...[]byte) (*cli.CLIConfig, error) {
	configFile := &cli.CLIConfig{}
	f := cli.BindAllEnv

	var err error

	_, err = loaderutils.LoadConfigFromViper(f, configFile, files...)
	return configFile, err
}
