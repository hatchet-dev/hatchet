package main

import (
	"time"

	"github.com/spf13/cobra"
)

func main() {
	var runFor time.Duration
	var sleep time.Duration
	var delay time.Duration
	var amount int

	var loadtest = &cobra.Command{
		Use: "loadtest",
		Run: func(cmd *cobra.Command, args []string) {
			do(runFor, sleep, delay, amount)
		},
	}

	loadtest.Flags().DurationVarP(&delay, "delay", "d", 500*time.Millisecond, "delay between sending events")
	loadtest.Flags().DurationVarP(&sleep, "sleep", "s", 10*time.Second, "delay between batch executions")
	loadtest.Flags().DurationVarP(&runFor, "runFor", "r", 20*time.Second, "runFor specifies the total time to run the load test")
	loadtest.Flags().IntVarP(&amount, "amount", "a", 10, "amount of continuous events")

	var rootCmd = &cobra.Command{Use: "app"}
	rootCmd.AddCommand(loadtest)
	rootCmd.Execute()
}
