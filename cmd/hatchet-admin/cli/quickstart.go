package cli

import (
	_ "embed"

	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fatih/color"

	"github.com/spf13/cobra"
)

var certDir string
var skip []string

const (
	StageCerts string = "certs"
)

var quickstartCmd = &cobra.Command{
	Use:   "quickstart",
	Short: "Command used to setup a Hatchet instance",
	Run: func(cmd *cobra.Command, args []string) {
		err := runQuickstart()

		if err != nil {
			red := color.New(color.FgRed)
			red.Printf("Error running [%s]:%s\n", cmd.Use, err.Error())
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(quickstartCmd)

	quickstartCmd.PersistentFlags().StringVar(
		&certDir,
		"cert-dir",
		"./certs",
		"path to the directory where certificates should be stored",
	)

	quickstartCmd.PersistentFlags().StringArrayVar(
		&skip,
		"skip",
		[]string{},
		"a list of steps to skip. possible values are \"certs\"",
	)
}

func runQuickstart() error {
	if !shouldSkip(StageCerts) {
		err := setupCerts()

		if err != nil {
			return fmt.Errorf("could not setup certs: %w", err)
		}
	}

	return nil
}

func shouldSkip(stage string) bool {
	for _, skipStage := range skip {
		if stage == skipStage {
			return true
		}
	}

	return false
}

//go:embed certs/cluster-cert.conf
var ClusterCertConf []byte

//go:embed certs/internal-admin-client-cert.conf
var InternalAdminClientCertConf []byte

//go:embed certs/worker-client-cert.conf
var WorkerClientCertConf []byte

//go:embed certs/generate-certs.sh
var GenerateCertsScript string

func setupCerts() error {
	color.New(color.FgGreen).Printf("Generating certificates in cert directory %s\n", certDir)

	// verify that bash and openssl are installed on the system
	if !commandExists("openssl") {
		return fmt.Errorf("openssl must be installed and available in your $PATH")
	}

	if !commandExists("bash") {
		return fmt.Errorf("bash must be installed and available in your $PATH")
	}

	cwd, err := os.Getwd()

	if err != nil {
		return err
	}

	// write certificate config files to system
	fullPathCertDir := filepath.Join(cwd, certDir)

	err = os.MkdirAll(fullPathCertDir, os.ModePerm)

	if err != nil {
		return fmt.Errorf("could not create cert directory: %w", err)
	}

	err = os.WriteFile(filepath.Join(fullPathCertDir, "./cluster-cert.conf"), ClusterCertConf, 0666)

	if err != nil {
		return fmt.Errorf("could not create cluster-cert.conf file: %w", err)
	}

	err = os.WriteFile(filepath.Join(fullPathCertDir, "./internal-admin-client-cert.conf"), InternalAdminClientCertConf, 0666)

	if err != nil {
		return fmt.Errorf("could not create internal-admin-client-cert.conf file: %w", err)
	}

	err = os.WriteFile(filepath.Join(fullPathCertDir, "./worker-client-cert.conf"), WorkerClientCertConf, 0666)

	if err != nil {
		return fmt.Errorf("could not create worker-client-cert.conf file: %w", err)
	}

	// run openssl commands
	c := exec.Command("bash", "-s", "-", fullPathCertDir)

	c.Stdin = strings.NewReader(GenerateCertsScript)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	err = c.Run()

	if err != nil {
		return err
	}

	return nil
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}
