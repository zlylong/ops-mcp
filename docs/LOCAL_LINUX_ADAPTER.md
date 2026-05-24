# Read-only Local Linux Adapter

> 中文版：[LOCAL_LINUX_ADAPTER.zh-CN.md](LOCAL_LINUX_ADAPTER.zh-CN.md)

Darwin Ops MCP ships with a safe mock catalog by default. For a trusted single-host deployment you can opt in to read-only Linux host collection by setting `DARWIN_OPS_MCP_MODE=local`.

Local mode is designed as a narrow observability adapter, not a remote shell. It keeps the same Tool Registry, policy, approval, execution history, audit masking, REST API, and MCP endpoint used by mock mode.

## When to use local mode

Use local mode when you want external AI agents or the Web UI to inspect the Docker host running Darwin Ops MCP:

- Host identity, kernel, distro, uptime, and virtualization markers
- Load average and CPU core count
- Memory and swap usage
- Filesystem usage for a requested path
- Top processes by resident memory
- Network interfaces and byte counters
- Optional service status, journal tail, ping, and DNS checks

Do not use local mode as a substitute for a fleet agent, SSH bastion, or Kubernetes adapter. It only inspects the host/container namespace visible to this backend container and the explicitly mounted read-only host filesystems.

## Enable local mode with Docker Compose

`docker-compose.yml` already contains the required read-only mounts:

```yaml
services:
  backend:
    environment:
      DARWIN_OPS_MCP_MODE: "${DARWIN_OPS_MCP_MODE:-mock}"
    volumes:
      - /proc:/host/proc:ro
      - /etc:/host/etc:ro
      - /usr/lib:/host/usr/lib:ro
```

Create or update `.env` on the deployment host:

```bash
DARWIN_OPS_MCP_MODE=local
# Recommended before exposing outside a trusted network:
DARWIN_OPS_MCP_API_TOKEN=<choose-a-strong-token>
```

Then rebuild and restart the backend:

```bash
docker compose build --no-cache backend
docker compose up -d backend
```

## Security boundary

Local mode is intentionally read-only:

- It does **not** expose arbitrary shell execution.
- It does **not** run user-provided commands.
- It does **not** mutate files, services, containers, Kubernetes resources, or Prometheus state.
- It only uses fixed read-only command shapes for `systemctl show`, `journalctl`, `ping`, and DNS resolution.
- `linux.journal_tail` remains medium risk and approval-required because logs may contain operational context or secrets.
- `/mcp` and `/api/v1/*` can be protected with `DARWIN_OPS_MCP_API_TOKEN`; `/healthz` remains unauthenticated for Docker/monitoring health checks.

Treat the returned host data as operational metadata. Enable the API token before exposing the service beyond a trusted LAN/VPN.

## Data sources

The local adapter prefers mounted host paths when available:

- `/host/proc` for kernel, uptime, load, memory, process, and network counters
- `/host/etc/hostname` for the host name
- `/host/etc/os-release` plus `/host/usr/lib/os-release` for distro metadata on systems where `/etc/os-release` is a symlink
- `statfs(2)` for filesystem usage of the requested path from the container's mount namespace
- Go's `net.Interfaces` and resolver for interfaces and DNS lookups

Container caveat: `/proc/sys/kernel/hostname` can report the container UTS hostname even when `/proc` is bind-mounted. The adapter therefore prefers `/host/etc/hostname` for host identity.

## MCP examples

List tools:

```bash
curl -s http://192.168.20.166:8080/mcp \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer <token>' \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'
```

Get host system information:

```bash
curl -s http://192.168.20.166:8080/mcp \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer <token>' \
  -d '{
    "jsonrpc": "2.0",
    "id": "sysinfo-1",
    "method": "tools/call",
    "params": {
      "name": "linux.system_info",
      "arguments": {
        "actor": "external-agent",
        "role": "viewer",
        "target": "host=192.168.20.166"
      }
    }
  }'
```

Get memory usage:

```bash
curl -s http://192.168.20.166:8080/mcp \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer <token>' \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"linux.memory_usage","arguments":{"actor":"external-agent","role":"viewer","target":"host=192.168.20.166"}}}'
```

## REST examples

The same tools are available through REST/OpenAPI:

```bash
curl -s http://192.168.20.166:8080/api/v1/tools/linux.system_info/execute \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer <token>' \
  -d '{
    "actor": "operator",
    "role": "viewer",
    "target": "host=192.168.20.166",
    "parameters": {}
  }'
```

## Verification checklist

After enabling local mode, verify:

```bash
curl -fsS http://localhost:8080/healthz
curl -fsS http://localhost:8080/swagger/index.html | grep -i 'Swagger UI'
curl -fsS http://localhost:8080/api/v1/tools | python3 -c 'import json,sys; print(len(json.load(sys.stdin)))'
```

Expected health shape:

```json
{
  "mode": "local",
  "status": "ok",
  "tools": 19
}
```

Then call at least one local Linux tool and confirm the returned `data.source` is `local`.

## Troubleshooting

### `mode` is still `mock`

Check that `.env` exists next to `docker-compose.yml` and contains `DARWIN_OPS_MCP_MODE=local`, then recreate the backend container:

```bash
docker compose up -d --force-recreate backend
```

### Backend exits with PostgreSQL authentication errors

Local adapter mode should use in-memory stores. If it tries to connect to PostgreSQL, confirm the running binary includes the local-mode fix and rebuild from the repository source with the current Dockerfile:

```bash
docker compose build --no-cache backend
docker compose up -d backend
```

### Distro is `unknown`

Make sure both `/etc` and `/usr/lib` are mounted read-only. Debian and many other distributions expose `/etc/os-release` as a symlink into `/usr/lib/os-release`.

### Hostname looks like a container ID

Make sure `/etc:/host/etc:ro` is mounted. The adapter prefers `/host/etc/hostname`; without it, container namespace hostname may leak into the result.

### `journal_tail` returns approval required

This is expected. Journal/log content can expose sensitive operational context, so `linux.journal_tail` is medium risk and requires approval even though it is read-only.
