[![Go](https://github.com/ThanhTNV/Golang-HTTP-Server/actions/workflows/go.yml/badge.svg)](https://github.com/ThanhTNV/Golang-HTTP-Server/actions/workflows/go.yml)
[![Docker Build & Compose](https://github.com/ThanhTNV/Golang-HTTP-Server/actions/workflows/docker.yml/badge.svg)](https://github.com/ThanhTNV/Golang-HTTP-Server/actions/workflows/docker.yml)
# GoGoGo API Server

A learning project to practice building HTTP API servers in Go.

## Tech Stack

- **Fiber v3** - HTTP framework (HTTPS with TLS)
- **Zerolog** - Structured JSON logging
- **GORM** - Database ORM handling
- **Air** - Hot reload during development

## Project Structure

```
.
├── app/                  # Application code and features
├── cert/                 # TLS certificate and key (gitignored)
├── db/                   # Database functions and models
├── logs/
│   ├── logger.go         # Zerolog setup, daily file writer, Fiber middleware
│   └── log-files/        # Output directory (gitignored)
│       └── Mar-2026/
│           └── 28-Mar-2026.txt
├── static/               # Static resources
├── generated/            # GORM-generated files
├── job/                  # Kubernetes Job — node resource logger
│   ├── main.go
│   ├── Dockerfile
│   ├── go.mod
│   └── k8s/              # PV, PVC, and Job manifests
├── daemonset/            # Kubernetes DaemonSet — node status HTTP API
│   ├── main.go
│   ├── Dockerfile
│   ├── go.mod
│   └── k8s/              # DaemonSet manifest
├── Dockerfile            # Main API server image
└── docker-compose.yml    # Local dev stack (API + Postgres)
```

## Kubernetes Workloads

The repository includes two standalone Go applications designed to run as Kubernetes workloads. Each lives in its own module with a separate `go.mod`, `Dockerfile`, and K8s manifests.

### Job — Node Resource Logger (`job/`)

A one-shot Kubernetes **Job** that samples the current node's CPU and memory usage and writes structured JSON logs to a PersistentVolume.

- Reads `/proc/stat` (two samples, 1 s apart) to compute CPU usage percentage.
- Reads `/proc/meminfo` for total, used, free, available, buffers, and cached memory.
- Falls back to Go `runtime.MemStats` when `/proc` is unavailable (e.g. local testing on macOS/Windows).
- Logs to both **stdout** (human-readable via `zerolog.ConsoleWriter`) and **`/data/job.log`** (JSON) on the mounted volume.

**K8s resources** (`job/k8s/`):

| File | Kind | Purpose |
|------|------|---------|
| `pv.yaml` | PersistentVolume | 2 Gi `hostPath` at `/data/pv-2gb` |
| `pvc.yaml` | PersistentVolumeClaim | Claims 2 Gi `ReadWriteOnce` |
| `job.yaml` | Job | Runs `job-go:latest`, mounts PVC at `/data`, `backoffLimit: 2` |

```bash
kubectl apply -f job/k8s/pv.yaml
kubectl apply -f job/k8s/pvc.yaml
kubectl apply -f job/k8s/job.yaml
```

### DaemonSet — Node Status HTTP API (`daemonset/`)

A long-running Kubernetes **DaemonSet** that exposes an HTTP API (Fiber v3, port 3000) returning real-time node metrics. One pod runs on every node in the cluster.

| Endpoint | Description |
|----------|-------------|
| `GET /health` | Health check — returns `OK` |
| `GET /status` | Full node status (CPU + memory + network) |
| `GET /status/cpu` | CPU count and usage percentage (sampled over 500 ms) |
| `GET /status/memory` | Memory total/used/free/available/buffers/cached (MB) and usage % |
| `GET /status/network` | Per-interface rx/tx bytes, packets, and errors |

Metrics are read from `/proc/stat`, `/proc/meminfo`, and `/proc/net/dev`.

**K8s resources** (`daemonset/k8s/`):

| File | Kind | Purpose |
|------|------|---------|
| `daemonset.yaml` | DaemonSet | Runs `api-go:latest`, exposes port 3000, resource limits 200 m CPU / 256 Mi memory |

```bash
kubectl apply -f daemonset/k8s/daemonset.yaml
```

## Building Docker Images

All three applications use multi-stage Alpine builds. Run these from the repository root:

```bash
# 1. Main API server (used by docker-compose.yml)
docker build -t helloworld:latest .

# 2. Job — node resource logger
docker build -t job-go:latest ./job

# 3. DaemonSet — node status API
docker build -t api-go:latest ./daemonset
```

> **Tip:** If you are using Minikube, point your shell at Minikube's Docker daemon first so the images are available to the cluster without a registry:
> ```bash
> eval $(minikube docker-env)      # Linux / macOS
> minikube docker-env | Invoke-Expression   # PowerShell
> ```

## Prerequisites

- Go 1.19 or higher
- Air (for hot reload)
- Docker (for building images)
- kubectl (for deploying K8s workloads)

## Installation

```bash
# Install dependencies
go mod download

# Install Air for hot reload
go install github.com/cosmtrek/air@latest
```

## Running Locally

The app serves **HTTPS** only (see `main.go` for the listen address, default `:8443`). Generate or copy a key and certificate into `cert/` as described in [Local HTTPS (self-signed certificate)](#local-https-self-signed-certificate).

### Development (with hot reload)
```bash
air
```

### Production Build
```bash
go build -o server
./server
```

## Local HTTPS (self-signed certificate)

Place PEM files where the server expects them:

| File | Purpose |
|------|---------|
| `cert/cert.pem` | Certificate (chain) |
| `cert/key.pem` | Private key (must be **unencrypted** PEM; password-protected keys will not load) |

Browsers expect **Subject Alternative Name (SAN)** entries for TLS to `localhost` and `127.0.0.1`. Run the commands below from the **repository root** so output paths match the app.

### Option A — OpenSSL config file (`san.cnf`)

Create `san.cnf` (project root is fine, or use it only for one-off generation):

```ini
[req]
default_bits       = 4096
prompt             = no
default_md         = sha256
distinguished_name = dn
req_extensions     = req_ext

[dn]
C  = VN
ST = HoChiMinh
L  = HoChiMinh
O  = Dev
OU = Local
CN = localhost

[req_ext]
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
IP.1  = 127.0.0.1
```

Generate the key and certificate:

```bash
openssl req -x509 -newkey rsa:4096 \
  -keyout cert/key.pem -out cert/cert.pem \
  -days 2650 -nodes \
  -config san.cnf
```

- `-config san.cnf` applies SAN and `default_md = sha256` (equivalent to passing `-sha256` on the command line).
- `subjectAltName` must be wired through `[req_ext]`; on older OpenSSL you cannot rely on `-addext` alone.

### Option B — One command with `-addext` (OpenSSL ≥ 1.1.1)

No config file:

```bash
openssl req -x509 -newkey rsa:4096 \
  -keyout cert/key.pem -out cert/cert.pem \
  -days 2650 -nodes \
  -subj "/C=VN/ST=HoChiMinh/L=HoChiMinh/O=Dev/OU=Local/CN=localhost" \
  -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"
```

Then start the app and open `https://localhost:<port>` (you will need to trust or bypass the browser warning for a self-signed cert).

### OpenSSL on Windows

Install with [winget](https://learn.microsoft.com/windows/package-manager/winget/), for example:

```powershell
winget install --id=FireDaemon.OpenSSL -e
```

If `openssl` is not on your `PATH`, extend it (adjust the install path if needed):

```powershell
$Env:PATH += ";C:\Program Files\FireDaemon OpenSSL 3\bin"
```

If your OpenSSL is too old for `-addext`, use **Option A** with `san.cnf`.

## Logging

The project uses [zerolog](https://github.com/rs/zerolog) for structured JSON logging, writing to daily rotating `.txt` files on disk.

### How it works

1. **`logs.Init()`** (called in `main.go`) configures the global zerolog logger with a custom `dailyWriter` that:
   - Creates a **folder per month** under `logs/log-files/` (e.g. `Mar-2026/`)
   - Creates a **file per day** inside that folder (e.g. `28-Mar-2026.txt`)
   - Rotates automatically at midnight — no external tool needed
2. **`logs.FiberMiddleware()`** returns a Fiber logger middleware config with a custom `LoggerFunc` that writes every HTTP request through zerolog with structured fields.
3. **Application code** (e.g. `db/connection.go`) uses `github.com/rs/zerolog/log` directly — all output goes through the same daily writer.

### Log format

Every line is a single JSON object:

```json
{"level":"info","time":"2026-03-28T10:30:56+07:00","message":"server starting"}
{"level":"error","error":"dial tcp [::1]:5432: connectex: ...","time":"2026-03-28T10:30:56+07:00","message":"database: connection failed; continuing without DB"}
{"level":"info","ip":"127.0.0.1","method":"GET","path":"/","status":200,"latency":0.123,"time":"2026-03-28T10:30:58+07:00","message":"request"}
```

Request log level is determined by HTTP status: **info** for 1xx–3xx, **warn** for 4xx, **error** for 5xx.

### File layout example

```
logs/log-files/
├── Mar-2026/
│   ├── 27-Mar-2026.txt
│   └── 28-Mar-2026.txt
└── Apr-2026/
    └── 01-Apr-2026.txt
```

### Usage in code

```go
import "github.com/rs/zerolog/log"

log.Info().Str("key", "value").Msg("something happened")
log.Error().Err(err).Msg("operation failed")
```

No extra setup is needed beyond calling `logs.Init()` once at startup and `defer logs.Close()` for cleanup.

### Concurrency safety

Fiber handles each HTTP request in its own goroutine, so multiple requests may log at the same time. This is safe because of two layers:

1. **Zerolog** builds each log event in a **private buffer** (allocated per call) and delivers the complete JSON line to the underlying `io.Writer` in a **single `Write` call**. There is no risk of interleaved partial lines from different goroutines.
2. **`dailyWriter`** guards every `Write` (and the day-rotation check inside it) with a `sync.Mutex`. Only one goroutine writes to the file at a time.

The flow for concurrent requests looks like this:

```
Goroutine A: zerolog event → private buffer → dailyWriter.Write → acquires mutex → file.Write → releases mutex
Goroutine B: zerolog event → private buffer → dailyWriter.Write → blocks on mutex → file.Write → releases mutex
```

Under extreme throughput the mutex can become a bottleneck because every goroutine serializes on disk I/O. For typical workloads this is a non-issue. If it ever becomes one, a buffered channel writer can be introduced in front of the file without changing any call sites.

## CI / CD (GitHub Actions)

Two workflows run on every push and pull request to `main`:

### Go (`go.yml`)

Builds and tests the Go source directly (no Docker).

| Step | Command |
|------|---------|
| Setup | `actions/setup-go` with Go 1.26.1 |
| Build | `go build -v ./...` |
| Test | `go test -v ./...` |

### Docker Build & Compose (`docker.yml`)

Builds the Docker image, verifies the full stack with Compose, and (on `main` push only) publishes to GitHub Container Registry.

| Job | Purpose |
|-----|---------|
| **Docker Build** | Builds the image with Buildx and writes layers to **GitHub Actions cache** (`type=gha, mode=max`). |
| **Docker Compose** | Rebuilds from cache (`load: true`), generates a throwaway self-signed cert, starts the app + Postgres with `docker compose up`, and runs `curl -fsk https://localhost/` to verify the app responds. |
| **Push to GHCR** | Rebuilds from cache and pushes `ghcr.io/<owner>/<repo>:latest`. Only runs on push to `main`, not on PRs. |
| **Cleanup Build Cache** | Deletes all Buildx `buildkit-*` entries from the Actions cache after the other jobs finish, keeping cache storage clean between runs. |

#### Cache flow

```
docker-build   ──(writes cache)──►  GHA cache
docker-compose ──(reads cache)──►   near-instant rebuild → compose test
push-image     ──(reads cache)──►   near-instant rebuild → push to GHCR
cleanup-cache  ──(deletes cache)──► cache entries removed
```

Each run benefits from the cache internally across jobs, then cleans up so it doesn't consume the 10 GB Actions cache quota between runs.

#### Pull the published image

```bash
docker pull ghcr.io/<owner>/<repo>:latest
```

## API Documentation

Add your endpoints and usage examples here.

---

**Note:** This is a learning project for practicing Go development patterns.
