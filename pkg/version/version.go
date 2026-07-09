// Package version is the fallback Hatchet version for source/dev builds. Released binaries set it via
// -ldflags; the embed package prefers the module version from the consumer's build info.
package version

const Version = "v0.83.4"
