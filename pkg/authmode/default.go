//go:build !authdisabled

package authmode

const IsDisabled = false

func EmbeddedPrivateKeyset() []byte { return nil }

func EmbeddedPublicKeyset() []byte { return nil }

func EmbeddedToken() string { return "" }
