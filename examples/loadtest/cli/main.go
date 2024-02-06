package main

import (
	"time"

	"github.com/spf13/cobra"
)

func main() {
	var runFor time.Duration
	var events int

	var loadtest = &cobra.Command{
		Use: "loadtest",
		Run: func(cmd *cobra.Command, args []string) {
			do(runFor, events)
		},
	}

	loadtest.Flags().IntVarP(&events, "events", "e", 10, "events per second")
	loadtest.Flags().DurationVarP(&runFor, "runFor", "r", 20*time.Second, "runFor specifies the total time to run the load test")

	var rootCmd = &cobra.Command{Use: "app"}
	rootCmd.AddCommand(loadtest)
	rootCmd.Execute()
}
