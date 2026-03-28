# GoGoGo API Server

A learning project to practice building HTTP API servers in Go.

## Tech Stack

- **Air** - Hot reload during development
- **Mux Router** - HTTP request routing
- **GORM** - Database ORM handling

## Project Structure

```
.
├── app/              # Application code and features
├── db/               # Database functions and models
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

## API Documentation

Add your endpoints and usage examples here.

---

**Note:** This is a learning project for practicing Go development patterns.