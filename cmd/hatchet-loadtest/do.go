package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/vicanso/go-charts/v2"

	v1 "github.com/hatchet-dev/hatchet/pkg/v1" //nolint:staticcheck // SA1019: used only for REST timing queries in --externalWorker mode
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

// expectedWorkflowNames returns the (namespaced) workflow names an external
// worker is expected to have registered, matching the exact naming scheme
// run.go itself would use for the same config (see run.go's EventFanout
// loop: "load-test-0", "load-test-1", ...).

func expectedWorkflowNames(namespace string, fanout int) []string {
	if fanout <= 0 {
		fanout = 1
	}

	names := make([]string, 0, fanout)
	for i := 0; i < fanout; i++ {
		names = append(names, applyNamespace(fmt.Sprintf("load-test-%d", i), namespace))
	}
	return names
}

// phaseAccumulator computes a simple running mean per phase from a stream of
// PhaseSample values - the externalWorker equivalent of the avgResult
// goroutines above, just fed from the engine's REST timing data instead of
// an in-process step handler.
type phaseAccumulator struct {
	queued     avgResult
	scheduling avgResult
	execution  avgResult
}

func accumulatePhases(samples <-chan PhaseSample) <-chan phaseAccumulator {
	out := make(chan phaseAccumulator, 1)

	go func() {
		var qCount, sCount, eCount int64
		var qAvg, sAvg, eAvg time.Duration
		var qSnaps, sSnaps, eSnaps []LatencySnapshot

		for s := range samples {
			now := time.Now()

			qCount++
			qAvg += (s.Queued - qAvg) / time.Duration(qCount)
			qSnaps = append(qSnaps, LatencySnapshot{t: now, latency: s.Queued})

			sCount++
			sAvg += (s.Scheduling - sAvg) / time.Duration(sCount)
			sSnaps = append(sSnaps, LatencySnapshot{t: now, latency: s.Scheduling})

			eCount++
			eAvg += (s.Execution - eAvg) / time.Duration(eCount)
			eSnaps = append(eSnaps, LatencySnapshot{t: now, latency: s.Execution})
		}

		out <- phaseAccumulator{
			queued:     avgResult{count: qCount, avg: qAvg, latencyResult: LatencyResult{snapshots: qSnaps}},
			scheduling: avgResult{count: sCount, avg: sAvg, latencyResult: LatencyResult{snapshots: sSnaps}},
			execution:  avgResult{count: eCount, avg: eAvg, latencyResult: LatencyResult{snapshots: eSnaps}},
		}
	}()

	return out
}

