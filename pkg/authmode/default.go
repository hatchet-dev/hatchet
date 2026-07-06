//go:build !authdisabled

package authmode

const Disabled = false

func EmbeddedPrivateKeyset() []byte { return nil }

func EmbeddedPublicKeyset() []byte { return nil }

func EmbeddedToken() string { return "" }
