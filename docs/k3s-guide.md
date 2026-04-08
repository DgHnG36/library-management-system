# K3s Deployment Guide — LMS (Library Management System)

## Yêu cầu

| Tool      | Phiên bản                |
| --------- | ------------------------ |
| k3s       | ≥ v1.29                  |
| Docker    | ≥ 24 (để build images)   |
| kubectl   | built-in trong k3s       |
| kustomize | ≥ v5 (hoặc `kubectl -k`) |

---

## 1. Cài đặt k3s

```bash
# Cài k3s (single-node)
curl -sfL https://get.k3s.io | sh -

# Kiểm tra node sẵn sàng
sudo k3s kubectl get nodes
```

Tạo alias để dùng `kubectl` thay vì gõ `sudo k3s kubectl`:

```bash
echo 'alias kubectl="sudo k3s kubectl"' >> ~/.bashrc
source ~/.bashrc
```

---

## 2. Build và import Docker images

k3s **không dùng Docker daemon của host** — cần import images thủ công vào containerd của k3s.

```bash
# Chạy từ root của repo
cd /path/to/lib-management-system

# Build tất cả services
docker build -f services/gateway-service/Dockerfile      -t lms/gateway-service:dev .
docker build -f services/user-service/Dockerfile         -t lms/user-service:dev .
docker build -f services/book-service/Dockerfile         -t lms/book-service:dev .
docker build -f services/order-service/Dockerfile        -t lms/order-service:dev .
docker build -f services/notification-service/Dockerfile -t lms/notification-service:dev .

# Import vào k3s containerd
docker save lms/gateway-service:dev      | sudo k3s ctr images import -
docker save lms/user-service:dev         | sudo k3s ctr images import -
docker save lms/book-service:dev         | sudo k3s ctr images import -
docker save lms/order-service:dev        | sudo k3s ctr images import -
docker save lms/notification-service:dev | sudo k3s ctr images import -

# Kiểm tra images đã được import
sudo k3s ctr images ls | grep lms
```

> **Tip:** Mỗi lần rebuild phải import lại. Có thể dùng script `Makefile` target để tự động hóa.

---

## 3. Deploy môi trường Dev

```bash
# Tạo namespace
kubectl create namespace lms-dev

# Dry-run — kiểm tra manifest được render đúng trước khi apply
kubectl kustomize k8s/overlays/dev

# Apply
kubectl apply -k k8s/overlays/dev

# Theo dõi tiến trình
kubectl get pods -n lms-dev -w
```

### Thứ tự khởi động

Kubernetes không đảm bảo thứ tự pod, nhưng các services có readinessProbe nên traffic chỉ được route khi sẵn sàng. Thứ tự khởi động thực tế:

```
postgres → rabbitmq → user/book/order service → gateway → notification
```

---

## 4. Kiểm tra trạng thái

```bash
# Tất cả resources trong namespace
kubectl get all -n lms-dev

# Logs theo service
kubectl logs -n lms-dev -l app=gateway       -f --tail=50
kubectl logs -n lms-dev -l app=user          -f --tail=50
kubectl logs -n lms-dev -l app=book          -f --tail=50
kubectl logs -n lms-dev -l app=order         -f --tail=50
kubectl logs -n lms-dev -l app=notification  -f --tail=50
kubectl logs -n lms-dev -l app=postgres      -f --tail=50
kubectl logs -n lms-dev -l app=rabbitmq      -f --tail=50

# Xem events nếu pod bị crash/pending
kubectl describe pod -n lms-dev <pod-name>
kubectl get events -n lms-dev --sort-by='.lastTimestamp'
```

---

## 5. Truy cập từ local (port-forward)

```bash
# Gateway — HTTP API
kubectl port-forward -n lms-dev svc/lms-gateway-service 8080:8080
# → http://localhost:8080/api/v1/...
# → http://localhost:8080/healthy
# → http://localhost:8080/metrics

# RabbitMQ Management UI
kubectl port-forward -n lms-dev svc/lms-rabbitmq 15672:15672
# → http://localhost:15672 (user: guest / pass: guest)

# PostgreSQL (psql trực tiếp)
kubectl port-forward -n lms-dev svc/lms-postgres 5432:5432
# psql -h localhost -U postgres -d lms_user_db
```

---

## 6. Deploy môi trường Production

