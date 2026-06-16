package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/vicanso/go-charts/v2"
)

type LatencySnapshot struct {
	t       time.Time
	latency time.Duration
}

type LatencyResult struct {
	snapshots []LatencySnapshot
}

func (lr *LatencyResult) GeneratePlot(plotPath string, plotName string) error {
	bytes, err := lr.PlotBytes(plotName)
	if err != nil {
		return err
	}

	// save to file
	f, err := os.Create(filepath.Join(plotPath, fmt.Sprintf("%s_plot.png", plotName)))
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(bytes)
	return err
}

func (lr *LatencyResult) PlotBytes(plotName string) ([]byte, error) {
	if len(lr.snapshots) == 0 {
		return nil, fmt.Errorf("no snapshots available")
	}

	xvals := make([]string, 0, len(lr.snapshots))
	yvals := make([]float64, 0, len(lr.snapshots))

	start := lr.snapshots[0].t

	for _, s := range lr.snapshots {
		elapsed := s.t.Sub(start).Seconds()
		xvals = append(xvals, fmt.Sprintf("%.2f", elapsed))

		latencyMs := float64(s.latency.Microseconds()) / 1000.0
		yvals = append(yvals, latencyMs)
	}

	p, err := charts.LineRender(
		[][]float64{yvals},
		charts.TitleTextOptionFunc(fmt.Sprintf("Task %s (ms)", plotName)),
		charts.XAxisDataOptionFunc(xvals),
		charts.LegendLabelsOptionFunc([]string{"Latency"}),
		charts.HeightOptionFunc(500),
		charts.WidthOptionFunc(1000),
	)
	if err != nil {
		return nil, err
	}
	return p.Bytes()
}

type avgResult struct {
	count         int64
	avg           time.Duration
	latencyResult LatencyResult
}

type loadTestTiming struct {
	registrationTimeout time.Duration
	startupDelay        time.Duration
	safetyBuffer        time.Duration
	safetyTimeout       time.Duration
	activeWindow        time.Duration
}

type runResult struct {
	executed int64
	uniques  int64
}

func calculateLoadTestTiming(config LoadTestConfig) loadTestTiming {
	registrationTimeout := config.RegistrationTimeout
	if registrationTimeout == 0 {
		registrationTimeout = 60 * time.Second
	}

	startupDelay := 10 * time.Second
	safetyBuffer := 30 * time.Second

	return loadTestTiming{
		registrationTimeout: registrationTimeout,
		startupDelay:        startupDelay,
		safetyBuffer:        safetyBuffer,
		safetyTimeout:       registrationTimeout + config.WorkerDelay + startupDelay + config.Duration + config.Wait + safetyBuffer,
		activeWindow:        startupDelay + config.Duration + config.Wait,
	}
}