func do(config LoadTestConfig) error {
	l.Info().Msgf("testing with duration=%s, eventsPerSecond=%d, delay=%s, wait=%s, concurrency=%d, averageDurationThreshold=%s", config.Duration, config.Events, config.Delay, config.Wait, config.Concurrency, config.AverageDurationThreshold)

	after := 10 * time.Second
	registrationTimeout := config.RegistrationTimeout
	if registrationTimeout == 0 {
		registrationTimeout = 60 * time.Second
	}

	// The worker may intentionally be delayed (WorkerDelay) before it starts consuming tasks.
	// The test timeout must include registration and this delay, otherwise we can cancel while work is still expected to complete.
	timeout := registrationTimeout + config.WorkerDelay + after + config.Duration + config.Wait + 30*time.Second

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ch := make(chan int64, 2)
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

	// Only populated when config.ExternalWorker is set - see below.
	var timingClient v1.HatchetClient //nolint:staticcheck // SA1019
	resolvedWorkflowIDs := make(chan []uuid.UUID, 1)

	if config.ExternalWorker {
		close(durations)
		ch <- 0
		ch <- 0

		go func() {
			hc, err := v1.NewHatchetClient(v1.Config{Namespace: config.Namespace, Logger: &l}) //nolint:staticcheck // SA1019
			if err != nil {
				registered <- fmt.Errorf("externalWorker: error creating hatchet client: %w", err)
				return
			}
			timingClient = hc

			names := expectedWorkflowNames(hc.V0().Namespace(), config.EventFanout)

			l.Info().Msgf("externalWorker: resolving workflow(s) %v (make sure a separately-running SDK worker, e.g. cmd/hatchet-loadtest/go, is already up and has registered them)...", names)

			ids, err := ResolveWorkflowIDs(ctx, hc.V0().API(), uuid.MustParse(hc.V0().TenantId()), names, registrationTimeout)
			if err != nil {
				registered <- fmt.Errorf("externalWorker: %w", err)
				return
			}

			l.Info().Msgf("externalWorker: resolved workflow(s) %v to ids %v", names, ids)

			resolvedWorkflowIDs <- ids
			registered <- nil
		}()
	} else {
		go func() {
			count, uniques := run(ctx, config, durations, registered)
			close(durations)
			ch <- count
			ch <- uniques
		}()
	}

	if err := waitForRegistration(registered, registrationTimeout); err != nil {
		return fmt.Errorf("❌ workflow registration failed within %s — engine must accept PutWorkflow on the current (pre-migration) schema: %w", registrationTimeout, err)
	}

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

	// externalWorker mode: start sweeping the engine's own timing data for
	// the resolved workflow(s), in parallel with emission below.
	var phaseSamples chan PhaseSample
	var phaseResultCh <-chan phaseAccumulator
	var cancelTiming context.CancelFunc

	if config.ExternalWorker {
		workflowIDs := <-resolvedWorkflowIDs

		var timingCtx context.Context
		timingCtx, cancelTiming = context.WithCancel(ctx)
		defer cancelTiming() // safe to call more than once; guards every return path below

		collector := NewTimingCollector(timingClient, workflowIDs, 2*time.Second)

		phaseSamples = make(chan PhaseSample, 256)
		phaseResultCh = accumulatePhases(phaseSamples)

		go func() {
			collector.Run(timingCtx, phaseSamples)
			close(phaseSamples)
		}()
	}

	emitted := emit(ctx, config.Namespace, config.Events, config.Duration, scheduled, config.PayloadSize)
	close(scheduled)

	executed := <-ch
	uniques := <-ch

	finalDurationResult := <-durationsResult
	finalScheduledResult := <-scheduledResult

	var phases phaseAccumulator
	if config.ExternalWorker {
		cancelTiming()
		phases = <-phaseResultCh
	}

	expected := int64(config.EventFanout) * emitted * int64(config.DagSteps)

	if config.ExternalWorker {
		log.Printf(
			"ℹ️ pushed %d, using %d events/s (externalWorker: engine-observed samples — queued n=%d, scheduling n=%d, execution n=%d)",
			emitted, config.Events, phases.queued.count, phases.scheduling.count, phases.execution.count,
		)

		if phases.execution.count == 0 {
			return fmt.Errorf("❌ no timing samples observed - check that the external SDK worker actually executed tasks for workflow(s) %v", expectedWorkflowNames(timingClient.V0().Namespace(), config.EventFanout))
		}

		if expected != phases.execution.count {
			log.Printf("⚠️ warning: pushed and executed-timing-sample counts do not match: expected=%d got=%d", expected, phases.execution.count)
		}
	} else {
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
	}

	if config.ExternalWorker {
		// engine-observed timing replaces the client-side duration/scheduling
		// averages above (which are meaningless here - durations was closed
		// with zero samples and there's no in-process step handler), rather
		// than reporting alongside them.
		log.Printf("ℹ️ final average queued time per event: %s", phases.queued.avg)
		log.Printf("ℹ️ final average scheduling time per event: %s", phases.scheduling.avg)
		log.Printf("ℹ️ final average duration per executed event: %s", phases.execution.avg)
	} else {
		log.Printf("ℹ️ final average duration per executed event: %s", finalDurationResult.avg)
		log.Printf("ℹ️ final average scheduling time per event: %s", finalScheduledResult.avg)
	}
	// In externalWorker mode, finalDurationResult/finalScheduledResult have no
	// snapshots (durations was closed with zero samples up front, and there's
	// no in-process step handler to feed scheduled) - use the engine-observed
	// phase results instead, same as the "final average" log lines above.
	durationForReport, schedulingForReport := finalDurationResult, finalScheduledResult
	if config.ExternalWorker {
		durationForReport, schedulingForReport = phases.execution, phases.scheduling
	}

	if ShouldSendSlack() {
		log.Printf("ℹ️ sending scheduling/duration plots to Slack")
		slackSender := NewSlackSender("hatchet-staging-loadtest-us-west-2")
		durationBytes, err := durationForReport.latencyResult.PlotBytes("duration")
		if err != nil {
			log.Printf("❌ failed to generate duration plot: %v ", err)
		}
		schedulingBytes, err := schedulingForReport.latencyResult.PlotBytes("scheduling")
		if err != nil {
			log.Printf("❌ failed to generate scheduling plot: %v ", err)
		}
		err = slackSender.Send(durationBytes, schedulingBytes, durationForReport.avg, schedulingForReport.avg)
		if err != nil {
			log.Printf("❌ failed to send duration plots to slack: %v ", err)
		}
		log.Printf("ℹ️ scheduling/duration successfully plots to Slack")
	} else {
		log.Printf("ℹ️ not all environment vars for sending plots to Slack enabled...skipping")
	}
	if config.PlotDir != "" {
		log.Printf("ℹ️ exporting scheduling/duration snapshot data")
		err := schedulingForReport.latencyResult.GeneratePlot(config.PlotDir, "scheduling")
		if err != nil {
			return err
		}
		err = durationForReport.latencyResult.GeneratePlot(config.PlotDir, "duration")
		if err != nil {
			return err
		}
		log.Printf("ℹ️ exported scheduling/duration snapshot data")
	}

	// Add a small tolerance (1% or 1ms, whichever is smaller)
	tolerance := config.AverageDurationThreshold / 100 // 1% tolerance
	if tolerance > time.Millisecond {
		tolerance = time.Millisecond
	}
	thresholdWithTolerance := config.AverageDurationThreshold + tolerance

	if config.ExternalWorker {
		if phases.execution.avg > thresholdWithTolerance {
			return fmt.Errorf("❌ average execution time is greater than the threshold (with tolerance): %s > %s (threshold: %s, tolerance: %s)", phases.execution.avg, thresholdWithTolerance, config.AverageDurationThreshold, tolerance)
		}
	} else {
		if expected != executed {
			log.Printf("⚠️ warning: pushed and executed counts do not match: expected=%d got=%d", expected, executed)
		}

		if expected != uniques {
			return fmt.Errorf("❌ pushed and unique executed counts do not match: expected=%d got=%d (fanout=%d pushed=%d dagSteps=%d)", expected, uniques, config.EventFanout, emitted, config.DagSteps)
		}

		if finalDurationResult.avg > thresholdWithTolerance {
			return fmt.Errorf("❌ average duration per executed event is greater than the threshold (with tolerance): %s > %s (threshold: %s, tolerance: %s)", finalDurationResult.avg, thresholdWithTolerance, config.AverageDurationThreshold, tolerance)
		}
	}

	log.Printf("✅ success")

	return nil
}
