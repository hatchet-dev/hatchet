package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/pingcap/errors"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/hatchet-dev/hatchet/pkg/config/loader"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/encryption"
	"github.com/hatchet-dev/hatchet/pkg/random"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var k8sQuickstartSkip []string
var k8sQuickstartOverwrite bool
var namespace string
var k8sQuickstartConfigName string
var k8sClientConfigName string
var k8sConfigResourceType string

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

var k8sWorkerTokenCmd = &cobra.Command{
	Use:   "create-worker-token",
	Short: "Generates a worker token within a secret on a Hatchet instance on Kubernetes. This command is only meant to run within a pod on a cluster with read/write access to secrets within the namespace.",
	Run: func(cmd *cobra.Command, args []string) {
		err := runCreateWorkerToken()

		if err != nil {
			red := color.New(color.FgRed)
			red.Printf("Error running [%s]:%s\n", cmd.Use, err.Error())
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(k8sCmd)

	k8sCmd.PersistentFlags().StringVar(
		&k8sConfigResourceType,
		"resource-type",
		"secret",
		"the type of resource to store config in (secret or configmap)",
	)

	k8sCmd.PersistentFlags().StringVar(
		&namespace,
		"namespace",
		"default",
		"the namespace to use for the secret/configmap",
	)

	k8sCmd.AddCommand(k8sQuickstartCmd)

	k8sQuickstartCmd.PersistentFlags().StringVar(
		&k8sQuickstartConfigName,
		"resource-name",
		"hatchet-config",
		"the name of the secret/configmap to use",
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
		false,
		"whether existing configmap should be overwritten, if it exists",
	)

	k8sCmd.AddCommand(k8sWorkerTokenCmd)

	k8sWorkerTokenCmd.PersistentFlags().StringVar(
		&k8sClientConfigName,
		"resource-name",
		"hatchet-client-config",
		"the name of the secret/configmap to use",
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
	var c *configModifier

	if k8sConfigResourceType == "secret" {
		secret, err := clientset.CoreV1().Secrets(namespace).Get(context.Background(), k8sQuickstartConfigName, metav1.GetOptions{})

		switch {
		case err != nil && !errors.IsNotFound(err):
			return fmt.Errorf("error getting secret: %w", err)
		case err != nil:
			exists = false
			c = newFromSecret(nil, k8sQuickstartConfigName)
		default:
			exists = secret != nil
			c = newFromSecret(secret, k8sQuickstartConfigName)
		}
	} else {
		configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.Background(), k8sQuickstartConfigName, metav1.GetOptions{})

		switch {
		case err != nil && !errors.IsNotFound(err):
			return fmt.Errorf("error getting configmap: %w", err)
		case err != nil:
			exists = false
			c = newFromConfigMap(nil, k8sQuickstartConfigName)
		default:
			exists = configMap != nil
			c = newFromConfigMap(configMap, k8sQuickstartConfigName)
		}
	}

	res := generatedConfig{
		authCookieSecrets:          c.get("SERVER_AUTH_COOKIE_SECRETS"),
		encryptionMasterKeyset:     c.get("SERVER_ENCRYPTION_MASTER_KEYSET"),
		encryptionJwtPrivateKeyset: c.get("SERVER_ENCRYPTION_JWT_PRIVATE_KEYSET"),
		encryptionJwtPublicKeyset:  c.get("SERVER_ENCRYPTION_JWT_PUBLIC_KEYSET"),
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

	// create or update the config
	c.set("SERVER_AUTH_COOKIE_SECRETS", res.authCookieSecrets)
	c.set("SERVER_ENCRYPTION_MASTER_KEYSET", res.encryptionMasterKeyset)
	c.set("SERVER_ENCRYPTION_JWT_PRIVATE_KEYSET", res.encryptionJwtPrivateKeyset)
	c.set("SERVER_ENCRYPTION_JWT_PUBLIC_KEYSET", res.encryptionJwtPublicKeyset)

	if exists {
		err = c.updateResource(clientset)
	} else {
		err = c.createResource(clientset)
	}

	if err != nil {
		return err
	}

	return nil
}

func runCreateWorkerToken() error {
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

	// read in the local config
	configLoader := loader.NewConfigLoader(configDirectory)

	cleanup, server, err := configLoader.CreateServerFromConfig("", func(scf *server.ServerConfigFile) {
		// disable rabbitmq since it's not needed to create the api token
		scf.MessageQueue.Enabled = false

		// disable security checks since we're not running the server
		scf.SecurityCheck.Enabled = false
	})

	if err != nil {
		return err
	}

	defer cleanup() // nolint:errcheck

	defer server.Disconnect() // nolint:errcheck

	expiresAt := time.Now().UTC().Add(100 * 365 * 24 * time.Hour)

	tenantId := tokenTenantId

	if tenantId == "" {
		tenantId = server.Seed.DefaultTenantID
	}

	defaultTok, err := server.Auth.JWTManager.GenerateTenantToken(context.Background(), tenantId, tokenName, false, &expiresAt)

	if err != nil {
		return err
	}

	var c *configModifier
	var exists bool

	if k8sConfigResourceType == "secret" {
		secret, err := clientset.CoreV1().Secrets(namespace).Get(context.Background(), k8sClientConfigName, metav1.GetOptions{})

		switch {
		case err != nil && !errors.IsNotFound(err):
			return fmt.Errorf("error getting secret: %w", err)
		case err != nil:
			exists = false
			c = newFromSecret(nil, k8sClientConfigName)
		default:
			exists = secret != nil
			c = newFromSecret(secret, k8sClientConfigName)
		}
	} else {
		configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.Background(), k8sClientConfigName, metav1.GetOptions{})

		switch {
		case err != nil && !errors.IsNotFound(err):
			return fmt.Errorf("error getting configmap: %w", err)
		case err != nil:
			exists = false
			c = newFromConfigMap(nil, k8sClientConfigName)
		default:
			exists = configMap != nil
			c = newFromConfigMap(configMap, k8sClientConfigName)
		}
	}

	c.set("HATCHET_CLIENT_TOKEN", defaultTok.Token)

	if exists {
		err = c.updateResource(clientset)
	} else {
		err = c.createResource(clientset)
	}

	if err != nil {
		return err
	}

	return nil
}

type configModifier struct {
	*corev1.Secret
	*corev1.ConfigMap
}

func newFromSecret(s *corev1.Secret, name string) *configModifier {
	// if secret is nil, create a new one
	if s == nil {
		s = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		}
	}

	return &configModifier{
		Secret: s,
	}
}

func newFromConfigMap(c *corev1.ConfigMap, name string) *configModifier {
	if c == nil {
		c = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		}
	}

	return &configModifier{
		ConfigMap: c,
	}
}

