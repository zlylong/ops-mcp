package linux

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type LocalAdapter struct {
	ProcRoot string
	EtcRoot  string
	Now      func() time.Time
}

func NewLocalAdapter() *LocalAdapter {
	return &LocalAdapter{ProcRoot: firstExistingDir("/host/proc", "/proc"), EtcRoot: firstExistingDir("/host/etc", "/etc"), Now: time.Now}
}

func firstExistingDir(paths ...string) string {
	for _, path := range paths {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return path
		}
	}
	return paths[len(paths)-1]
}

func (a *LocalAdapter) SystemInfo(ctx context.Context, params map[string]any) (map[string]any, error) {
	uptime, _ := a.readUptime()
	bootTime := a.now().Add(-time.Duration(uptime) * time.Second).UTC().Format(time.RFC3339)
	return map[string]any{"hostname": strings.TrimSpace(a.readFirst("sys/kernel/hostname")), "kernel": strings.TrimSpace(a.readFirst("sys/kernel/osrelease")), "distribution": a.distribution(), "architecture": runtime.GOARCH, "uptimeSeconds": uptime, "bootTime": bootTime, "virtualization": a.virtualization(), "source": "local"}, nil
}

func (a *LocalAdapter) LoadAverage(ctx context.Context, params map[string]any) (map[string]any, error) {
	fields := strings.Fields(a.readFirst("loadavg"))
	if len(fields) < 3 {
		return nil, errors.New("invalid loadavg data")
	}
	load1, _ := strconv.ParseFloat(fields[0], 64)
	load5, _ := strconv.ParseFloat(fields[1], 64)
	load15, _ := strconv.ParseFloat(fields[2], 64)
	return map[string]any{"load1": load1, "load5": load5, "load15": load15, "cpuCores": a.cpuCores(), "source": "local"}, nil
}

func (a *LocalAdapter) MemoryUsage(ctx context.Context, params map[string]any) (map[string]any, error) {
	mem := a.meminfo()
	total := mem["MemTotal"] / 1024
	available := mem["MemAvailable"] / 1024
	free := mem["MemFree"] / 1024
	used := total - available
	usedPercent := 0.0
	if total > 0 {
		usedPercent = round1(float64(used) / float64(total) * 100)
	}
	return map[string]any{"totalMiB": total, "usedMiB": used, "freeMiB": free, "availableMiB": available, "usedPercent": usedPercent, "swapTotalMiB": mem["SwapTotal"] / 1024, "swapUsedMiB": (mem["SwapTotal"] - mem["SwapFree"]) / 1024, "source": "local"}, nil
}

func (a *LocalAdapter) DiskUsage(ctx context.Context, params map[string]any) (map[string]any, error) {
	path := stringParam(params, "path", "/")
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return nil, fmt.Errorf("statfs %q: %w", path, err)
	}
	total := float64(stat.Blocks*uint64(stat.Bsize)) / 1024 / 1024 / 1024
	available := float64(stat.Bavail*uint64(stat.Bsize)) / 1024 / 1024 / 1024
	used := total - available
	usedPercent := 0.0
	if total > 0 {
		usedPercent = used / total * 100
	}
	return map[string]any{"path": path, "filesystem": "local", "totalGiB": round1(total), "usedGiB": round1(used), "availableGiB": round1(available), "usedPercent": round1(usedPercent), "source": "local"}, nil
}

func (a *LocalAdapter) ProcessList(ctx context.Context, params map[string]any) (map[string]any, error) {
	limit := intParam(params, "limit", 10)
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	procs := a.processes()
	sort.Slice(procs, func(i, j int) bool { return procs[i].rssKiB > procs[j].rssKiB })
	if limit < len(procs) {
		procs = procs[:limit]
	}
	out := make([]map[string]any, 0, len(procs))
	for _, p := range procs {
		out = append(out, map[string]any{"pid": p.pid, "user": p.user, "cpuPercent": 0, "memoryPercent": p.memoryPercent, "command": p.command})
	}
	return map[string]any{"processes": out, "limit": limit, "source": "local"}, nil
}

func (a *LocalAdapter) NetworkInterfaces(ctx context.Context, params map[string]any) (map[string]any, error) {
	counters := a.netdev()
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	out := []map[string]any{}
	for _, iface := range ifaces {
		addrs, _ := iface.Addrs()
		addrText := []string{}
		for _, addr := range addrs {
			addrText = append(addrText, addr.String())
		}
		state := "DOWN"
		if iface.Flags&net.FlagUp != 0 {
			state = "UP"
		}
		counter := counters[iface.Name]
		out = append(out, map[string]any{"name": iface.Name, "state": state, "addresses": addrText, "rxMiB": round1(float64(counter.rx) / 1024 / 1024), "txMiB": round1(float64(counter.tx) / 1024 / 1024)})
	}
	return map[string]any{"interfaces": out, "source": "local"}, nil
}

