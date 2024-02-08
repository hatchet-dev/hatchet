package main

import (
	"time"

	"github.com/spf13/cobra"
)

func main() {
	var duration time.Duration
	var wait time.Duration
	var events int

	var loadtest = &cobra.Command{
		Use: "loadtest",
		Run: func(cmd *cobra.Command, args []string) {
			if err := do(duration, events, wait); err != nil {
				panic(err)
			}
		},
	}

	loadtest.Flags().IntVarP(&events, "events", "e", 10, "events per second")
	loadtest.Flags().DurationVarP(&duration, "duration", "d", 10*time.Second, "duration specifies the total time to run the load test")
	loadtest.Flags().DurationVarP(&wait, "wait", "w", 10*time.Second, "wait specifies the total time to wait until events complete")

	cmd := &cobra.Command{Use: "app"}
	cmd.AddCommand(loadtest)
	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}
