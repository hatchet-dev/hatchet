package loaderutils

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type TokenConf struct {
	ExpiresAt            time.Time
	ServerURL            string
	GrpcBroadcastAddress string
	TenantId             string
}

func GetConfFromJWT(token string) (*TokenConf, error) {
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

	// parse the exp claim
	expiresAtFloat, ok := claims["exp"].(float64)
	if !ok {
		return nil, fmt.Errorf("exp claim not found")
	}

	expiresAt := time.Unix(int64(expiresAtFloat), 0)

	tenantId, ok := claims["sub"].(string)

	if !ok {
		return nil, fmt.Errorf("sub claim not found")
	}

	return &TokenConf{
		ExpiresAt:            expiresAt,
		ServerURL:            serverURL,
		GrpcBroadcastAddress: grpcBroadcastAddress,
		TenantId:             tenantId,
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
