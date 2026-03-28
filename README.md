# GoGoGo — Kubernetes Learning Project

A hands-on project for learning Go microservices and Kubernetes. The repository contains three Go applications and a full set of K8s manifests to deploy them on a local Minikube cluster.

## Repository Structure

```
.
├── api/                        # Main API server (Deployment)
│   ├── main.go                 # Fiber v3 HTTP server with GORM + Zerolog
│   ├── app/                    # Feature handlers (e.g. pets)
│   ├── db/                     # Database connection and queries
│   ├── logs/                   # Zerolog daily-rotating file writer + Fiber middleware
│   ├── generated/              # GORM-generated code
│   ├── static/                 # Static assets
│   ├── Dockerfile
│   ├── docker-compose.yml      # Local dev stack (API + Postgres)
│   └── go.mod
│
├── job/                        # Node resource logger (Job)
│   ├── main.go                 # Reads /proc for CPU + memory, logs to PV
│   ├── Dockerfile
│   ├── k8s/                    # PV, PVC, and Job manifests
│   └── go.mod
│
├── daemonset/                  # Node status HTTP API (DaemonSet)
│   ├── main.go                 # Fiber v3 server exposing /status endpoints
│   ├── Dockerfile
│   ├── k8s/                    # DaemonSet manifest
│   └── go.mod
│
├── k8s/                        # Kubernetes manifests and deploy scripts
│   ├── resources/              # ConfigMap, Secret, PV, PVC, Services, Ingress
│   ├── workloads/              # Deployment, ReplicaSet, DaemonSet, StatefulSet, Job, Pod
│   ├── helm/                   # NGINX Ingress Controller Helm values
│   ├── cert/                   # Generated TLS cert/key (gitignored)
│   ├── Apply-Resource.ps1      # Step 1: apply resources
│   ├── Appy-Workload.ps1       # Step 2: apply workloads
│   ├── Apply-ExternalAccess.ps1# Step 3: apply services + ingress
│   ├── Remove-Exist-Workload.ps1
│   ├── Create_SelfSigned_Cert.ps1
│   └── san.cnf
│
└── .github/workflows/
    ├── ci-api.yml              # Go build + test for api/
    └── docker-api.yml          # Docker build, compose test, push to GHCR
```

## Applications

### `api/` — Main API Server (Deployment)

The primary service, deployed as a Kubernetes **Deployment** with 3 replicas. Built with:

- **Fiber v3** — HTTP framework
- **GORM** — PostgreSQL ORM
- **Zerolog** — Structured JSON logging with daily file rotation

The Deployment reads database credentials from a **Secret** and connection config from a **ConfigMap**. Exposed inside the cluster via ClusterIP and externally via NodePort and Ingress.

| Image | Port | K8s Workload |
|-------|------|--------------|
| `api-go:latest` | 3000 | Deployment (3 replicas) |

### `job/` — Node Resource Logger (Job)

A one-shot Kubernetes **Job** that samples the node's CPU and memory usage, then exits. Logs are persisted to a PersistentVolume so they survive after the pod terminates.

- Reads `/proc/stat` (two samples 1 s apart) to compute CPU usage percentage
- Reads `/proc/meminfo` for total, used, free, available, buffers, and cached memory
- Falls back to Go `runtime.MemStats` when `/proc` is unavailable (local dev on macOS/Windows)
- Writes to both **stdout** (human-readable `zerolog.ConsoleWriter`) and **`/data/job.log`** (JSON) on the mounted volume

| Image | K8s Workload | Storage |
|-------|--------------|---------|
| `job-go:latest` | Job (`backoffLimit: 2`) | PVC `pvc-2gb` mounted at `/data` |

### `daemonset/` — Node Status HTTP API (DaemonSet)

A long-running Kubernetes **DaemonSet** that runs one pod on every node and exposes real-time node metrics over HTTP.

| Endpoint | Description |
|----------|-------------|
| `GET /health` | Health check — returns `OK` |
| `GET /status` | Full node status (CPU + memory + network) |
| `GET /status/cpu` | CPU count and usage percentage (sampled over 500 ms) |
| `GET /status/memory` | Memory breakdown in MB and usage % |
| `GET /status/network` | Per-interface rx/tx bytes, packets, and errors |

Metrics are read from `/proc/stat`, `/proc/meminfo`, and `/proc/net/dev`.

| Image | Port | K8s Workload |
|-------|------|--------------|
| `api-go:latest` | 3000 | DaemonSet (1 pod per node) |

## `k8s/` — Kubernetes Manifests

All manifests needed to run the system on a Minikube cluster.

### Resources (`k8s/resources/`)

| File | Kind | Purpose |
|------|------|---------|
| `configmap.yaml` | ConfigMap | `POSTGRES_HOST`, `POSTGRES_PORT` |
| `secret.yaml` | Secret | DB credentials (`postgres-secret`) + TLS cert/key (`tls-secret`) |
| `persistent-volume.yaml` | PersistentVolume | `pv-2gb` (Job), `postgres-pv` (StatefulSet) |
| `persistent-volume-claim.yaml` | PersistentVolumeClaim | `pvc-2gb` (Job) |
| `service-clusterip.yaml` | Service (ClusterIP) | Internal access to Deployment pods |
| `service-nodeport.yaml` | Service (NodePort) | External access on port 32080 |
| `service-loadbalancer.yaml` | Service (LoadBalancer) | External access via cloud/tunnel LB |
| `ingress.yaml` | Ingress | NGINX Ingress with TLS — `/api` and `/daemonset` routes |

