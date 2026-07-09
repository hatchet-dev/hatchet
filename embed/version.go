package embed

import (
	"regexp"
	"runtime/debug"
	"strings"

	"github.com/hatchet-dev/hatchet/pkg/version"
)

const hatchetModulePath = "github.com/hatchet-dev/hatchet"

var pseudoVersionRe = regexp.MustCompile(`\d{14}-[0-9a-f]{12}`)

var embedVersion = resolveVersion()

func resolveVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Path == hatchetModulePath && isUsableTag(info.Main.Version) {
			return info.Main.Version
		}
		for _, d := range info.Deps {
			m := d
			if m.Replace != nil {
				m = m.Replace
			}
			if m.Path == hatchetModulePath && isUsableTag(m.Version) {
				return m.Version
			}
		}
	}
	return version.Version
}

func isUsableTag(v string) bool {
	return strings.HasPrefix(v, "v") && v != "(devel)" && !pseudoVersionRe.MatchString(v)
}