> **Bắt buộc trước khi deploy prod:**
>
> 1. Thay `ghcr.io/your-org/...` trong [k8s/overlays/prod/kustomization.yaml](../k8s/overlays/prod/kustomization.yaml) bằng registry thật.
> 2. Inject secrets thật qua CI/CD (xem mục 7).

```bash
# Tạo namespace
kubectl create namespace lms

# Dry-run
kubectl kustomize k8s/overlays/prod

# Apply
kubectl apply -k k8s/overlays/prod

# Theo dõi rollout
kubectl rollout status deployment/lms-gateway-service -n lms
kubectl rollout status deployment/lms-user-service    -n lms
kubectl rollout status deployment/lms-book-service    -n lms
kubectl rollout status deployment/lms-order-service   -n lms
```

---

## 7. Quản lý Secrets trong production

**Không commit** giá trị thật vào `secrets/*.yaml`. Dùng một trong các cách sau:

### Cách 1 — Tạo Secret thủ công qua CI/CD (đơn giản nhất)

```bash
# Chạy trong CI/CD pipeline TRƯỚC bước kustomize apply
kubectl create secret generic lms-gateway-secret -n lms \
  --from-literal=JWT_SECRET="$PROD_JWT_SECRET" \
  --from-literal=REDIS_PASSWORD="$PROD_REDIS_PASSWORD" \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl create secret generic lms-postgres-secret -n lms \
  --from-literal=POSTGRES_USER="postgres" \
  --from-literal=POSTGRES_PASSWORD="$PROD_DB_PASSWORD" \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl create secret generic lms-rabbitmq-secret -n lms \
  --from-literal=RABBITMQ_DEFAULT_USER="$RABBITMQ_USER" \
  --from-literal=RABBITMQ_DEFAULT_PASS="$RABBITMQ_PASS" \
  --dry-run=client -o yaml | kubectl apply -f -
# ... tương tự cho các service còn lại
```

### Cách 2 — Sealed Secrets (an toàn hơn, có thể commit)

```bash
# Cài Sealed Secrets controller
kubectl apply -f https://github.com/bitnami-labs/sealed-secrets/releases/download/v0.27.0/controller.yaml

# Encrypt secret
kubeseal --format yaml < k8s/base/secrets/gateway-secret.yaml \
  > k8s/overlays/prod/sealed-gateway-secret.yaml

# Add vào prod kustomization
# resources:
#   - sealed-gateway-secret.yaml
```

---

## 8. Cập nhật image (Rolling Update)

```bash
# Rebuild và import image mới
docker build -f services/gateway-service/Dockerfile -t lms/gateway-service:dev .
docker save lms/gateway-service:dev | sudo k3s ctr images import -

# Trigger rolling restart (k3s dùng cached image mới)
kubectl rollout restart deployment/lms-gateway-service -n lms-dev

# Kiểm tra rollout
kubectl rollout status deployment/lms-gateway-service -n lms-dev
```

---

## 9. Xóa hoàn toàn

```bash
# Dev
kubectl delete -k k8s/overlays/dev
kubectl delete namespace lms-dev

# Prod
kubectl delete -k k8s/overlays/prod
kubectl delete namespace lms

# Xóa PVC (dữ liệu postgres, rabbitmq) — CẢNH BÁO: không thể phục hồi
kubectl delete pvc --all -n lms-dev
```

---

## 10. Troubleshooting thường gặp

| Triệu chứng                                | Nguyên nhân                                 | Giải pháp                                                            |
| ------------------------------------------ | ------------------------------------------- | -------------------------------------------------------------------- |
| Pod `ImagePullBackOff`                     | k3s không tìm thấy image                    | `docker save ... \| sudo k3s ctr images import -`                    |
| Pod `CrashLoopBackOff`                     | App lỗi khi start                           | `kubectl logs <pod> -n lms-dev --previous`                           |
| Pod `Pending` mãi                          | Không đủ tài nguyên hoặc PVC chưa bound     | `kubectl describe pod <pod>` xem Events                              |
| gRPC readinessProbe fail                   | Service chưa implement gRPC Health Protocol | Tạm thời đổi sang `tcpSocket: {port: 4004X}`                         |
| Gateway không kết nối được user/book/order | DNS chưa resolve                            | Kiểm tra Service name khớp với `GRPC_*_SERVICE_ADDR` trong ConfigMap |
