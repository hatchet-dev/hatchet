package loaderutils

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"

	"github.com/hatchet-dev/hatchet/pkg/config/client"
	"github.com/hatchet-dev/hatchet/pkg/config/shared"
)

// ParseTLSMinVersion parses a TLS minimum version string.
// Empty defaults to TLS 1.3.
func ParseTLSMinVersion(s string) (uint16, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "1.3", "tls1.3", "tls_1.3", "tls13":
		return tls.VersionTLS13, nil
	case "1.2", "tls1.2", "tls_1.2", "tls12":
		return tls.VersionTLS12, nil
	default:
		return 0, fmt.Errorf("unsupported TLS minimum version %q: must be \"1.2\" or \"1.3\"", s)
	}
}

func LoadClientTLSConfig(tlsConfig *client.ClientTLSConfigFile, serverName string) (*tls.Config, error) {
	res, ca, err := LoadBaseTLSConfig(&tlsConfig.Base)

	if err != nil {
		return nil, err
	}

	res.ServerName = serverName

	switch tlsConfig.Base.TLSStrategy {
	case "tls":
		if ca != nil {
			res.RootCAs = ca
		}
	case "mtls":
		res.ServerName = tlsConfig.TLSServerName
		res.RootCAs = ca
	case "none":
		return nil, nil
	default:
		return nil, fmt.Errorf("invalid TLS strategy: %s", tlsConfig.Base.TLSStrategy)
	}

	return res, nil
}

func LoadServerTLSConfig(tlsConfig *shared.TLSConfigFile) (*tls.Config, error) {
	res, ca, err := LoadBaseTLSConfig(tlsConfig)

	if err != nil {
		return nil, err
	}

	switch tlsConfig.TLSStrategy {
	case "tls":
		res.ClientAuth = tls.VerifyClientCertIfGiven

		if ca != nil {
			res.ClientCAs = ca
		}
	case "mtls":
		if ca == nil {
			return nil, fmt.Errorf("Client CA is required for mTLS")
		}

		res.ClientAuth = tls.RequireAndVerifyClientCert
		res.ClientCAs = ca
	default:
		return nil, fmt.Errorf("invalid TLS strategy: %s", tlsConfig.TLSStrategy)
	}

	return res, nil
}

func LoadBaseTLSConfig(tlsConfig *shared.TLSConfigFile) (*tls.Config, *x509.CertPool, error) {
	var x509Cert tls.Certificate
	var err error

	switch {
	case tlsConfig.TLSCert != "" && tlsConfig.TLSKey != "":
		x509Cert, err = tls.X509KeyPair([]byte(tlsConfig.TLSCert), []byte(tlsConfig.TLSKey))
	case tlsConfig.TLSCertFile != "" && tlsConfig.TLSKeyFile != "":
		x509Cert, err = tls.LoadX509KeyPair(tlsConfig.TLSCertFile, tlsConfig.TLSKeyFile)
	}

	var caBytes []byte

	switch {
	case tlsConfig.TLSRootCA != "":
		caBytes = []byte(tlsConfig.TLSRootCA)
	case tlsConfig.TLSRootCAFile != "":
		caBytes, err = os.ReadFile(tlsConfig.TLSRootCAFile)
	}

	var ca *x509.CertPool

	if len(caBytes) != 0 {
		ca = x509.NewCertPool()

		if ok := ca.AppendCertsFromPEM(caBytes); !ok {
			return nil, nil, fmt.Errorf("could not append root CA to cert pool: %w", err)
		}
	}

	minVersion, err := ParseTLSMinVersion(tlsConfig.TLSMinVersion)
	if err != nil {
		return nil, nil, err
	}

	res := &tls.Config{
		MinVersion: minVersion,
	}

	if len(x509Cert.Certificate) != 0 {
		res.Certificates = []tls.Certificate{x509Cert}
	}

	return res, ca, nil
}
