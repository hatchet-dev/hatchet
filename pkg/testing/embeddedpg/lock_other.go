//go:build !unix

package embeddedpg

func lockBinaries(binPath string) (func(), error) {
	return func() {}, nil
}
