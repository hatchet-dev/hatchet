// Command restartharness measures how much a hatchet-engine restart disrupts
// tenants: `run` drives load through SIGTERM restarts of one engine build and
// writes results, `compare` scores two results files against each other.
// See README.md.
package main

import (
	"fmt"
	"os"
)

const usage = `usage:
  restartharness run --engine-bin <path> --migrate-bin <path> --admin-bin <path> [--track <name>] [--restarts 5] --out <file.json>
  restartharness compare [--tracks <name>,...] <base.json> <head.json>
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "run":
		err = runCmd(os.Args[2:])
	case "compare":
		err = compareCmd(os.Args[2:], os.Stdout)
	default:
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "restartharness:", err)
		os.Exit(1)
	}
}
