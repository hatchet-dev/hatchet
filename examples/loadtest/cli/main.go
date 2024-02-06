package main

import (
	"time"

	"github.com/spf13/cobra"
)

func main() {
	var duration time.Duration
	var events int

	var loadtest = &cobra.Command{
		Use: "loadtest",
		Run: func(cmd *cobra.Command, args []string) {
			do(duration, events)
		},
	}

	loadtest.Flags().IntVarP(&events, "events", "e", 10, "events per second")
	loadtest.Flags().DurationVarP(&duration, "duration", "d", 20*time.Second, "runFor specifies the total time to run the load test")

	cmd := &cobra.Command{Use: "app"}
	cmd.AddCommand(loadtest)
	cmd.Execute()
}
