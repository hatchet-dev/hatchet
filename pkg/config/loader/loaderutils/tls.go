package loaderutils

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/hatchet-dev/hatchet/pkg/config/client"
	"github.com/hatchet-dev/hatchet/pkg/config/shared"
)

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

	res := &tls.Config{
		MinVersion: tls.VersionTLS13,
	}

	if len(x509Cert.Certificate) != 0 {
		res.Certificates = []tls.Certificate{x509Cert}
	}

	return res, ca, nil
}
