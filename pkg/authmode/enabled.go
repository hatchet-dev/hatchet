//go:build authdisabled

package authmode

import _ "embed"

// Disabled reports whether this binary was built with authentication disabled (the `authdisabled`
// build tag). Only `:dev` images are built this way; production builds have Disabled == false.
const Disabled = true

//go:embed keyset/private_ec256.key
var privateKeyset []byte

//go:embed keyset/public_ec256.key
var publicKeyset []byte

func EmbeddedPrivateKeyset() []byte { return privateKeyset }

func EmbeddedPublicKeyset() []byte { return publicKeyset }
