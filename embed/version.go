package embed

import (
	"fmt"
	"runtime/debug"
	"strings"
)

const hatchetModulePath = "github.com/hatchet-dev/hatchet"

func resolveVersion() (string, error) {
	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Path == hatchetModulePath && isUsableVersion(info.Main.Version) {
			return info.Main.Version, nil
		}
		for _, d := range info.Deps {
			m := d
			if m.Replace != nil {
				m = m.Replace
			}
			if m.Path == hatchetModulePath && isUsableVersion(m.Version) {
				return m.Version, nil
			}
		}
	}

	return "", fmt.Errorf("could not resolve the Hatchet module version from build info")
}

func isUsableVersion(v string) bool {
	return strings.HasPrefix(v, "v") && v != "(devel)"
}
