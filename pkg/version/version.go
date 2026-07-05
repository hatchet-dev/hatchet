// Package version holds the canonical Hatchet version.
//
// It is the single source of truth for the version string. Release binaries override their
// package-main Version var via `-ldflags "-X main.Version=<tag>"` at build time, but library
// consumers that import Hatchet directly — notably github.com/hatchet-dev/hatchet/hatchetembed,
// which runs the engine in-process and reports this version to SDK clients — read this constant, so
// it MUST match the release tag. The check-version-matches-tag CI guard enforces Version == tag on
// every release, and it is bumped in the same change that cuts a release.
package version

// Version is the current Hatchet engine/server version.
const Version = "v0.83.4"
