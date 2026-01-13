package cli

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/charmbracelet/log"
	"github.com/spf13/viper"

	"github.com/hatchet-dev/hatchet/pkg/config/cli"
	"github.com/hatchet-dev/hatchet/pkg/config/loader/loaderutils"
)

var (
	HomeDir             string
	CLIConfig           *cli.CLIConfig
	ProfilesViperConfig *viper.Viper
	Logger              *log.Logger
	viperMutex          sync.RWMutex // Protects ProfilesViperConfig from concurrent access
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

	var logFormatter = log.TextFormatter

	if cliConfig.Logger.Format == "json" {
		logFormatter = log.JSONFormatter
	}

	Logger = log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: true,
		Prefix:          cliConfig.Logger.Prefix,
		Formatter:       logFormatter, // TODO: allow multiple formatters
	})

	// load the profiles file, or write a new one if it doesn't exist
	profilesFilePath := filepath.Join(hatchetDir, cliConfig.ProfileFileName)

	var profilesFileBytes []byte

	if _, err := os.Stat(profilesFilePath); err == nil {
		profilesFileBytes, err = os.ReadFile(profilesFilePath)

		if err != nil {
			log.Fatalf("could not read profiles file: %v\n", err)
		}
	} else if os.IsNotExist(err) {
		// if the file does not exist, create an empty config file
		err := os.WriteFile(profilesFilePath, []byte{}, 0600)

		if err != nil {
			log.Fatalf("could not create profiles file: %v\n", err)
		}
	}

	_, err = loadProfilesConfigFile(profilesFileBytes)

	if err != nil {
		log.Fatalf("could not load profiles config file: %v\n", err)
	}

	ProfilesViperConfig.SetConfigFile(profilesFilePath)
	ProfilesViperConfig.SetConfigType("yaml")
}

// loadCLIConfigFile loads the CLI config file via viper
func loadCLIConfigFile(files ...[]byte) (*cli.CLIConfig, error) {
	configFile := &cli.CLIConfig{}
	f := cli.BindAllEnv

	var err error

	_, err = loaderutils.LoadConfigFromViper(f, configFile, files...)
	return configFile, err
}

// loadProfilesConfigFile loads the profiles config file via viper
func loadProfilesConfigFile(files ...[]byte) (*cli.ProfileFile, error) {
	configFile := &cli.ProfileFile{}

	var err error

	ProfilesViperConfig, err = loaderutils.LoadConfigFromViper(func(v *viper.Viper) {}, configFile, files...)
	return configFile, err
}