### Workloads (`k8s/workloads/`)

| File | Kind | Image | Purpose |
|------|------|-------|---------|
| `deployment.yaml` | Deployment | `api-go:latest` | Main API, 3 replicas, env from Secret + ConfigMap |
| `replicaset.yaml` | ReplicaSet | `api-go:latest` | Standalone ReplicaSet (for learning) |
| `daemonset.yaml` | DaemonSet | `daemonset-go:latest` | Node status API, 1 pod per node + ClusterIP Service |
| `statefulset.yaml` | StatefulSet | `postgres:17-alpine` | PostgreSQL with stable identity + headless Service |
| `job.yaml` | Job | `job-go:latest` | Node resource logger with PVC |
| `pod.yaml` | Pod | — | Standalone pod (for learning) |

### Architecture

```
                    https://localhost
                         │
                         ▼
              ┌─────────────────────┐
              │  NGINX Ingress      │  TLS terminated with tls-secret
              │  Controller (Helm)  │
              ├─────────────────────┤
              │  /api(/|$)(.*)      │──► k8s-pod-clusterip-svc:3000 ──► Deployment (api-go)
              │  /daemonset(/|$)(.*) │──► k8s-pod-daemonset-svc:3000 ──► DaemonSet (daemonset-go)
              └─────────────────────┘

Also available via NodePort:
  :32080 → k8s-pod-nodeport-svc → Deployment (api-go)

Internal (cluster only):
  postgres-svc:5432              → StatefulSet (postgres)
  k8s-pod-clusterip-svc:3000    → Deployment (api-go)
  k8s-pod-daemonset-svc:3000    → DaemonSet (daemonset-go)

Standalone:
  Job (job-go)                   → runs once, logs to PVC (pvc-2gb)
```

## Building Docker Images

All three applications use multi-stage Alpine builds. Run from the repository root:

```bash
# 1. API server — main service (Deployment)
docker build -t api-go:latest ./api

# 2. Job — node resource logger
docker build -t job-go:latest ./job

# 3. DaemonSet — node status API
docker build -t daemonset-go:latest ./daemonset
```

To make images available inside Minikube without a registry:

```bash
# Linux / macOS
eval $(minikube docker-env)

# PowerShell
minikube docker-env | Invoke-Expression
```

Then re-run the `docker build` commands above, or load pre-built images:

```bash
minikube image load api-go:latest
minikube image load job-go:latest
minikube image load daemonset-go:latest
```

## Deploying to Minikube

### Prerequisites

- [Minikube](https://minikube.sigs.k8s.io/) running (`minikube start`)
- `kubectl` configured
- [Helm](https://helm.sh/) (for NGINX Ingress Controller)
- All three Docker images built and loaded (see above)

### Step-by-step

```powershell
cd k8s

# 1. Generate TLS cert and store in K8s Secret
Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass
.\Create_SelfSigned_Cert.ps1

# 2. Apply resources (ConfigMap, Secret, PV, PVC)
.\Apply-Resource.ps1

# 3. Apply workloads (StatefulSet, Deployment, DaemonSet, ReplicaSet, Job)
.\Appy-Workload.ps1

# 4. Install NGINX Ingress Controller
helm install nginx-ingress oci://registry-1.docker.io/bitnamicharts/nginx-ingress-controller `
  -f helm/nginx-ingress-controller/nginx-ingress-controller.yaml

# 5. Apply Services + Ingress
.\Apply-ExternalAccess.ps1
kubectl apply -f resources/ingress.yaml
```

### Accessing Services

On Windows with Docker driver, `minikube ip` is not directly reachable. Use one of:

**Via Ingress** (recommended):

```powershell
# Start tunnel in a separate terminal
minikube tunnel

# Then access:
# https://localhost/api/         → Deployment (api-go)
# https://localhost/daemonset/   → DaemonSet (daemonset-go)
```

**Via NodePort:**

```powershell
minikube service k8s-pod-nodeport-svc
```

**Via port-forward** (debugging):

```powershell
kubectl port-forward svc/k8s-pod-clusterip-svc 3000:3000    # api-go
kubectl port-forward svc/k8s-pod-daemonset-svc 3001:3000    # daemonset-go
```

## Local Development (API only)

```bash
cd api

# Install dependencies
go mod download

# Install Air for hot reload
go install github.com/cosmtrek/air@latest

# Run with hot reload
air

# Or build and run directly
go build -o server
./server
```

The API server also runs locally with Docker Compose (API + Postgres):

```bash
cd api
docker compose up
```

## CI / CD (GitHub Actions)

Two workflows trigger on changes to `api/**`:

### Go API (`ci-api.yml`)

Builds and tests the Go source.

| Step | Command |
|------|---------|
| Setup | `actions/setup-go` with Go 1.26.1 |
| Build | `go build -v ./...` |
| Test | `go test -v ./...` |

### Docker Build & Compose (`docker-api.yml`)

Builds the Docker image, verifies the stack with Compose, and publishes to GHCR.

| Job | Purpose |
|-----|---------|
| **Docker Build** | Builds `api/` with Buildx, writes layers to GHA cache |
| **Docker Compose** | Rebuilds from cache, starts API + Postgres, runs `curl http://localhost:3000/` |
| **Push to GHCR** | Pushes `ghcr.io/<owner>/<repo>:latest` (main branch only) |
| **Cleanup Build Cache** | Deletes Buildx cache entries to stay within GHA quota |

---

**Note:** This is a learning project for practicing Go and Kubernetes patterns.
