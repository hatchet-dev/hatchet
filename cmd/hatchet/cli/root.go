package cli

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type runtime string

const (
	runtimeGo         runtime = "go"
	runtimePython     runtime = "python"
	runtimeTypeScript runtime = "typescript"
)

var runtimeGlobWatcher = map[runtime]string{
	runtimeGo:         "**/*.go",
	runtimePython:     "**/*.py",
	runtimeTypeScript: "**/*.{ts,js}",
}

var rootCmd = &cobra.Command{
	Use:   "hatchet",
	Short: "CLI for managing Hatchet workers",
}

func Execute() {
	viper.SetConfigName("config")
	viper.SetConfigType("json")
	viper.AddConfigPath("$HOME/.hatchet")
	err := viper.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// create a new config file
			err := os.MkdirAll(os.ExpandEnv("$HOME/.hatchet"), 0755)
			if err != nil {
				log.Fatalf("Error creating config directory: %v", err)
			}

			err = os.WriteFile(os.ExpandEnv("$HOME/.hatchet/config.json"), []byte("{}"), 0644)
			if err != nil {
				log.Fatalf("Error creating config file: %v", err)
			}
		} else {
			log.Fatalf("Error reading config file: %v", err)
		}
	}

	rootCmd.Execute()
}

func detectRuntimes() []runtime {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Error getting current directory: %v", err)
	}

	var runtimes []runtime

	if couldBeGoRuntime(cwd) {
		runtimes = append(runtimes, runtimeGo)
	}

	if couldBePythonRuntime(cwd) {
		runtimes = append(runtimes, runtimePython)
	}

	if couldBeTypeScriptRuntime(cwd) {
		runtimes = append(runtimes, runtimeTypeScript)
	}

	return runtimes
}

func couldBeGoRuntime(cwd string) bool {
	if goModFile, err := pathExists(filepath.Join(cwd, "go.mod")); err != nil {
		log.Printf("Error checking for go.mod file: %v", err)
	} else if goModFile {
		return true
	}

	var containsGoFiles bool

	_ = filepath.Walk(cwd, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if filepath.Ext(path) == ".go" {
			containsGoFiles = true
			return filepath.SkipAll
		}

		return nil
	})

	return containsGoFiles
}

func couldBePythonRuntime(cwd string) bool {
	if envFile, err := pathExists(filepath.Join(cwd, "environment.yml")); err != nil {
		log.Printf("Error checking for environment.yml file: %v", err)
	} else if envFile {
		return true
	}

	if requirementsFile, err := pathExists(filepath.Join(cwd, "requirements.txt")); err != nil {
		log.Printf("Error checking for requirements.txt file: %v", err)
	} else if requirementsFile {
		return true
	}

	if lockFile, err := pathExists(filepath.Join(cwd, "package-list.txt")); err != nil {
		log.Printf("Error checking for package-list.txt file: %v", err)
	} else if lockFile {
		return true
	}

	if pyprojectTOMLFile, err := pathExists(filepath.Join(cwd, "pyproject.toml")); err != nil {
		log.Printf("Error checking for pyproject.toml file: %v", err)
	} else if pyprojectTOMLFile {
		return true
	}

	var containsPythonFiles bool

	_ = filepath.Walk(cwd, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if filepath.Ext(path) == ".py" {
			containsPythonFiles = true
			return filepath.SkipAll
		}

		return nil
	})

	return containsPythonFiles
}

func couldBeTypeScriptRuntime(cwd string) bool {
	if packageJSONFile, err := pathExists(filepath.Join(cwd, "package.json")); err != nil {
		log.Printf("Error checking for package.json file: %v", err)
	} else if packageJSONFile {
		return true
	}

	if yarnLockFile, err := pathExists(filepath.Join(cwd, "yarn.lock")); err != nil {
		log.Printf("Error checking for yarn.lock file: %v", err)
	} else if yarnLockFile {
		return true
	}

	if pnpmLockFile, err := pathExists(filepath.Join(cwd, "pnpm-lock.yaml")); err != nil {
		log.Printf("Error checking for pnpm-lock.yaml file: %v", err)
	} else if pnpmLockFile {
		return true
	}

	if bunbLockFile, err := pathExists(filepath.Join(cwd, "bun.lockb")); err != nil {
		log.Printf("Error checking for bun.lockb file: %v", err)
	} else if bunbLockFile {
		return true
	}

	if bunLockFile, err := pathExists(filepath.Join(cwd, "bun.lock")); err != nil {
		log.Printf("Error checking for bun.lockb file: %v", err)
	} else if bunLockFile {
		return true
	}

	if denoJSONFile, err := pathExists(filepath.Join(cwd, "deno.json")); err != nil {
		log.Printf("Error checking for deno.json file: %v", err)
	} else if denoJSONFile {
		return true
	}

	if denoJSONCFile, err := pathExists(filepath.Join(cwd, "deno.jsonc")); err != nil {
		log.Printf("Error checking for deno.jsonc file: %v", err)
	} else if denoJSONCFile {
		return true
	}

	var containsTypeScriptFiles bool

	_ = filepath.Walk(cwd, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if filepath.Ext(path) == ".ts" || filepath.Ext(path) == ".js" {
			containsTypeScriptFiles = true
			return filepath.SkipAll
		}

		return nil
	})

	return containsTypeScriptFiles
}
