[![Go](https://github.com/ThanhTNV/Golang-HTTP-Server/actions/workflows/go.yml/badge.svg)](https://github.com/ThanhTNV/Golang-HTTP-Server/actions/workflows/go.yml)
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
├── app/              # Application code and features
├── cert/             # TLS certificate and key (gitignored)
├── db/               # Database functions and models
├── logs/
│   ├── logger.go     # Zerolog setup, daily file writer, Fiber middleware
│   └── log-files/    # Output directory (gitignored)
│       └── Mar-2026/
│           └── 28-Mar-2026.txt
├── static/           # Static resources
└── generated/        # GORM-generated files
```

## Prerequisites

- Go 1.19 or higher
- Air (for hot reload)

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

## API Documentation

Add your endpoints and usage examples here.

---

**Note:** This is a learning project for practicing Go development patterns.