func (c *configModifier) set(key, value string) {
	if c.Secret != nil {
		if c.Secret.Data == nil {
			c.Secret.Data = map[string][]byte{}
		}

		c.Secret.Data[key] = []byte(value)
	}

	if c.ConfigMap != nil {
		if c.ConfigMap.Data == nil {
			c.ConfigMap.Data = map[string]string{}
		}

		c.ConfigMap.Data[key] = value
	}
}

func (c *configModifier) get(key string) string {
	if c.Secret != nil {
		return string(c.Secret.Data[key])
	}

	if c.ConfigMap != nil {
		return c.ConfigMap.Data[key]
	}

	return ""
}

func (c *configModifier) createResource(clientset *kubernetes.Clientset) error {
	if c.Secret != nil {
		_, err := clientset.CoreV1().Secrets(namespace).Create(context.Background(), c.Secret, metav1.CreateOptions{})

		if err != nil {
			return fmt.Errorf("error creating secret: %w", err)
		}
	}

	if c.ConfigMap != nil {
		_, err := clientset.CoreV1().ConfigMaps(namespace).Create(context.Background(), c.ConfigMap, metav1.CreateOptions{})

		if err != nil {
			return fmt.Errorf("error creating configmap: %w", err)
		}
	}

	return nil
}

func (c *configModifier) updateResource(clientset *kubernetes.Clientset) error {
	if c.Secret != nil {
		_, err := clientset.CoreV1().Secrets(namespace).Update(context.Background(), c.Secret, metav1.UpdateOptions{})

		if err != nil {
			return fmt.Errorf("error updating secret: %w", err)
		}
	}

	if c.ConfigMap != nil {
		_, err := clientset.CoreV1().ConfigMaps(namespace).Update(context.Background(), c.ConfigMap, metav1.UpdateOptions{})

		if err != nil {
			return fmt.Errorf("error updating configmap: %w", err)
		}
	}

	return nil
}
