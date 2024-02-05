package loader

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

type addresses struct {
	serverURL            string
	grpcBroadcastAddress string
}

func getAddressesFromJWT(token string) (*addresses, error) {
	claims, err := extractClaimsFromJWT(token)
	if err != nil {
		return nil, err
	}

	serverURL, ok := claims["server_url"].(string)
	if !ok {
		return nil, fmt.Errorf("server_url claim not found")
	}

	grpcBroadcastAddress, ok := claims["grpc_broadcast_address"].(string)
	if !ok {
		return nil, fmt.Errorf("grpc_broadcast_address claim not found")
	}

	return &addresses{
		serverURL:            serverURL,
		grpcBroadcastAddress: grpcBroadcastAddress,
	}, nil
}

func extractClaimsFromJWT(token string) (map[string]interface{}, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}

	claimsData, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}

	var claims map[string]interface{}
	err = json.Unmarshal(claimsData, &claims)
	if err != nil {
		return nil, err
	}

	return claims, nil
}