func (a *LocalAdapter) ServiceStatus(ctx context.Context, params map[string]any) (map[string]any, error) {
	service := stringParam(params, "service", "darwin-ops-mcp-backend")
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "systemctl", "show", service, "--property=ActiveState,SubState,NRestarts,ExecMainStartTimestamp", "--no-pager").CombinedOutput()
	if err != nil {
		return map[string]any{"service": service, "active": false, "state": "unavailable", "subState": "unknown", "error": strings.TrimSpace(string(out)), "source": "local"}, nil
	}
	kv := parseKeyValueLines(string(out), "=")
	return map[string]any{"service": service, "active": kv["ActiveState"] == "active", "state": kv["ActiveState"], "subState": kv["SubState"], "since": kv["ExecMainStartTimestamp"], "restartCount": kv["NRestarts"], "source": "local"}, nil
}

func (a *LocalAdapter) JournalTail(ctx context.Context, params map[string]any) (map[string]any, error) {
	unit := stringParam(params, "unit", "darwin-ops-mcp-backend")
	lines := intParam(params, "lines", 50)
	if lines <= 0 || lines > 200 {
		lines = 50
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "journalctl", "-u", unit, "-n", strconv.Itoa(lines), "--no-pager", "--output=short-iso").CombinedOutput()
	if err != nil {
		return map[string]any{"unit": unit, "lines": []string{}, "requestedLines": lines, "error": strings.TrimSpace(string(out)), "source": "local"}, nil
	}
	return map[string]any{"unit": unit, "lines": splitNonEmptyLines(string(out)), "requestedLines": lines, "source": "local"}, nil
}

func (a *LocalAdapter) Ping(ctx context.Context, params map[string]any) (map[string]any, error) {
	host := stringParam(params, "host", "1.1.1.1")
	count := intParam(params, "count", 4)
	if count <= 0 || count > 10 {
		count = 4
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(count+3)*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "ping", "-c", strconv.Itoa(count), "-W", "2", host).CombinedOutput()
	result := parsePingSummary(string(out))
	result["host"] = host
	result["count"] = count
	result["source"] = "local"
	if err != nil {
		result["error"] = strings.TrimSpace(string(out))
	}
	return result, nil
}

func (a *LocalAdapter) DNSLookup(ctx context.Context, params map[string]any) (map[string]any, error) {
	host := stringParam(params, "host", "example.com")
	start := a.now()
	records, err := net.DefaultResolver.LookupHost(ctx, host)
	result := map[string]any{"host": host, "records": records, "server": "system-resolver", "durationMs": round1(float64(a.now().Sub(start).Microseconds()) / 1000), "source": "local"}
	if err != nil {
		result["error"] = err.Error()
	}
	return result, nil
}

func (a *LocalAdapter) hostname() string {
	if data, err := os.ReadFile(filepath.Join(a.EtcRoot, "hostname")); err == nil {
		if hostname := strings.TrimSpace(string(data)); hostname != "" {
			return hostname
		}
	}
	return strings.TrimSpace(a.readFirst("sys/kernel/hostname"))
}

