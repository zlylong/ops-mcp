package linux

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalAdapterReadsProcFixtures(t *testing.T) {
	root := t.TempDir()
	proc := filepath.Join(root, "proc")
	etc := filepath.Join(root, "etc")
	require.NoError(t, os.MkdirAll(filepath.Join(proc, "sys/kernel"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(proc, "1"), 0o755))
	require.NoError(t, os.MkdirAll(etc, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(proc, "sys/kernel/hostname"), []byte("ops-host\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(proc, "sys/kernel/osrelease"), []byte("6.12.test\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(proc, "uptime"), []byte("3600.00 1.00\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(proc, "loadavg"), []byte("0.10 0.20 0.30 1/2 3\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(proc, "cpuinfo"), []byte("processor\t: 0\nprocessor\t: 1\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(proc, "meminfo"), []byte("MemTotal:       2048000 kB\nMemFree:         512000 kB\nMemAvailable:   1024000 kB\nSwapTotal:       256000 kB\nSwapFree:        128000 kB\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(proc, "1/cgroup"), []byte("0::/init.scope\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(etc, "os-release"), []byte("PRETTY_NAME=\"Test Linux\"\n"), 0o644))

	adapter := &LocalAdapter{ProcRoot: proc, EtcRoot: etc, Now: func() time.Time { return time.Unix(7200, 0) }}

	sys, err := adapter.SystemInfo(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, "ops-host", sys["hostname"])
	assert.Equal(t, "6.12.test", sys["kernel"])
	assert.Equal(t, "Test Linux", sys["distribution"])
	assert.Equal(t, int64(3600), sys["uptimeSeconds"])
	assert.Equal(t, "local", sys["source"])

	load, err := adapter.LoadAverage(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, 0.10, load["load1"])
	assert.Equal(t, 2, load["cpuCores"])

	mem, err := adapter.MemoryUsage(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, uint64(2000), mem["totalMiB"])
	assert.Equal(t, uint64(1000), mem["availableMiB"])
	assert.Equal(t, 50.0, mem["usedPercent"])
}

func TestLocalAdapterProcessListFixture(t *testing.T) {
	root := t.TempDir()
	proc := filepath.Join(root, "proc")
	require.NoError(t, os.MkdirAll(filepath.Join(proc, "123"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(proc, "meminfo"), []byte("MemTotal:       1000 kB\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(proc, "123/status"), []byte("Name:\ttestproc\nUid:\t1000\t1000\t1000\t1000\nVmRSS:\t100 kB\n"), 0o644))

	adapter := &LocalAdapter{ProcRoot: proc, EtcRoot: root, Now: time.Now}
	result, err := adapter.ProcessList(context.Background(), map[string]any{"limit": 1})
	require.NoError(t, err)
	processes := result["processes"].([]map[string]any)
	require.Len(t, processes, 1)
	assert.Equal(t, 123, processes[0]["pid"])
	assert.Equal(t, "testproc", processes[0]["command"])
	assert.Equal(t, 10.0, processes[0]["memoryPercent"])
}

func TestParsePingSummary(t *testing.T) {
	result := parsePingSummary("4 packets transmitted, 4 received, 0% packet loss, time 3005ms\nrtt min/avg/max/mdev = 1.100/2.200/3.300/0.100 ms\n")
	assert.Equal(t, 0.0, result["packetLossPercent"])
	assert.Equal(t, 1.1, result["minRttMs"])
	assert.Equal(t, 2.2, result["avgRttMs"])
	assert.Equal(t, 3.3, result["maxRttMs"])
}
