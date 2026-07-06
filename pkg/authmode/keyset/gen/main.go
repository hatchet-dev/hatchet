// Command gen regenerates the embedded auth-disabled API token (keyset/token.jwt), signed with the
// committed private keyset. Run from the repo root:
//
//	go run ./pkg/authmode/keyset/gen > pkg/authmode/keyset/token.jwt
//
// The claims are fixed (see pkg/authmode/token.go) so the one token validates on every authdisabled
// instance. Only rerun this if the keyset or those claims change.
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/tink-crypto/tink-go/jwt"

	"github.com/hatchet-dev/hatchet/pkg/authmode"
	"github.com/hatchet-dev/hatchet/pkg/encryption"
)

func main() {
	keysetBytes, err := os.ReadFile("pkg/authmode/keyset/private_ec256.key")
	if err != nil {
		fatal(err)
	}

	handle, err := encryption.InsecureHandleFromBytes(keysetBytes)
	if err != nil {
		fatal(err)
	}

	signer, err := jwt.NewSigner(handle)
	if err != nil {
		fatal(err)
	}

	issuedAt := time.Now().UTC()
	// effectively never; the validator requires an exp claim so we can't omit it
	expiresAt := time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
	audience := authmode.EmbeddedTokenAudience
	issuer := authmode.EmbeddedTokenIssuer
	subject := authmode.EmbeddedTokenTenantID

	rawJWT, err := jwt.NewRawJWT(&jwt.RawJWTOptions{
		IssuedAt:  &issuedAt,
		ExpiresAt: &expiresAt,
		Audience:  &audience,
		Issuer:    &issuer,
		Subject:   &subject,
		CustomClaims: map[string]interface{}{
			"token_id":               authmode.EmbeddedTokenID,
			"server_url":             authmode.EmbeddedTokenServerURL,
			"grpc_broadcast_address": authmode.EmbeddedTokenGRPCAddress,
		},
	})
	if err != nil {
		fatal(err)
	}

	token, err := signer.SignAndEncode(rawJWT)
	if err != nil {
		fatal(err)
	}

	fmt.Print(token)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
