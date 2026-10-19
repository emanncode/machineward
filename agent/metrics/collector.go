package metrics

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/emanncode/machineward/protocol"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/process"
)

const maxProcesses = 25

// Collector keeps process handles between samples so CPU percentages can be
// calculated as deltas without blocking once per process.
type Collector struct {
	// Collect must be called serially. The agent owns one collector, and
	// gopsutil keeps host CPU delta baselines in package-global state.
	processes map[int32]processSample
}

type processSample struct {
	handle    processReader
	startedAt int64
}

type processReader interface {
	PID() int32
	CreateTimeWithContext(context.Context) (int64, error)
	NameWithContext(context.Context) (string, error)
	MemoryInfoWithContext(context.Context) (*process.MemoryInfoStat, error)
	PercentWithContext(context.Context, time.Duration) (float64, error)
}

type gopsutilProcess struct {
	*process.Process
}

func (p gopsutilProcess) PID() int32 { return p.Pid }

func NewCollector() *Collector {
	return &Collector{processes: make(map[int32]processSample)}
}

// listProcesses enumerates PIDs directly so failures to inspect an individual
// process are observable instead of being silently discarded by gopsutil's
// ProcessesWithContext helper.
func listProcesses(ctx context.Context) ([]processReader, uint32, error) {
	pids, err := process.PidsWithContext(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("list process IDs: %w", err)
	}

	return collectProcessList(ctx, pids, func(ctx context.Context, pid int32) (processReader, error) {
		current, err := process.NewProcessWithContext(ctx, pid)
		if err != nil {
			return nil, err
		}
		return gopsutilProcess{Process: current}, nil
	})
}

type processFactory func(context.Context, int32) (processReader, error)

func collectProcessList(ctx context.Context, pids []int32, create processFactory) ([]processReader, uint32, error) {
	readers := make([]processReader, 0, len(pids))
	var skipped uint32
	for _, pid := range pids {
		if err := ctx.Err(); err != nil {
			return nil, skipped, err
		}

		current, err := create(ctx, pid)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return nil, skipped, ctxErr
			}
			if !processExited(err) {
				skipped++
			}
			continue
		}
		readers = append(readers, current)
	}

	return readers, skipped, nil
}

func processExited(err error) bool {
	return errors.Is(err, process.ErrorProcessNotRunning) || errors.Is(err, os.ErrNotExist)
}

func (c *Collector) Collect(ctx context.Context) (protocol.MetricsSnapshot, error) {
	totalCPU, err := cpu.PercentWithContext(ctx, 0, false)
	if err != nil {
		return protocol.MetricsSnapshot{}, fmt.Errorf("collect total CPU: %w", err)
	}
	perCoreCPU, err := cpu.PercentWithContext(ctx, 0, true)
	if err != nil {
		return protocol.MetricsSnapshot{}, fmt.Errorf("collect per-core CPU: %w", err)
	}
	if len(totalCPU) == 0 {
		return protocol.MetricsSnapshot{}, fmt.Errorf("collect total CPU: no readings returned")
	}

	virtualMemory, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return protocol.MetricsSnapshot{}, fmt.Errorf("collect memory: %w", err)
	}
	swapMemory, err := mem.SwapMemoryWithContext(ctx)
	if err != nil {
		return protocol.MetricsSnapshot{}, fmt.Errorf("collect swap: %w", err)
	}
	processes, skipped, err := listProcesses(ctx)
	if err != nil {
		return protocol.MetricsSnapshot{}, err
	}

	processMetrics, currentProcesses, processSkipped, err := collectProcessMetrics(ctx, processes, c.processes)
	if err != nil {
		return protocol.MetricsSnapshot{}, err
	}
	skipped += processSkipped
	c.processes = currentProcesses

	sort.Slice(processMetrics, func(i, j int) bool {
		if processMetrics[i].CPUPercent == processMetrics[j].CPUPercent {
			return processMetrics[i].MemoryBytes > processMetrics[j].MemoryBytes
		}
		return processMetrics[i].CPUPercent > processMetrics[j].CPUPercent
	})
	if len(processMetrics) > maxProcesses {
		processMetrics = processMetrics[:maxProcesses]
	}

	return protocol.MetricsSnapshot{
		CollectedAt: time.Now().UTC(),
		CPU: protocol.CPU{
			TotalPercent:   totalCPU[0],
			PerCorePercent: perCoreCPU,
		},
		Memory: protocol.Memory{
			TotalBytes:  virtualMemory.Total,
			UsedBytes:   virtualMemory.Used,
			UsedPercent: virtualMemory.UsedPercent,
		},
		Swap: protocol.Memory{
			TotalBytes:  swapMemory.Total,
			UsedBytes:   swapMemory.Used,
			UsedPercent: swapMemory.UsedPercent,
		},
		Processes:        processMetrics,
		ProcessesSkipped: skipped,
	}, nil
}

func collectProcessMetrics(ctx context.Context, processes []processReader, previous map[int32]processSample) ([]protocol.ProcessMetric, map[int32]processSample, uint32, error) {
	currentProcesses := make(map[int32]processSample, len(processes))
	processMetrics := make([]protocol.ProcessMetric, 0, len(processes))
	var skipped uint32

	for _, current := range processes {
		if err := ctx.Err(); err != nil {
			return nil, nil, skipped, err
		}

		startedAt, err := current.CreateTimeWithContext(ctx)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return nil, nil, skipped, ctxErr
			}
			if !processExited(err) {
				skipped++
			}
			continue
		}

		pid := current.PID()
		sample, exists := previous[pid]
		if !exists || sample.startedAt != startedAt {
			sample = processSample{handle: current, startedAt: startedAt}
		}
		currentProcesses[pid] = sample

		name, err := sample.handle.NameWithContext(ctx)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return nil, nil, skipped, ctxErr
			}
			if !processExited(err) {
				skipped++
			}
			continue
		}
		memory, err := sample.handle.MemoryInfoWithContext(ctx)
		if err != nil || memory == nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return nil, nil, skipped, ctxErr
			}
			if err == nil || !processExited(err) {
				skipped++
			}
			continue
		}
		cpuPercent, err := sample.handle.PercentWithContext(ctx, 0)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return nil, nil, skipped, ctxErr
			}
			if !processExited(err) {
				skipped++
			}
			continue
		}
		if cpuPercent < 0 {
			cpuPercent = 0
		}

		processMetrics = append(processMetrics, protocol.ProcessMetric{
			PID:             pid,
			Name:            name,
			CPUPercent:      cpuPercent,
			MemoryBytes:     memory.RSS,
			StartedAtUnixMS: startedAt,
		})
	}
	return processMetrics, currentProcesses, skipped, nil
}
