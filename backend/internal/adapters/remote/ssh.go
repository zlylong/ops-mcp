package remote

import (
	"context"
	"errors"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// SSHAdapter executes explicitly approved commands on remote hosts through the
// local ssh client. It does not handle passwords; operators should provide SSH
// keys through the runtime environment/container.
type SSHAdapter struct{}

func NewSSHAdapter() *SSHAdapter { return &SSHAdapter{} }

func (a *SSHAdapter) Command(ctx context.Context, params map[string]any) (map[string]any, error) {
	host := strings.TrimSpace(stringParam(params, "host", ""))
	command := strings.TrimSpace(stringParam(params, "command", ""))
	user := strings.TrimSpace(stringParam(params, "user", "root"))
	port := intParam(params, "port", 22)
	timeoutSeconds := intParam(params, "timeoutSeconds", 30)
	if host == "" {
		return nil, errors.New("host is required")
	}
	if strings.ContainsAny(host, " \t\n\r/") {
		return nil, errors.New("host contains invalid characters")
	}
	if command == "" {
		return nil, errors.New("command is required")
	}
	if port <= 0 || port > 65535 {
		return nil, errors.New("port must be between 1 and 65535")
	}
	if timeoutSeconds <= 0 || timeoutSeconds > 120 {
		timeoutSeconds = 30
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	target := host
	if user != "" {
		target = user + "@" + host
	}
	args := []string{
		"-o", "BatchMode=yes",
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "ConnectTimeout=10",
		"-p", strconv.Itoa(port),
		target,
		command,
	}
	out, err := exec.CommandContext(ctx, "ssh", args...).CombinedOutput()
	result := map[string]any{
		"host":    host,
		"user":    user,
		"port":    port,
		"command": command,
		"output":  string(out),
		"source":  "ssh",
	}
	if ctx.Err() == context.DeadlineExceeded {
		result["timedOut"] = true
		return result, errors.New("ssh command timed out")
	}
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			result["exitCode"] = exit.ExitCode()
		}
		return result, err
	}
	result["exitCode"] = 0
	return result, nil
}

func stringParam(params map[string]any, key, fallback string) string {
	if v, ok := params[key].(string); ok {
		return v
	}
	return fallback
}

func intParam(params map[string]any, key string, fallback int) int {
	switch v := params[key].(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
