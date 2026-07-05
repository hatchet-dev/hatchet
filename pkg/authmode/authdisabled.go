//go:build authdisabled

package authmode

import _ "embed"

const Disabled = true

//go:embed keyset/private_ec256.key
var privateKeyset []byte

//go:embed keyset/public_ec256.key
var publicKeyset []byte

func EmbeddedPrivateKeyset() []byte { return privateKeyset }

func EmbeddedPublicKeyset() []byte { return publicKeyset }
