package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

type LatencySnapshot struct {
	t       time.Time
	latency time.Duration
}
type LatencyResult struct {
	snapshots []LatencySnapshot
}

func (lr *LatencyResult) PlotLatency(outputFile string) error {
	line := charts.NewLine()

	xvals := make([]string, 0, len(lr.snapshots))
	yvals := make([]opts.LineData, 0, len(lr.snapshots))
	start := lr.snapshots[0].t

	for _, s := range lr.snapshots {
		elapsedMs := float64(s.t.Sub(start).Seconds())

		xvals = append(xvals, fmt.Sprintf("%f", elapsedMs))
		yvals = append(yvals, opts.LineData{
			Value: float64(s.latency.Microseconds()) / 1000.0, // ms
		})
	}

	line.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: "Latency Over Time",
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Name: "Latency (ms)",
		}),
		charts.WithXAxisOpts(opts.XAxis{
			Name: "Time",
		}),
	)

	line.SetXAxis(xvals).
		AddSeries("Latency", yvals)

	f, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer f.Close()

	return line.Render(f)
}

type avgResult struct {
	count         int64
	avg           time.Duration
	latencyResult LatencyResult
}

func do(config LoadTestConfig) error {
	l.Info().Msgf("testing with duration=%s, eventsPerSecond=%d, delay=%s, wait=%s, concurrency=%d, averageDurationThreshold=%s", config.Duration, config.Events, config.Delay, config.Wait, config.Concurrency, config.AverageDurationThreshold)

	after := 10 * time.Second

	// The worker may intentionally be delayed (WorkerDelay) before it starts consuming tasks.
	// The test timeout must include this delay, otherwise we can cancel while work is still expected to complete.
	timeout := config.WorkerDelay + after + config.Duration + config.Wait + 30*time.Second

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ch := make(chan int64, 2)
	durations := make(chan executionEvent, config.Events)

	// Compute running average for executed durations using a rolling average.
	durationsResult := make(chan avgResult)
	go func() {
		var count int64
		var avg time.Duration
		var snapshots []LatencySnapshot

		for d := range durations {
			count++
			if count == 1 {
				avg = d.duration
			} else {
				avg += (d.duration - avg) / time.Duration(count)
			}
			snapshots = append(snapshots, LatencySnapshot{
				t:       d.startedAt,
				latency: d.duration,
			})
		}
		durationsResult <- avgResult{count: count, avg: avg, latencyResult: LatencyResult{snapshots: snapshots}}
	}()

	// Start worker and ensure it has time to register
	workerStarted := make(chan struct{})

	go func() {
		if config.WorkerDelay > 0 {
			// run a worker to register the workflow
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			run(ctx, config, durations)
			cancel()
			l.Info().Msgf("wait %s before starting the worker", config.WorkerDelay)
			time.Sleep(config.WorkerDelay)
		}
		l.Info().Msg("starting worker now")

		// Signal that worker is starting
		close(workerStarted)

		count, uniques := run(ctx, config, durations)
		close(durations)
		ch <- count
		ch <- uniques
	}()

	// Wait for worker to start, then give it time to register workflows
	<-workerStarted
	time.Sleep(after)

	scheduled := make(chan time.Duration, config.Events)

	// Compute running average for scheduled times using a rolling average.
	scheduledResult := make(chan avgResult)
	go func() {
		var count int64
		var avg time.Duration
		var snapshots []LatencySnapshot
		for d := range scheduled {
			count++
			if count == 1 {
				avg = d
			} else {
				avg += (d - avg) / time.Duration(count)
			}
			snapshots = append(snapshots, LatencySnapshot{
				t:       time.Now(),
				latency: d,
			})
		}
		scheduledResult <- avgResult{count: count, avg: avg, latencyResult: LatencyResult{snapshots: snapshots}}
	}()

	emitted := emit(ctx, config.Namespace, config.Events, config.Duration, scheduled, config.PayloadSize)
	close(scheduled)

	executed := <-ch
	uniques := <-ch

	finalDurationResult := <-durationsResult
	finalScheduledResult := <-scheduledResult

	expected := int64(config.EventFanout) * emitted * int64(config.DagSteps)

	// NOTE: `emit()` returns successfully pushed events (not merely generated IDs),
	// so `emitted` here is effectively "pushed".
	log.Printf(
		"ℹ️ pushed %d, executed %d, uniques %d, using %d events/s (fanout=%d dagSteps=%d expected=%d)",
		emitted,
		executed,
		uniques,
		config.Events,
		config.EventFanout,
		config.DagSteps,
		expected,
	)

	if executed == 0 {
		return fmt.Errorf("❌ no events executed")
	}

	log.Printf("ℹ️ final average duration per executed event: %s", finalDurationResult.avg)
	log.Printf("ℹ️ final average scheduling time per event: %s", finalScheduledResult.avg)
	if config.PlotDir != "" {
		log.Printf("ℹ️ exporting scheduling/duration snapshot data")
		err := finalScheduledResult.latencyResult.PlotLatency(filepath.Join(config.PlotDir, "scheduling_latency.html"))
		if err != nil {
			return err
		}
		err = finalDurationResult.latencyResult.PlotLatency(filepath.Join(config.PlotDir, "duration_latency.html"))
		if err != nil {
			return err
		}
	}
	if expected != executed {
		log.Printf("⚠️ warning: pushed and executed counts do not match: expected=%d got=%d", expected, executed)
	}

	if expected != uniques {
		return fmt.Errorf("❌ pushed and unique executed counts do not match: expected=%d got=%d (fanout=%d pushed=%d dagSteps=%d)", expected, uniques, config.EventFanout, emitted, config.DagSteps)
	}

	// Add a small tolerance (1% or 1ms, whichever is smaller)
	tolerance := config.AverageDurationThreshold / 100 // 1% tolerance
	if tolerance > time.Millisecond {
		tolerance = time.Millisecond
	}
	thresholdWithTolerance := config.AverageDurationThreshold + tolerance

	if finalDurationResult.avg > thresholdWithTolerance {
		return fmt.Errorf("❌ average duration per executed event is greater than the threshold (with tolerance): %s > %s (threshold: %s, tolerance: %s)", finalDurationResult.avg, thresholdWithTolerance, config.AverageDurationThreshold, tolerance)
	}

	log.Printf("✅ success")

	return nil
}
