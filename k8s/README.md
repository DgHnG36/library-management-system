# Kubernetes Manifests

This directory contains all Kubernetes manifests for deploying the Library Management System to a cluster. It is structured using [Kustomize](https://kustomize.io/) with a base layer and environment-specific overlays.

---

## Directory Layout

```
k8s/
├── base/                         # Shared base manifests
│   ├── kustomization.yaml
│   ├── ingress.yml               # Ingress rules
│   ├── configmaps/               # ConfigMaps for each service
│   ├── secrets/                  # Secret templates (do NOT commit real values)
│   ├── gateway-service/
│   ├── user-service/
│   ├── book-service/
│   ├── order-service/
│   ├── notification-service/
│   ├── postgres/                 # StatefulSet + Service
│   ├── rabbitmq/                 # StatefulSet + Service
│   └── redis/                    # Deployment + Service
└── overlays/
    ├── dev/                      # Development-specific patches
    └── prod/                     # Production-specific patches
```

---

## Deploying

### Prerequisites

- `kubectl` connected to your target cluster
- `kustomize` v5+ (or `kubectl` v1.27+ with built-in kustomize)

### Development

```bash
kubectl apply -k k8s/overlays/dev
```

### Production

```bash
kubectl apply -k k8s/overlays/prod
```

### Apply Base Only

```bash
kubectl apply -k k8s/base
```

---

## Secrets

Secret manifests under `base/secrets/` contain **placeholder values only**. Before deploying, replace them with real credentials using one of the following approaches:

- Kubernetes Secrets with base64-encoded values
- [Sealed Secrets](https://github.com/bitnami-labs/sealed-secrets)
- AWS Secrets Manager / HashiCorp Vault with a CSI driver

---

## Service Ports

| Service         | Container Port   |
| --------------- | ---------------- |
| gateway-service | `8080`           |
| user-service    | `40041`          |
| book-service    | `40042`          |
| order-service   | `40043`          |
| PostgreSQL      | `5432`           |
| RabbitMQ        | `5672` / `15672` |
| Redis           | `6379`           |

---

## Local Cluster Setup

### Option A — k3s (Linux / WSL2)

#### 1. Install k3s

```bash
curl -sfL https://get.k3s.io | sh -

# Verify the node is ready
sudo k3s kubectl get nodes
```

Add a `kubectl` alias so you don't need to type `sudo k3s kubectl` every time:

```bash
echo 'alias kubectl="sudo k3s kubectl"' >> ~/.bashrc
source ~/.bashrc
```

#### 2. Configure kubeconfig

```bash
# Export the kubeconfig so standard kubectl picks it up
mkdir -p ~/.kube
sudo cp /etc/rancher/k3s/k3s.yaml ~/.kube/config
sudo chown $USER:$USER ~/.kube/config
chmod 600 ~/.kube/config

# (Optional) point KUBECONFIG at a non-default path
export KUBECONFIG=~/.kube/config

# Verify
kubectl cluster-info
kubectl get nodes
```

#### 3. Install the Ingress NGINX Controller

k3s ships with Traefik by default. To use the NGINX ingress (as configured in `base/ingress.yml`), deploy the NGINX ingress controller instead:

```bash
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.10.1/deploy/static/provider/cloud/deploy.yaml

# Wait for the controller to be ready
kubectl wait --namespace ingress-nginx \
  --for=condition=ready pod \
  --selector=app.kubernetes.io/component=controller \
  --timeout=120s
```

#### 4. Build and import Docker images

k3s uses its own containerd and does **not** share images with the Docker daemon. Images must be imported manually:

```bash
# Build all service images
docker build -f services/gateway-service/Dockerfile      -t lms/gateway-service:dev .
docker build -f services/user-service/Dockerfile         -t lms/user-service:dev .
docker build -f services/book-service/Dockerfile         -t lms/book-service:dev .
docker build -f services/order-service/Dockerfile        -t lms/order-service:dev .
docker build -f services/notification-service/Dockerfile -t lms/notification-service:dev .

# Import into k3s containerd
docker save lms/gateway-service:dev      | sudo k3s ctr images import -
docker save lms/user-service:dev         | sudo k3s ctr images import -
docker save lms/book-service:dev         | sudo k3s ctr images import -
docker save lms/order-service:dev        | sudo k3s ctr images import -
docker save lms/notification-service:dev | sudo k3s ctr images import -

# Verify
sudo k3s ctr images ls | grep lms
```

> You must re-import after every rebuild.

#### 5. Deploy

```bash
# Create namespace
kubectl create namespace lms-dev

# Preview rendered manifests (dry-run)
kubectl kustomize k8s/overlays/dev

# Apply
kubectl apply -k k8s/overlays/dev

# Watch pods come up
kubectl get pods -n lms-dev -w
```

---

### Option B — Minikube (Linux / macOS / Windows)

#### 1. Install Minikube

```bash
# Linux / macOS (via curl)
curl -LO https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64
sudo install minikube-linux-amd64 /usr/local/bin/minikube

# macOS (via Homebrew)
brew install minikube

# Windows (via Chocolatey)
choco install minikube
```

#### 2. Start the cluster

```bash
# Start with Docker driver (recommended)
minikube start --driver=docker --cpus=4 --memory=4096

# Verify
minikube status
kubectl cluster-info
```

#### 3. Configure kubeconfig

Minikube automatically updates `~/.kube/config` and sets the current context. To confirm:

```bash
kubectl config current-context   # should print "minikube"
kubectl config get-contexts
```

To switch back to a different cluster later:

```bash
kubectl config use-context <other-context>
```

#### 4. Enable the Ingress NGINX add-on

```bash
minikube addons enable ingress

# Wait for the controller pod
kubectl wait --namespace ingress-nginx \
  --for=condition=ready pod \
  --selector=app.kubernetes.io/component=controller \
  --timeout=120s
```

#### 5. Load Docker images into Minikube

```bash
# Build images first (from repo root)
docker build -f services/gateway-service/Dockerfile      -t lms/gateway-service:dev .
docker build -f services/user-service/Dockerfile         -t lms/user-service:dev .
docker build -f services/book-service/Dockerfile         -t lms/book-service:dev .
docker build -f services/order-service/Dockerfile        -t lms/order-service:dev .
docker build -f services/notification-service/Dockerfile -t lms/notification-service:dev .

# Load into Minikube's image store
minikube image load lms/gateway-service:dev
minikube image load lms/user-service:dev
minikube image load lms/book-service:dev
minikube image load lms/order-service:dev
minikube image load lms/notification-service:dev

# Verify
minikube image ls | grep lms
```

#### 6. Deploy

```bash
kubectl create namespace lms-dev

# Preview
kubectl kustomize k8s/overlays/dev

# Apply
kubectl apply -k k8s/overlays/dev

kubectl get pods -n lms-dev -w
```

#### 7. Access the gateway via Minikube tunnel

Minikube's ingress is not reachable on `localhost` directly. Use the tunnel or port-forward:

```bash
# Option 1 — Minikube tunnel (creates a LoadBalancer IP, run in a separate terminal)
minikube tunnel

# Option 2 — port-forward directly to the gateway service
kubectl port-forward -n lms-dev svc/lms-gateway-service 8080:8080
# → http://localhost:8080

# Get the Minikube ingress IP (for Option 1)
minikube ip
# Then curl http://<minikube-ip>/api/v1/...
```

---

## Accessing Services (port-forward — works on any cluster)

```bash
# Gateway REST API
kubectl port-forward -n lms-dev svc/lms-gateway-service 8080:8080
# → http://localhost:8080/healthy
# → http://localhost:8080/api/v1/...
# → http://localhost:8080/metrics

# RabbitMQ Management UI
kubectl port-forward -n lms-dev svc/lms-rabbitmq 15672:15672
# → http://localhost:15672  (guest / guest)

# PostgreSQL
kubectl port-forward -n lms-dev svc/lms-postgres 5432:5432
# psql -h localhost -U postgres -d lms_user_db
```

---

## Checking Status

```bash
kubectl get all -n lms-dev

# Logs per service
kubectl logs -n lms-dev -l app=gateway      -f --tail=50
kubectl logs -n lms-dev -l app=user         -f --tail=50
kubectl logs -n lms-dev -l app=book         -f --tail=50
kubectl logs -n lms-dev -l app=order        -f --tail=50
kubectl logs -n lms-dev -l app=notification -f --tail=50

# Debug a failing pod
kubectl describe pod -n lms-dev <pod-name>
kubectl get events -n lms-dev --sort-by='.lastTimestamp'
```

---

## See Also

- [monitoring/](../monitoring/) — Prometheus & Grafana
