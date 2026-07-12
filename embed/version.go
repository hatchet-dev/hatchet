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
			if d.Path != hatchetModulePath {
				continue
			}
			if d.Replace != nil && isUsableVersion(d.Replace.Version) {
				return d.Replace.Version, nil
			}
			if isUsableVersion(d.Version) {
				return d.Version, nil
			}
		}
	}

	return "", fmt.Errorf("could not resolve the Hatchet module version from build info")
}

func isUsableVersion(v string) bool {
	return strings.HasPrefix(v, "v") && v != "(devel)"
}
