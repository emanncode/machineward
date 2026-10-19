package metrics

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shirou/gopsutil/v4/process"
)

type fakeProcess struct {
	pid        int32
	startedAt  int64
	createErr  error
	name       string
	nameErr    error
	memory     *process.MemoryInfoStat
	memoryErr  error
	cpuPercent float64
	cpuErr     error
}

func (p *fakeProcess) PID() int32 { return p.pid }

func (p *fakeProcess) CreateTimeWithContext(context.Context) (int64, error) {
	return p.startedAt, p.createErr
}

func (p *fakeProcess) NameWithContext(context.Context) (string, error) {
	return p.name, p.nameErr
}

func (p *fakeProcess) MemoryInfoWithContext(context.Context) (*process.MemoryInfoStat, error) {
	return p.memory, p.memoryErr
}

func (p *fakeProcess) PercentWithContext(context.Context, time.Duration) (float64, error) {
	return p.cpuPercent, p.cpuErr
}

func TestCollectProcessListCountsInspectionErrorsButNotExitedPIDs(t *testing.T) {
	available := &fakeProcess{pid: 3}
	readers, skipped, err := collectProcessList(context.Background(), []int32{1, 2, 3}, func(_ context.Context, pid int32) (processReader, error) {
		switch pid {
		case 1:
			return nil, process.ErrorProcessNotRunning
		case 2:
			return nil, process.ErrorNotPermitted
		default:
			return available, nil
		}
	})
	if err != nil {
		t.Fatalf("collectProcessList returned error: %v", err)
	}
	if skipped != 1 {
		t.Fatalf("skipped = %d, want 1", skipped)
	}
	if len(readers) != 1 || readers[0].PID() != 3 {
		t.Fatalf("readers = %#v, want only PID 3", readers)
	}
}

func TestCollectProcessMetricsCountsUnavailableDataAndProtectsPIDReuse(t *testing.T) {
	old := &fakeProcess{pid: 10, startedAt: 100, name: "old", memory: &process.MemoryInfoStat{RSS: 100}, cpuPercent: 70}
	reusedPID := &fakeProcess{pid: 10, startedAt: 200, name: "new", memory: &process.MemoryInfoStat{RSS: 200}, cpuPercent: 0}
	inaccessible := &fakeProcess{pid: 11, startedAt: 300, nameErr: process.ErrorNotPermitted}
	exited := &fakeProcess{pid: 12, createErr: process.ErrorProcessNotRunning}

	rows, next, skipped, err := collectProcessMetrics(context.Background(), []processReader{reusedPID, inaccessible, exited}, map[int32]processSample{
		10: {handle: old, startedAt: 100},
	})
	if err != nil {
		t.Fatalf("collectProcessMetrics returned error: %v", err)
	}
	if skipped != 1 {
		t.Fatalf("skipped = %d, want 1", skipped)
	}
	if len(rows) != 1 || rows[0].PID != 10 || rows[0].Name != "new" || rows[0].CPUPercent != 0 {
		t.Fatalf("rows = %#v, want new PID 10 baseline row", rows)
	}
	if next[10].handle != reusedPID {
		t.Fatalf("PID 10 cache retained old process handle after start time changed")
	}
	if _, exists := next[12]; exists {
		t.Fatalf("exited PID 12 should not be cached")
	}
}

func TestCollectProcessMetricsKeepsHandleForCPUInterval(t *testing.T) {
	previous := &fakeProcess{pid: 21, startedAt: 400, name: "cached", memory: &process.MemoryInfoStat{RSS: 64}, cpuPercent: 12}
	current := &fakeProcess{pid: 21, startedAt: 400, name: "fresh", memory: &process.MemoryInfoStat{RSS: 128}, cpuPercent: 50}

	rows, next, skipped, err := collectProcessMetrics(context.Background(), []processReader{current}, map[int32]processSample{
		21: {handle: previous, startedAt: 400},
	})
	if err != nil {
		t.Fatalf("collectProcessMetrics returned error: %v", err)
	}
	if skipped != 0 || len(rows) != 1 {
		t.Fatalf("skipped = %d, rows = %d, want 0 and 1", skipped, len(rows))
	}
	if rows[0].Name != "cached" || rows[0].CPUPercent != 12 || next[21].handle != previous {
		t.Fatalf("collector did not preserve the process handle and its CPU baseline: row=%#v", rows[0])
	}
}

func TestCollectProcessMetricsSkipsCPUReadFailure(t *testing.T) {
	reader := &fakeProcess{
		pid:       31,
		startedAt: 500,
		name:      "busy-process",
		memory:    &process.MemoryInfoStat{RSS: 128},
		cpuErr:    process.ErrorNotPermitted,
	}

	rows, _, skipped, err := collectProcessMetrics(context.Background(), []processReader{reader}, nil)
	if err != nil {
		t.Fatalf("collectProcessMetrics returned error: %v", err)
	}
	if skipped != 1 || len(rows) != 0 {
		t.Fatalf("skipped = %d, rows = %d, want 1 and 0", skipped, len(rows))
	}
}

func TestCollectProcessMetricsPropagatesContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, _, err := collectProcessMetrics(ctx, []processReader{&fakeProcess{pid: 1}}, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}
