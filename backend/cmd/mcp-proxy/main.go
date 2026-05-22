package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const defaultEndpoint = "http://127.0.0.1:8080/mcp"

type config struct {
	endpoint string
	token    string
	timeout  time.Duration
}

type frameMode int

const (
	frameLine frameMode = iota
	frameHeader
)

type frame struct {
	mode frameMode
	body []byte
}

type proxy struct {
	cfg    config
	client *http.Client
	in     *bufio.Reader
	out    io.Writer
	err    io.Writer
}

func main() {
	cfg := parseConfig(os.Args[1:], os.Getenv)
	p := &proxy{
		cfg:    cfg,
		client: &http.Client{Timeout: cfg.timeout},
		in:     bufio.NewReader(os.Stdin),
		out:    os.Stdout,
		err:    os.Stderr,
	}
	if err := p.run(context.Background()); err != nil && !errors.Is(err, io.EOF) {
		fmt.Fprintf(os.Stderr, "darwin-ops-mcp-proxy: %v\n", err)
		os.Exit(1)
	}
}

func parseConfig(args []string, getenv func(string) string) config {
	fs := flag.NewFlagSet("darwin-ops-mcp-proxy", flag.ExitOnError)
	fs.SetOutput(os.Stderr)
	endpoint := fs.String("url", firstNonEmpty(getenv("DARWIN_OPS_MCP_URL"), defaultEndpoint), "HTTP MCP endpoint URL")
	token := fs.String("token", getenv("DARWIN_OPS_MCP_API_TOKEN"), "Bearer token for the MCP endpoint")
	timeout := fs.Duration("timeout", envDuration(getenv("DARWIN_OPS_MCP_PROXY_TIMEOUT"), 120*time.Second), "per-request timeout")
	fs.Parse(args)
	return config{endpoint: strings.TrimSpace(*endpoint), token: strings.TrimSpace(*token), timeout: *timeout}
}

func (p *proxy) run(ctx context.Context) error {
	for {
		fr, err := readFrame(p.in)
		if err != nil {
			return err
		}
		body := bytes.TrimSpace(fr.body)
		if len(body) == 0 {
			continue
		}
		respBody, err := p.forward(ctx, body)
		if err != nil {
			return err
		}
		if len(bytes.TrimSpace(respBody)) == 0 {
			continue
		}
		if err := writeFrame(p.out, fr.mode, respBody); err != nil {
			return err
		}
	}
}

func (p *proxy) forward(ctx context.Context, body []byte) ([]byte, error) {
	if p.cfg.endpoint == "" {
		return nil, errors.New("endpoint URL is empty")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if p.cfg.token != "" {
		req.Header.Set("Authorization", "Bearer "+p.cfg.token)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP MCP endpoint returned %s: %s", resp.Status, strings.TrimSpace(string(respBody)))
	}
	return respBody, nil
}

func readFrame(r *bufio.Reader) (frame, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return frame{}, err
	}
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return frame{mode: frameLine, body: nil}, nil
	}
	if !strings.HasPrefix(strings.ToLower(trimmed), "content-length:") {
		return frame{mode: frameLine, body: []byte(trimmed)}, nil
	}
	lengthText := strings.TrimSpace(strings.TrimPrefix(trimmed, "Content-Length:"))
	if lengthText == trimmed {
		lengthText = strings.TrimSpace(strings.TrimPrefix(trimmed, "content-length:"))
	}
	length, err := strconv.Atoi(lengthText)
	if err != nil || length < 0 {
		return frame{}, fmt.Errorf("invalid Content-Length: %q", lengthText)
	}
	for {
		header, err := r.ReadString('\n')
		if err != nil {
			return frame{}, err
		}
		if strings.TrimRight(header, "\r\n") == "" {
			break
		}
	}
	body := make([]byte, length)
	if _, err := io.ReadFull(r, body); err != nil {
		return frame{}, err
	}
	return frame{mode: frameHeader, body: body}, nil
}

func writeFrame(w io.Writer, mode frameMode, body []byte) error {
	body = bytes.TrimSpace(body)
	if mode == frameHeader {
		_, err := fmt.Fprintf(w, "Content-Length: %d\r\n\r\n%s", len(body), body)
		return err
	}
	_, err := fmt.Fprintf(w, "%s\n", body)
	return err
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func envDuration(value string, fallback time.Duration) time.Duration {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	d, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return d
}
