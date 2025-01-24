package oommonitor

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"syscall"
	"time"

	"github.com/rs/zerolog"
)

type MonitorOpts struct {
	ThresholdBytes uint64
	Signal         string
	CheckInterval  time.Duration
	L              *zerolog.Logger
}

type Monitor struct {
	ThresholdBytes uint64
	Signal         os.Signal
	CheckInterval  time.Duration
	l              *zerolog.Logger
}

func NewMonitor(opts MonitorOpts) (*Monitor, error) {
	if opts.ThresholdBytes <= 0 {

		return nil, fmt.Errorf("ThresholdBytes must be greater than 0")
	}

	if opts.Signal == "" {
		return nil, fmt.Errorf("Signal must be specified")
	}
	if opts.CheckInterval <= 0 {
		opts.L.Warn().Msg("CheckInterval not specified, using default value of 10 seconds")
		opts.CheckInterval = 10 * time.Second
	}

	signal, err := getSignal(opts.Signal)

	if err != nil {
		return nil, err
	}

	return &Monitor{
		ThresholdBytes: opts.ThresholdBytes,
		Signal:         signal,
		CheckInterval:  opts.CheckInterval,
		l:              opts.L,
	}, nil
}

func getSignal(signal string) (os.Signal, error) {
	switch signal {
	case "SIGTERM":
		return syscall.SIGTERM, nil
	case "SIGKILL":
		return syscall.SIGKILL, nil
	case "SIGUSR1":
		return syscall.SIGUSR1, nil
	case "SIGUSR2":
		return syscall.SIGUSR2, nil
	case "SIGINT":
		return syscall.SIGINT, nil
	case "SIGHUP":
		return syscall.SIGHUP, nil
	default:
		return nil, fmt.Errorf("unsupported signal: %s please choose one of SIGTERM, SIGKILL, SIGUSR1, SIGUSR2, SIGINT, SIGHUP", signal)
	}
}

func (m *Monitor) StartMonitor(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				m.l.Info().Msg("OOM monitor stopped")
				return
			default:
				runtime.GC()
				var memStats runtime.MemStats
				runtime.ReadMemStats(&memStats)
				m.l.Info().Uint64("Alloc", memStats.Alloc).Uint64("Threshold", m.ThresholdBytes).Msg("Checking memory usage")

				if memStats.Alloc > m.ThresholdBytes {
					m.l.Warn().Uint64("Alloc", memStats.Alloc).Uint64("Threshold", m.ThresholdBytes).Msgf("Memory threshold exceeded, sending signal %s", m.Signal)
					pid := os.Getpid()
					process, err := os.FindProcess(pid)
					if err != nil {
						m.l.Error().Err(err).Msgf("Failed to find process for pid (%d)", pid)
					}
					err = process.Signal(m.Signal)
					if err != nil {
						m.l.Error().Err(err).Msg("Failed to send signal")
					}
				} else if memStats.Alloc > uint64(0.95*float64(m.ThresholdBytes)) && memStats.Alloc <= m.ThresholdBytes {
					m.l.Warn().Uint64("Alloc", memStats.Alloc).Uint64("Threshold", m.ThresholdBytes).Msg("Memory usage nearing threshold")
					m.logMemoryUsage()
				}

				time.Sleep(m.CheckInterval)
			}
		}
	}()
}

func (m *Monitor) logMemoryUsage() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	m.l.Warn().Msg("Detailed memory usage:")
	m.l.Warn().Uint64("Alloc", memStats.Alloc).Msg("Heap allocation in bytes")
	m.l.Warn().Uint64("TotalAlloc", memStats.TotalAlloc).Msg("Total bytes allocated")
	m.l.Warn().Uint64("Sys", memStats.Sys).Msg("System memory obtained by the program")
	m.l.Warn().Uint64("HeapIdle", memStats.HeapIdle).Msg("Bytes in idle (unused) spans")
	m.l.Warn().Uint64("HeapInuse", memStats.HeapInuse).Msg("Bytes in in-use spans")
	m.l.Warn().Uint64("HeapReleased", memStats.HeapReleased).Msg("Bytes of physical memory returned to the OS")
	m.l.Warn().Uint64("HeapObjects", memStats.HeapObjects).Msg("Number of allocated heap objects")
}

func (m *Monitor) Describe() string {
	return fmt.Sprintf("threshold %d bytes,  signal %s, checkInterval %s", m.ThresholdBytes, m.Signal, m.CheckInterval)
}
