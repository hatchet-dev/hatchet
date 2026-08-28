package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vicanso/go-charts/v2"

	"github.com/hatchet-dev/hatchet/pkg/loadtest/eventkeys"
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
		names = append(names, applyNamespace(eventkeys.WorkflowStandardName(i), namespace))
	}
	return names
}

func workflowNamesForKey(key eventkeys.EventKey, namespace string, fanout int) []string {
	switch key {
	case eventkeys.EventKeyDefault:
		return expectedWorkflowNames(namespace, fanout)
	case eventkeys.EventKeyBatch:
		return []string{applyNamespace(eventkeys.WorkflowBatchName, namespace)}
	case eventkeys.EventKeyDurable:
		return []string{applyNamespace(eventkeys.WorkflowDurableName, namespace)}
	case eventkeys.EventKeyDag:
		return []string{applyNamespace(eventkeys.WorkflowDagName, namespace)}
	default:
		return nil
	}
}

const dagWorkflowSteps = 2

func executionsPerPush(key eventkeys.EventKey, fanout, dagSteps int) int64 {
	switch key {
	case eventkeys.EventKeyDefault:
		return int64(fanout) * int64(dagSteps)
	case eventkeys.EventKeyDag:
		return dagWorkflowSteps
	default:
		return 1
	}
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

type phasesByKey struct {
	byKey   map[eventkeys.EventKey]phaseAccumulator
	overall phaseAccumulator
}

type resolvedWorkflowSet struct {
	ids  []uuid.UUID
	keys map[uuid.UUID]eventkeys.EventKey
}

type phaseAcc struct {
	qCount, sCount, eCount int64
	qAvg, sAvg, eAvg       time.Duration
	qSnaps, sSnaps, eSnaps []LatencySnapshot
}

func (p *phaseAcc) add(s PhaseSample) {
	now := time.Now()

	p.qCount++
	p.qAvg += (s.Queued - p.qAvg) / time.Duration(p.qCount)
	p.qSnaps = append(p.qSnaps, LatencySnapshot{t: now, latency: s.Queued})

	p.sCount++
	p.sAvg += (s.Scheduling - p.sAvg) / time.Duration(p.sCount)
	p.sSnaps = append(p.sSnaps, LatencySnapshot{t: now, latency: s.Scheduling})

	p.eCount++
	p.eAvg += (s.Execution - p.eAvg) / time.Duration(p.eCount)
	p.eSnaps = append(p.eSnaps, LatencySnapshot{t: now, latency: s.Execution})
}

func (p *phaseAcc) result() phaseAccumulator {
	return phaseAccumulator{
		queued:     avgResult{count: p.qCount, avg: p.qAvg, latencyResult: LatencyResult{snapshots: p.qSnaps}},
		scheduling: avgResult{count: p.sCount, avg: p.sAvg, latencyResult: LatencyResult{snapshots: p.sSnaps}},
		execution:  avgResult{count: p.eCount, avg: p.eAvg, latencyResult: LatencyResult{snapshots: p.eSnaps}},
	}
}

func accumulatePhases(samples <-chan PhaseSample) <-chan phasesByKey {
	out := make(chan phasesByKey, 1)

	go func() {
		overall := &phaseAcc{}
		byKey := make(map[eventkeys.EventKey]*phaseAcc)

		for s := range samples {
			overall.add(s)

			a := byKey[s.EventKey]
			if a == nil {
				a = &phaseAcc{}
				byKey[s.EventKey] = a
			}
			a.add(s)
		}

		res := phasesByKey{
			byKey:   make(map[eventkeys.EventKey]phaseAccumulator, len(byKey)),
			overall: overall.result(),
		}
		for k, a := range byKey {
			res.byKey[k] = a.result()
		}

		out <- res
	}()

	return out
}

func do(config LoadTestConfig) error {
	if len(config.EventKeys) == 0 {
		config.EventKeys = []eventkeys.EventKey{eventkeys.EventKeyDefault}
	}

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
	resolvedWorkflows := make(chan resolvedWorkflowSet, 1)

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

			var names []string
			nameToKey := make(map[string]eventkeys.EventKey)
			for _, key := range config.EventKeys {
				for _, n := range workflowNamesForKey(key, hc.V0().Namespace(), config.EventFanout) {
					names = append(names, n)
					nameToKey[n] = key
				}
			}

			l.Info().Msgf("externalWorker: resolving workflow(s) %v (make sure a separately-running SDK worker, e.g. cmd/hatchet-loadtest/go, is already up and has registered them)...", names)

			ids, err := ResolveWorkflowIDs(ctx, hc.V0().API(), uuid.MustParse(hc.V0().TenantId()), names, registrationTimeout)
			if err != nil {
				registered <- fmt.Errorf("externalWorker: %w", err)
				return
			}

			l.Info().Msgf("externalWorker: resolved workflow(s) %v to ids %v", names, ids)

			workflowKeys := make(map[uuid.UUID]eventkeys.EventKey, len(ids))
			for i, id := range ids {
				workflowKeys[id] = nameToKey[names[i]]
			}

			resolvedWorkflows <- resolvedWorkflowSet{ids: ids, keys: workflowKeys}
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

	scheduled := make(chan scheduledSample, config.Events)

	scheduledResult := make(chan map[eventkeys.EventKey]avgResult, 1)
	go func() {
		type acc struct {
			count     int64
			avg       time.Duration
			snapshots []LatencySnapshot
		}
		accs := make(map[eventkeys.EventKey]*acc)
		for s := range scheduled {
			a := accs[s.eventKey]
			if a == nil {
				a = &acc{}
				accs[s.eventKey] = a
			}
			a.count++
			if a.count == 1 {
				a.avg = s.latency
			} else {
				a.avg += (s.latency - a.avg) / time.Duration(a.count)
			}
			a.snapshots = append(a.snapshots, LatencySnapshot{t: time.Now(), latency: s.latency})
		}
		out := make(map[eventkeys.EventKey]avgResult, len(accs))
		for k, a := range accs {
			out[k] = avgResult{count: a.count, avg: a.avg, latencyResult: LatencyResult{snapshots: a.snapshots}}
		}
		scheduledResult <- out
	}()

	// externalWorker mode: start sweeping the engine's own timing data for
	// the resolved workflow(s), in parallel with emission below.
	var phaseSamples chan PhaseSample
	var phaseResultCh <-chan phasesByKey
	var cancelTiming context.CancelFunc
	var collector *TimingCollector

	if config.ExternalWorker {
		resolved := <-resolvedWorkflows

		// new context so that the timing drain is independent.
		var timingCtx context.Context
		timingCtx, cancelTiming = context.WithCancel(context.Background())
		defer cancelTiming() // safe to call more than once; guards every return path below

		collector = NewTimingCollector(timingClient, resolved.ids, resolved.keys, timingPollInterval, config.TimingSampleRate)

		phaseSamples = make(chan PhaseSample, 256)
		phaseResultCh = accumulatePhases(phaseSamples)

		go func() {
			collector.Run(timingCtx, phaseSamples)
			close(phaseSamples)
		}()
	}

	pushedByKey := emit(ctx, config.Namespace, config.Events, config.Duration, scheduled, config.PayloadSize, config.EmitWorkers, config.EventKeys)
	close(scheduled)

	pbkj, err := json.MarshalIndent(pushedByKey, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal pushedByKey: %w", err)
	}

	log.Printf("ℹ️ pushed per event key: %v", string(pbkj))

	emitted := pushedByKey[eventkeys.EventKeyDefault]

	executed := <-ch
	uniques := <-ch

	finalDurationResult := <-durationsResult
	finalScheduledByKey := <-scheduledResult
	finalScheduledResult := finalScheduledByKey[eventkeys.EventKeyDefault]

	var phases phasesByKey
	if config.ExternalWorker {
		// Give the engine config.Wait to finish runs that were still in flight
		// when emission stopped.
		if config.Wait > 0 {
			l.Info().Msgf("externalWorker: waiting %s for in-flight runs to complete before stopping timing collection...", config.Wait)
			time.Sleep(config.Wait)
		}

		// keep the collector running to make sure we don't miss any runs on the tail end
		l.Info().Msg("externalWorker: waiting for timing collector to fetch all discovered runs...")
		logTicker := time.NewTicker(5 * time.Second)
		var zeroSince time.Time
		for {
			pending := collector.Pending()

			if pending == 0 {
				if zeroSince.IsZero() {
					zeroSince = time.Now()
				} else if time.Since(zeroSince) >= 2*timingPollInterval {
					break
				}
			} else {
				zeroSince = time.Time{}
			}

			select {
			case <-logTicker.C:
				l.Info().Msgf("externalWorker: still waiting on %d run(s)...", pending)
			case <-time.After(200 * time.Millisecond):
			}
		}
		logTicker.Stop()

		cancelTiming()
		phases = <-phaseResultCh
	}

	benchPhases := phases.byKey[eventkeys.EventKeyDefault]
	standardSelected := slices.Contains(config.EventKeys, eventkeys.EventKeyDefault)

	expected := int64(config.EventFanout) * emitted * int64(config.DagSteps)

	if config.ExternalWorker {
		sampleRate := config.TimingSampleRate
		if sampleRate <= 0 || sampleRate > 1 {
			sampleRate = 1
		}

		var expectedAllKeys int64
		for _, k := range config.EventKeys {
			expectedAllKeys += pushedByKey[k] * executionsPerPush(k, config.EventFanout, config.DagSteps)
		}
		expectedSampled := int64(float64(expectedAllKeys) * sampleRate)

		for _, k := range config.EventKeys {
			p := phases.byKey[k]
			log.Printf(
				"ℹ️ pushed %d %q events, using %d events/s (externalWorker: engine-observed samples, %.0f%% sampled — queued n=%d, scheduling n=%d, execution n=%d)",
				pushedByKey[k], k, config.Events, sampleRate*100, p.queued.count, p.scheduling.count, p.execution.count,
			)
		}

		if phases.overall.execution.count == 0 {
			return fmt.Errorf("❌ no timing samples observed - check that the external SDK worker actually executed tasks for workflow(s) %v", expectedWorkflowNames(timingClient.V0().Namespace(), config.EventFanout))
		}

		if expectedSampled > 0 {
			lower, upper := expectedSampled/2, expectedSampled+expectedSampled/2
			if phases.overall.execution.count < lower || phases.overall.execution.count > upper {
				log.Printf("⚠️ warning: engine-observed sample count is well outside the expected range: expected≈%d (%.0f%% sample of %d pushed across %d key(s)) got=%d", expectedSampled, sampleRate*100, expectedAllKeys, len(config.EventKeys), phases.overall.execution.count)
			}
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
		// The engine-observed phase results are what we report on; there's no
		// in-process step handler feeding client-side duration/scheduling here.
		log.Printf("ℹ️ overall engine timing (n=%d):", phases.overall.execution.count)
		log.Printf("ℹ️   final average queued time per event: %s", phases.overall.queued.avg)
		log.Printf("ℹ️   final average scheduling time per event: %s", phases.overall.scheduling.avg)
		log.Printf("ℹ️   final average duration per executed event: %s", phases.overall.execution.avg)

		for _, k := range config.EventKeys {
			p := phases.byKey[k]
			log.Printf("ℹ️ engine timing for %s (n=%d):", k, p.execution.count)
			log.Printf("ℹ️   queued=%s scheduling=%s execution=%s", p.queued.avg, p.scheduling.avg, p.execution.avg)
			if r, ok := finalScheduledByKey[k]; ok {
				log.Printf("ℹ️   scheduling (push) latency: avg=%s n=%d", r.avg, r.count)
			}
		}
	} else {
		log.Printf("ℹ️ final average duration per executed event: %s", finalDurationResult.avg)
		log.Printf("ℹ️ final average scheduling time per event: %s", finalScheduledResult.avg)
		for _, k := range config.EventKeys {
			if r, ok := finalScheduledByKey[k]; ok {
				log.Printf("ℹ️ scheduling (push) latency for %s: avg=%s n=%d", k, r.avg, r.count)
			}
		}
	}
	// In externalWorker mode, finalDurationResult/finalScheduledResult have no
	// snapshots (durations was closed with zero samples up front, and there's
	// no in-process step handler to feed scheduled) - use the engine-observed
	// phase results instead, same as the "final average" log lines above.
	durationForReport, schedulingForReport := finalDurationResult, finalScheduledResult
	if config.ExternalWorker {
		durationForReport, schedulingForReport = phases.overall.execution, phases.overall.scheduling
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
		for _, k := range config.EventKeys {
			r, ok := finalScheduledByKey[k]
			if !ok || len(r.latencyResult.snapshots) == 0 {
				continue
			}
			plotName := "scheduling-" + strings.ReplaceAll(k.String(), ":", "-")
			if err := r.latencyResult.GeneratePlot(config.PlotDir, plotName); err != nil {
				return err
			}
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
		if standardSelected {
			if benchPhases.execution.count == 0 {
				return fmt.Errorf("❌ no timing samples observed for %q - check that the external SDK worker actually executed tasks for workflow(s) %v", eventkeys.EventKeyDefault, expectedWorkflowNames(timingClient.V0().Namespace(), config.EventFanout))
			}

			if expected != benchPhases.execution.count {
				log.Printf("⚠️ warning: pushed and executed-timing-sample counts do not match: expected=%d got=%d", expected, benchPhases.execution.count)
			}

			if benchPhases.execution.avg > thresholdWithTolerance {
				return fmt.Errorf("❌ average execution time is greater than the threshold (with tolerance): %s > %s (threshold: %s, tolerance: %s)", benchPhases.execution.avg, thresholdWithTolerance, config.AverageDurationThreshold, tolerance)
			}
		} else if phases.overall.execution.count == 0 {
			return fmt.Errorf("❌ no timing samples observed for selected event key(s) %v - check that the external SDK worker executed the corresponding workflow(s)", eventkeys.Names(config.EventKeys))
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
