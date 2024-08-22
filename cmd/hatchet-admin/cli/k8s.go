package cli

import (
	"context"
	"os"

	"github.com/fatih/color"
	"github.com/pingcap/errors"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/hatchet-dev/hatchet/pkg/encryption"
	"github.com/hatchet-dev/hatchet/pkg/random"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var k8sQuickstartSkip []string
var k8sQuickstartOverwrite bool
var namespace string
var k8sResourceName string

var k8sCmd = &cobra.Command{
	Use:   "k8s",
	Short: "Commands used to setup a Hatchet instance on Kubernetes",
}

var k8sQuickstartCmd = &cobra.Command{
	Use:   "quickstart",
	Short: "Quickstart for generating environment variables on a Hatchet instance on Kubernetes. This command is only meant to run within a pod on a cluster with read/write access to configmaps within the namespace.",
	Run: func(cmd *cobra.Command, args []string) {
		err := runK8sQuickstart()

		if err != nil {
			red := color.New(color.FgRed)
			red.Printf("Error running [%s]:%s\n", cmd.Use, err.Error())
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(k8sCmd)
	k8sCmd.AddCommand(k8sQuickstartCmd)

	k8sQuickstartCmd.PersistentFlags().StringVar(
		&namespace,
		"namespace",
		"default",
		"the namespace to use for the configmap",
	)

	k8sQuickstartCmd.PersistentFlags().StringVar(
		&k8sResourceName,
		"resource-name",
		"hatchet-config",
		"the name of the configmap to use",
	)

	k8sQuickstartCmd.PersistentFlags().StringArrayVar(
		&k8sQuickstartSkip,
		"skip",
		[]string{},
		"a list of steps to skip. possible values are \"keys\"",
	)

	k8sQuickstartCmd.PersistentFlags().BoolVar(
		&k8sQuickstartOverwrite,
		"overwrite",
		true,
		"whether existing configmap should be overwritten, if it exists",
	)
}

type generatedConfig struct {
	authCookieSecrets          string
	encryptionMasterKeyset     string
	encryptionJwtPrivateKeyset string
	encryptionJwtPublicKeyset  string
}

func runK8sQuickstart() error {
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		return err
	}

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)

	if err != nil {
		return err
	}

	// read from the configmap
	exists := false
	configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.Background(), k8sResourceName, metav1.GetOptions{})

	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	res := generatedConfig{}

	if exists && configMap.Data != nil {
		res.authCookieSecrets = configMap.Data["SERVER_AUTH_COOKIE_SECRETS"]
		res.encryptionMasterKeyset = configMap.Data["SERVER_ENCRYPTION_MASTER_KEYSET"]
		res.encryptionJwtPrivateKeyset = configMap.Data["SERVER_ENCRYPTION_JWT_PRIVATE_KEYSET"]
		res.encryptionJwtPublicKeyset = configMap.Data["SERVER_ENCRYPTION_JWT_PUBLIC_KEYSET"]
	}

	if k8sQuickstartOverwrite || res.authCookieSecrets == "" {
		// generate the random strings for SERVER_AUTH_COOKIE_SECRETS
		authCookieSecret1, err := random.Generate(16)

		if err != nil {
			return err
		}

		authCookieSecret2, err := random.Generate(16)

		if err != nil {
			return err
		}

		res.authCookieSecrets = authCookieSecret1 + " " + authCookieSecret2
	}

	if k8sQuickstartOverwrite || res.encryptionMasterKeyset == "" || res.encryptionJwtPrivateKeyset == "" || res.encryptionJwtPublicKeyset == "" {
		masterKeyBytes, privateEc256, publicEc256, err := encryption.GenerateLocalKeys()

		if err != nil {
			return err
		}

		res.encryptionMasterKeyset = string(masterKeyBytes)
		res.encryptionJwtPrivateKeyset = string(privateEc256)
		res.encryptionJwtPublicKeyset = string(publicEc256)
	}

	// create or update the configmap
	configMap = &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: k8sResourceName,
		},
		Data: map[string]string{
			"SERVER_AUTH_COOKIE_SECRETS":           res.authCookieSecrets,
			"SERVER_ENCRYPTION_MASTER_KEYSET":      res.encryptionMasterKeyset,
			"SERVER_ENCRYPTION_JWT_PRIVATE_KEYSET": res.encryptionJwtPrivateKeyset,
			"SERVER_ENCRYPTION_JWT_PUBLIC_KEYSET":  res.encryptionJwtPublicKeyset,
		},
	}

	if exists {
		_, err = clientset.CoreV1().ConfigMaps(namespace).Update(context.Background(), configMap, metav1.UpdateOptions{})

		if err != nil {
			return err
		}
	} else {
		_, err = clientset.CoreV1().ConfigMaps(namespace).Create(context.Background(), configMap, metav1.CreateOptions{})

		if err != nil {
			return err
		}
	}

	return nil
}