func waitOrDone(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func do(config LoadTestConfig) error {
	l.Info().Msgf("testing with duration=%s, eventsPerSecond=%d, delay=%s, wait=%s, concurrency=%d, averageDurationThreshold=%s", config.Duration, config.Events, config.Delay, config.Wait, config.Concurrency, config.AverageDurationThreshold)

	timing := calculateLoadTestTiming(config)

	safetyCtx, safetyCancel := context.WithTimeout(context.Background(), timing.safetyTimeout)
	defer safetyCancel()

	workerCtx, stopWorker := context.WithCancel(safetyCtx)
	defer stopWorker()

	ch := make(chan runResult, 1)
	durations := make(chan executionEvent, config.Events)

	// Compute running average for executed durations using a rolling average.
	durationsResult := make(chan avgResult, 1)
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

	registered := make(chan error, 1)

	go func() {
		count, uniques := run(workerCtx, config, durations, registered)
		close(durations)
		ch <- runResult{executed: count, uniques: uniques}
	}()

	if err := waitForRegistration(registered, timing.registrationTimeout); err != nil {
		return fmt.Errorf("❌ workflow registration failed within %s — engine must accept PutWorkflow on the current (pre-migration) schema: %w", timing.registrationTimeout, err)
	}

	if err := waitOrDone(safetyCtx, timing.startupDelay); err != nil {
		return fmt.Errorf("LOADTEST_SAFETY_TIMEOUT: safety deadline reached during startup delay: activeWindow=%s safetyTimeout=%s: %w", timing.activeWindow, timing.safetyTimeout, err)
	}

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

	emitted := emit(safetyCtx, config.Namespace, config.Events, config.Duration, scheduled, config.PayloadSize)
	close(scheduled)

	if err := safetyCtx.Err(); err != nil {
		return fmt.Errorf("LOADTEST_SAFETY_TIMEOUT: safety deadline reached while emitting events: pushed=%d activeWindow=%s safetyTimeout=%s: %w", emitted, timing.activeWindow, timing.safetyTimeout, err)
	}

	if err := waitOrDone(safetyCtx, config.Wait); err != nil {
		return fmt.Errorf("LOADTEST_SAFETY_TIMEOUT: safety deadline reached while waiting for executions: pushed=%d wait=%s activeWindow=%s safetyTimeout=%s: %w", emitted, config.Wait, timing.activeWindow, timing.safetyTimeout, err)
	}

	stopWorker()

	var result runResult
	select {
	case result = <-ch:
	case <-safetyCtx.Done():
		return fmt.Errorf("LOADTEST_SAFETY_TIMEOUT: worker did not stop before safety deadline: pushed=%d activeWindow=%s safetyTimeout=%s: %w", emitted, timing.activeWindow, timing.safetyTimeout, safetyCtx.Err())
	}

	executed := result.executed
	uniques := result.uniques

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
	if ShouldSendSlack() {
		log.Printf("ℹ️ sending scheduling/duration plots to Slack")
		slackSender := NewSlackSender("hatchet-staging-loadtest-us-west-2")
		durationBytes, err := finalDurationResult.latencyResult.PlotBytes("duration")
		if err != nil {
			log.Printf("❌ failed to generate duration plot: %v ", err)
		}
		schedulingBytes, err := finalScheduledResult.latencyResult.PlotBytes("scheduling")
		if err != nil {
			log.Printf("❌ failed to generate scheduling plot: %v ", err)
		}
		err = slackSender.Send(durationBytes, schedulingBytes, finalDurationResult.avg, finalScheduledResult.avg)
		if err != nil {
			log.Printf("❌ failed to send duration plots to slack: %v ", err)
		}
		log.Printf("ℹ️ scheduling/duration successfully plots to Slack")
	} else {
		log.Printf("ℹ️ not all environment vars for sending plots to Slack enabled...skipping")
	}
	if config.PlotDir != "" {
		log.Printf("ℹ️ exporting scheduling/duration snapshot data")
		err := finalScheduledResult.latencyResult.GeneratePlot(config.PlotDir, "scheduling")
		if err != nil {
			return err
		}
		err = finalDurationResult.latencyResult.GeneratePlot(config.PlotDir, "duration")
		if err != nil {
			return err
		}
		log.Printf("ℹ️ exported scheduling/duration snapshot data")
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
		return fmt.Errorf(
			"LOADTEST_THRESHOLD_EXCEEDED: average duration per executed event exceeded threshold: observed=%s thresholdWithTolerance=%s threshold=%s tolerance=%s dagSteps=%d eventFanout=%d pushed=%d executed=%d uniques=%d eventsPerSecond=%d activeWindow=%s safetyTimeout=%s",
			finalDurationResult.avg,
			thresholdWithTolerance,
			config.AverageDurationThreshold,
			tolerance,
			config.DagSteps,
			config.EventFanout,
			emitted,
			executed,
			uniques,
			config.Events,
			timing.activeWindow,
			timing.safetyTimeout,
		)
	}

	log.Printf("✅ success")

	return nil
}
