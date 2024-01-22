package loaderutils

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/hatchet-dev/hatchet/internal/config/client"
	"github.com/hatchet-dev/hatchet/internal/config/shared"
)

func LoadClientTLSConfig(tlsConfig *client.ClientTLSConfigFile) (*tls.Config, error) {
	res, ca, err := LoadBaseTLSConfig(&tlsConfig.Base)

	if err != nil {
		return nil, err
	}

	res.ServerName = tlsConfig.TLSServerName
	res.RootCAs = ca

	return res, nil
}

func LoadServerTLSConfig(tlsConfig *shared.TLSConfigFile) (*tls.Config, error) {
	res, ca, err := LoadBaseTLSConfig(tlsConfig)

	if err != nil {
		return nil, err
	}

	res.ClientAuth = tls.RequireAndVerifyClientCert
	res.ClientCAs = ca

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
	default:
		return nil, nil, fmt.Errorf("no cert or key provided")
	}

	var caBytes []byte

	switch {
	case tlsConfig.TLSRootCA != "":
		caBytes = []byte(tlsConfig.TLSRootCA)
	case tlsConfig.TLSRootCAFile != "":
		caBytes, err = os.ReadFile(tlsConfig.TLSRootCAFile)
	default:
		return nil, nil, fmt.Errorf("no root CA provided")
	}

	ca := x509.NewCertPool()

	if ok := ca.AppendCertsFromPEM(caBytes); !ok {
		return nil, nil, fmt.Errorf("could not append root CA to cert pool: %w", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{x509Cert},
		MinVersion:   tls.VersionTLS13,
	}, ca, nil
}
