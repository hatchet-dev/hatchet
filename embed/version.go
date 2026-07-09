package embed

import (
	"fmt"
	"regexp"
	"runtime/debug"
	"strings"
)

const hatchetModulePath = "github.com/hatchet-dev/hatchet"

var pseudoVersionRe = regexp.MustCompile(`\d{14}-[0-9a-f]{12}`)

func resolveVersion() (string, error) {
	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Path == hatchetModulePath && isUsableTag(info.Main.Version) {
			return info.Main.Version, nil
		}
		for _, d := range info.Deps {
			m := d
			if m.Replace != nil {
				m = m.Replace
			}
			if m.Path == hatchetModulePath && isUsableTag(m.Version) {
				return m.Version, nil
			}
		}
	}

	return "", fmt.Errorf("could not resolve the Hatchet module version from build info; set it explicitly with WithVersion")
}

func isUsableTag(v string) bool {
	return strings.HasPrefix(v, "v") && v != "(devel)" && !pseudoVersionRe.MatchString(v)
}
