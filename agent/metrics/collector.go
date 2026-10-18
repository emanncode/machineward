package metrics

import (
	"context"
	"fmt"
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
	processes map[int32]processSample
}

type processSample struct {
	handle    *process.Process
	startedAt int64
}

func NewCollector() *Collector {
	return &Collector{processes: make(map[int32]processSample)}
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
	processes, err := process.ProcessesWithContext(ctx)
	if err != nil {
		return protocol.MetricsSnapshot{}, fmt.Errorf("list processes: %w", err)
	}

	currentProcesses := make(map[int32]processSample, len(processes))
	processMetrics := make([]protocol.ProcessMetric, 0, len(processes))
	for _, current := range processes {
		if err := ctx.Err(); err != nil {
			return protocol.MetricsSnapshot{}, err
		}

		startedAt, err := current.CreateTimeWithContext(ctx)
		if err != nil {
			continue // The process may have exited during collection.
		}

		sample, exists := c.processes[current.Pid]
		if !exists || sample.startedAt != startedAt {
			sample = processSample{handle: current, startedAt: startedAt}
		}
		currentProcesses[current.Pid] = sample

		name, err := sample.handle.NameWithContext(ctx)
		if err != nil {
			continue
		}
		memory, err := sample.handle.MemoryInfoWithContext(ctx)
		if err != nil || memory == nil {
			continue
		}
		cpuPercent, err := sample.handle.PercentWithContext(ctx, 0)
		if err != nil {
			cpuPercent = 0
		}
		if cpuPercent < 0 {
			cpuPercent = 0
		}

		processMetrics = append(processMetrics, protocol.ProcessMetric{
			PID:             current.Pid,
			Name:            name,
			CPUPercent:      cpuPercent,
			MemoryBytes:     memory.RSS,
			StartedAtUnixMS: startedAt,
		})
	}
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
		Processes: processMetrics,
	}, nil
}
