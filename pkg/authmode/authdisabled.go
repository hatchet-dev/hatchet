//go:build authdisabled

package authmode

import (
	"bytes"
	_ "embed"
)

const Disabled = true

//go:embed keyset/private_ec256.key
var privateKeyset []byte

//go:embed keyset/public_ec256.key
var publicKeyset []byte

//go:embed keyset/token.jwt
var embeddedToken []byte

// TrimSpace guards against an editor adding a trailing newline to the committed keyset files,
// which would corrupt the base64.
func EmbeddedPrivateKeyset() []byte { return bytes.TrimSpace(privateKeyset) }

func EmbeddedPublicKeyset() []byte { return bytes.TrimSpace(publicKeyset) }

func EmbeddedToken() string { return string(bytes.TrimSpace(embeddedToken)) }
