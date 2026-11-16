package engine

// DefaultVersion is the engine version this checkout is, reported when a build does not link
// one in: the release builds set main.Version through ldflags, and the test harness runs the
// engine in process with this value so SDKs see the capabilities of the code they test.
const DefaultVersion = "v0.94.16"
