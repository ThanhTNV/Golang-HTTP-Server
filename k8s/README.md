# Module 2 Homework — Kubernetes Local Cluster with Minikube

Dựng một cụm Kubernetes local bằng Minikube và triển khai một hệ thống giả lập gồm nhiều thành phần khác nhau. Hệ thống này phải sử dụng đầy đủ các loại workload và resource cơ bản nhưng theo hướng gần với production hơn.

## Project Structure

```
├── workloads/
│   ├── deployment.yaml        # Main app (api-go), 3 replicas, env from Secret + ConfigMap
│   ├── replicaset.yaml        # Standalone ReplicaSet (api-go), 3 replicas
│   ├── daemonset.yaml         # DaemonSet (daemonset-go) + ClusterIP Service
│   ├── statefulset.yaml       # PostgreSQL StatefulSet + ClusterIP Service (postgres-svc:5432)
│   ├── job.yaml               # One-off Job (job-go) with PVC storage
│   └── pod.yaml               # Standalone Pod (for learning)
├── resources/
│   ├── configmap.yaml         # POSTGRES_HOST, POSTGRES_PORT
│   ├── secret.yaml            # postgres-secret (credentials) + tls-secret (cert/key)
│   ├── persistent-volume.yaml # PVs: pv-2gb (job), postgres-pv (statefulset)
│   ├── persistent-volume-claim.yaml  # PVC: pvc-2gb (job)
│   ├── ingress.yaml           # NGINX Ingress with TLS, routes /api and /daemonset
│   ├── service-clusterip.yaml # ClusterIP for Deployment (internal)
│   ├── service-nodeport.yaml  # NodePort for Deployment (:32080)
│   └── service-loadbalancer.yaml
├── helm/
│   └── nginx-ingress-controller/
│       ├── nginx-ingress-controller.yaml  # Helm values (image + default TLS cert)
│       └── install.txt                    # Helm install command
├── Apply-Resource.ps1         # Step 1: Apply ConfigMap, Secret, PV, PVC
├── Appy-Workload.ps1          # Step 2: Apply all workloads
├── Apply-ExternalAccess.ps1   # Step 3: Apply Services + Ingress
├── Remove-Exist-Workload.ps1  # Cleanup existing workloads
├── Create_SelfSigned_Cert.ps1 # Generate self-signed cert and store in K8s Secret
└── san.cnf                    # OpenSSL SAN config for cert generation
```

## Requirements Checklist

| Requirement | Status | Implementation |
|---|---|---|
| Deployment cho application chính | Done | `workloads/deployment.yaml` — api-go, 3 replicas |
| ReplicaSet để hiểu cơ chế duy trì số lượng Pod | Done | `workloads/replicaset.yaml` — standalone ReplicaSet |
| DaemonSet để chạy một Pod trên mỗi node | Done | `workloads/daemonset.yaml` — daemonset-go on every node |
| Job hoặc CronJob cho tác vụ nền | Done | `workloads/job.yaml` — one-off job with PVC |
| Service để expose nội bộ | Done | `service-clusterip.yaml` (Deployment), `postgres-svc` (StatefulSet), `k8s-pod-daemonset-svc` (DaemonSet) |
| ConfigMap để cấu hình ứng dụng | Done | `resources/configmap.yaml` — POSTGRES_HOST, POSTGRES_PORT |
| Secret để lưu thông tin nhạy cảm | Done | `resources/secret.yaml` — DB credentials + TLS cert/key |
| Bind ConfigMap và Secret vào Pod (env vars) | Done | Deployment + StatefulSet use `valueFrom` with `secretKeyRef` / `configMapKeyRef` |

## Architecture

```
                    https://localhost
                         │
                         ▼
              ┌─────────────────────┐
              │  NGINX Ingress      │  TLS terminated with tls-secret
              │  Controller (Helm)  │
              ├─────────────────────┤
              │  /api(/|$)(.*)      │──► k8s-pod-clusterip-svc:3000 ──► Deployment Pods (api-go)
              │  /daemonset(/|$)(.*) │──► k8s-pod-daemonset-svc:3000 ──► DaemonSet Pods (daemonset-go)
              └─────────────────────┘

Also available via NodePort (minikube service):
  :32080 → k8s-pod-nodeport-svc       → Deployment Pods (api-go)

Internal (cluster only):
  postgres-svc:5432                    → StatefulSet Pod (postgres)
  k8s-pod-clusterip-svc:3000          → Deployment Pods (api-go)
  k8s-pod-daemonset-svc:3000          → DaemonSet Pods (daemonset-go)

Standalone:
  Job (job-go)                         → runs once with PVC (pvc-2gb)
  ReplicaSet (api-go)                  → 3 replicas (for learning)
```

## How to Deploy

Prerequisites: Minikube running, kubectl configured, local images built.

```powershell
# Load local images into Minikube (imagePullPolicy: Never is set in manifests)
minikube image load api-go:latest
minikube image load job-go:latest
minikube image load daemonset-go:latest

# Generate TLS cert and store in K8s Secret
Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass
.\Create_SelfSigned_Cert.ps1

# Deploy (run in order)
.\Apply-Resource.ps1
.\Appy-Workload.ps1

# Install NGINX Ingress Controller via Helm
helm install nginx-ingress oci://registry-1.docker.io/bitnamicharts/nginx-ingress-controller `
  -f helm/nginx-ingress-controller/nginx-ingress-controller.yaml

# Apply Services + Ingress
.\Apply-ExternalAccess.ps1
kubectl apply -f resources/ingress.yaml
```

## How to Access Services

On Windows with Docker driver, `minikube ip` is not directly reachable.

### Via Ingress (recommended)

```powershell
# Start tunnel in a separate terminal (required for LoadBalancer / Ingress)
minikube tunnel

# Then access:
# https://localhost/api/status        → Deployment (api-go)
# https://localhost/daemonset/status  → DaemonSet (daemonset-go)
```

### Via NodePort

```powershell
minikube service k8s-pod-nodeport-svc
```

### Via port-forward (debugging)

```powershell
kubectl port-forward svc/k8s-pod-clusterip-svc 3000:3000    # api-go on localhost:3000
kubectl port-forward svc/k8s-pod-daemonset-svc 3001:3000    # daemonset-go on localhost:3001
```

## TLS Certificate

Generate a self-signed certificate and store it as a Kubernetes Secret:

```powershell
.\Create_SelfSigned_Cert.ps1
```

This uses `san.cnf` to create a cert with SAN for localhost/127.0.0.1 and stores it in the `tls-secret` Secret. The Ingress and NGINX Ingress Controller both reference this secret for HTTPS termination.