func (a *LocalAdapter) readFirst(rel string) string {
	data, _ := os.ReadFile(filepath.Join(a.ProcRoot, rel))
	return strings.TrimSpace(string(data))
}
func (a *LocalAdapter) now() time.Time {
	if a.Now != nil {
		return a.Now()
	}
	return time.Now()
}
func (a *LocalAdapter) readUptime() (int64, error) {
	fields := strings.Fields(a.readFirst("uptime"))
	if len(fields) == 0 {
		return 0, errors.New("missing uptime")
	}
	seconds, err := strconv.ParseFloat(fields[0], 64)
	return int64(seconds), err
}
func (a *LocalAdapter) distribution() string {
	data, err := os.ReadFile(filepath.Join(a.EtcRoot, "os-release"))
	if err != nil {
		return "unknown"
	}
	kv := parseKeyValueLines(string(data), "=")
	if pretty := kv["PRETTY_NAME"]; pretty != "" {
		return pretty
	}
	return kv["NAME"]
}
func (a *LocalAdapter) virtualization() string {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return "docker"
	}
	data, _ := os.ReadFile(filepath.Join(a.ProcRoot, "1/cgroup"))
	text := string(data)
	for _, marker := range []string{"docker", "kubepods", "containerd", "lxc"} {
		if strings.Contains(text, marker) {
			return marker
		}
	}
	return "unknown"
}
func (a *LocalAdapter) cpuCores() int {
	data, err := os.ReadFile(filepath.Join(a.ProcRoot, "cpuinfo"))
	if err != nil {
		return runtime.NumCPU()
	}
	count := strings.Count(string(data), "processor\t:")
	if count == 0 {
		return runtime.NumCPU()
	}
	return count
}
func (a *LocalAdapter) meminfo() map[string]uint64 {
	data, _ := os.ReadFile(filepath.Join(a.ProcRoot, "meminfo"))
	out := map[string]uint64{}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		fields := strings.Fields(strings.TrimSuffix(scanner.Text(), ":"))
		if len(fields) >= 2 {
			key := strings.TrimSuffix(fields[0], ":")
			value, _ := strconv.ParseUint(fields[1], 10, 64)
			out[key] = value
		}
	}
	return out
}

type processInfo struct {
	pid           int
	user          string
	command       string
	rssKiB        uint64
	memoryPercent float64
}

func (a *LocalAdapter) processes() []processInfo {
	entries, _ := os.ReadDir(a.ProcRoot)
	mem := a.meminfo()
	total := mem["MemTotal"]
	out := []processInfo{}
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || !entry.IsDir() {
			continue
		}
		status, err := os.ReadFile(filepath.Join(a.ProcRoot, entry.Name(), "status"))
		if err != nil {
			continue
		}
		kv := parseStatus(string(status))
		rss, _ := strconv.ParseUint(kv["VmRSS"], 10, 64)
		memPct := 0.0
		if total > 0 {
			memPct = round1(float64(rss) / float64(total) * 100)
		}
		out = append(out, processInfo{pid: pid, user: kv["Uid"], command: kv["Name"], rssKiB: rss, memoryPercent: memPct})
	}
	return out
}

type netCounter struct{ rx, tx uint64 }

func (a *LocalAdapter) netdev() map[string]netCounter {
	data, _ := os.ReadFile(filepath.Join(a.ProcRoot, "net/dev"))
	out := map[string]netCounter{}
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		fields := strings.Fields(parts[1])
		if len(fields) < 16 {
			continue
		}
		rx, _ := strconv.ParseUint(fields[0], 10, 64)
		tx, _ := strconv.ParseUint(fields[8], 10, 64)
		out[name] = netCounter{rx: rx, tx: tx}
	}
	return out
}
func parseKeyValueLines(text, sep string) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(text, "\n") {
		key, value, ok := strings.Cut(line, sep)
		if !ok {
			continue
		}
		out[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), `"`)
	}
	return out
}
func parseStatus(text string) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(text, "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		fields := strings.Fields(value)
		if len(fields) == 0 {
			continue
		}
		out[strings.TrimSpace(key)] = fields[0]
	}
	return out
}
func splitNonEmptyLines(text string) []string {
	out := []string{}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}
func parsePingSummary(text string) map[string]any {
	result := map[string]any{"packetLossPercent": 100.0, "avgRttMs": 0.0, "minRttMs": 0.0, "maxRttMs": 0.0}
	for _, line := range strings.Split(text, "\n") {
		if strings.Contains(line, "packet loss") {
			for _, field := range strings.Split(line, ",") {
				field = strings.TrimSpace(field)
				if strings.Contains(field, "packet loss") {
					value := strings.TrimSuffix(strings.Fields(field)[0], "%")
					loss, _ := strconv.ParseFloat(value, 64)
					result["packetLossPercent"] = loss
				}
			}
		}
		if strings.Contains(line, "min/avg/max") {
			_, rhs, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			parts := strings.Split(strings.Fields(strings.TrimSpace(rhs))[0], "/")
			if len(parts) >= 3 {
				minRTT, _ := strconv.ParseFloat(parts[0], 64)
				avgRTT, _ := strconv.ParseFloat(parts[1], 64)
				maxRTT, _ := strconv.ParseFloat(parts[2], 64)
				result["minRttMs"] = round1(minRTT)
				result["avgRttMs"] = round1(avgRTT)
				result["maxRttMs"] = round1(maxRTT)
			}
		}
	}
	return result
}
func round1(v float64) float64 { return float64(int(v*10+0.5)) / 10 }
