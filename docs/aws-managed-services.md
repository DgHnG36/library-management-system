# AWS Managed Services Guide — LMS

Hướng dẫn triển khai **LMS lên EC2 chạy k3s**, kết nối **AWS RDS (PostgreSQL)**
và **AWS SQS**, build image lưu trên **Docker Hub**, deploy qua **GitHub Actions CD**.

---

## Tóm tắt kiến trúc production

```
GitHub Actions (CD)
  ├─ build & push images → Docker Hub (docker.io/DgHnG36/lms-*)
  └─ kubectl apply -k k8s/overlays/prod
                          │
                    EC2 (k3s cluster)
                    ├─ gateway-service (x3 replicas, Ingress port 80)
                    ├─ user-service
                    ├─ book-service
                    ├─ order-service      ──publish──▶ AWS SQS
                    ├─ notification-service ◀─poll───  AWS SQS
                    ├─ redis (in-cluster)
                    └─ rabbitmq (in-cluster, dùng bởi order-service)
                          │
               ┌──────────┴──────────┐
          AWS RDS (PostgreSQL)   AWS SES (email)
```

| Service    | Chạy ở đâu         | Ghi chú                                      |
| ---------- | ------------------ | -------------------------------------------- |
| PostgreSQL | AWS RDS            | Không cần StatefulSet trong K8s              |
| SQS        | AWS SQS            | Không cần deploy gì trong K8s                |
| RabbitMQ   | Self-hosted in k3s | order-service publish; notification dùng SQS |
| Redis      | Self-hosted in k3s | Gateway rate-limit / cache                   |

---

## 1. Chuẩn bị EC2 + k3s

### 1.1 Tạo EC2 instance

- **AMI**: Ubuntu 24.04 LTS
- **Instance type**: `t3.medium` (2 vCPU, 4GB RAM) cho dev/staging; `t3.large`+ cho prod
- **Storage**: 30GB gp3
- **Security Group** — Inbound:
  | Port | Source | Mục đích |
  |------|--------|----------|
  | 22 | Your IP | SSH |
  | 80 | 0.0.0.0/0 | HTTP (NGINX Ingress) |
  | 443 | 0.0.0.0/0 | HTTPS (nếu có TLS) |
  | 6443 | GitHub Actions IP ranges | kubectl từ CI/CD |

- **IAM Role** gắn vào EC2 (thay cho hardcode credentials):
  ```json
  {
    "Version": "2012-10-17",
    "Statement": [
      {
        "Effect": "Allow",
        "Action": [
          "sqs:ReceiveMessage",
          "sqs:DeleteMessage",
          "sqs:GetQueueAttributes"
        ],
        "Resource": "arn:aws:sqs:ap-southeast-1:<account-id>:notification-queue"
      },
      {
        "Effect": "Allow",
        "Action": ["sqs:SendMessage"],
        "Resource": "arn:aws:sqs:ap-southeast-1:<account-id>:notification-queue"
      },
      {
        "Effect": "Allow",
        "Action": ["ses:SendEmail", "ses:SendRawEmail"],
        "Resource": "*"
      }
    ]
  }
  ```
  > Nếu dùng IAM Role gắn EC2, **không cần** `AWS_ACCESS_KEY_ID`/`AWS_SECRET_ACCESS_KEY` trong secret — boto3/sdk tự lấy qua instance metadata.

### 1.2 Cài k3s

```bash
# SSH vào EC2
ssh -i your-key.pem ubuntu@<EC2_PUBLIC_IP>

# Cài k3s (single-node)
curl -sfL https://get.k3s.io | sh -

# Verify
sudo k3s kubectl get nodes

# Lấy kubeconfig để dùng từ local / CI
sudo cat /etc/rancher/k3s/k3s.yaml
```

> **Lưu ý**: Thay `127.0.0.1` trong kubeconfig bằng `<EC2_PUBLIC_IP>` trước khi lưu vào GitHub Secrets.

### 1.3 Cài NGINX Ingress Controller

k3s mặc định dùng Traefik. Nếu muốn dùng NGINX:

```bash
# Tắt Traefik khi cài k3s
curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC="--disable=traefik" sh -

# Cài NGINX Ingress
sudo k3s kubectl apply -f \
  https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/cloud/deploy.yaml

# Verify — chờ External IP
sudo k3s kubectl get svc -n ingress-nginx
```

### 1.4 Tạo namespace

```bash
sudo k3s kubectl create namespace lms
```

---

## 2. AWS RDS — PostgreSQL

### 2.1 Tạo RDS instance

```bash
aws rds create-db-instance \
  --db-instance-identifier lms-postgres \
  --db-instance-class db.t3.micro \
  --engine postgres \
  --engine-version "16.3" \
  --master-username postgres \
  --master-user-password "$PROD_DB_PASSWORD" \
  --allocated-storage 20 \
  --storage-type gp3 \
  --no-publicly-accessible \
  --vpc-security-group-ids sg-xxxxxxxx \
  --db-subnet-group-name lms-subnet-group \
  --backup-retention-period 7 \
  --region ap-southeast-1
```

> **Network**: RDS và EC2 phải cùng VPC. Security group RDS phải allow TCP `5432` từ security group của EC2.

### 2.2 Tạo databases

```bash
RDS_ENDPOINT=$(aws rds describe-db-instances \
  --db-instance-identifier lms-postgres \
  --query 'DBInstances[0].Endpoint.Address' \
  --output text)

psql -h "$RDS_ENDPOINT" -U postgres -c "CREATE DATABASE lms_user_db;"
psql -h "$RDS_ENDPOINT" -U postgres -c "CREATE DATABASE lms_book_db;"
psql -h "$RDS_ENDPOINT" -U postgres -c "CREATE DATABASE lms_order_db;"
```

### 2.3 K8s config — prod overlay

File `k8s/overlays/prod/kustomization.yaml` patch ConfigMaps trỏ đến RDS:

```yaml
patches:
  - patch: |-
      apiVersion: v1
      kind: ConfigMap
      metadata:
        name: lms-user-config
      data:
        DB_HOST: "lms-postgres.xxxxxxxxxx.ap-southeast-1.rds.amazonaws.com"
        DB_SSL_MODE: "require"
  # ... tương tự book-config, order-config
  # Scale down StatefulSet postgres — không dùng trong prod
  - patch: |-
      apiVersion: apps/v1
      kind: StatefulSet
      metadata:
        name: lms-postgres
      spec:
        replicas: 0
```

---

## 3. AWS SQS — Notification Queue

### 3.1 Tạo SQS queue

```bash
aws sqs create-queue \
  --queue-name notification-queue \
  --attributes '{
    "VisibilityTimeout": "60",
    "MessageRetentionPeriod": "86400",
    "ReceiveMessageWaitTimeSeconds": "20"
  }' \
  --region ap-southeast-1

# Lấy Queue URL
aws sqs get-queue-url \
  --queue-name notification-queue \
  --region ap-southeast-1
# Output: https://sqs.ap-southeast-1.amazonaws.com/<account-id>/notification-queue
```

> `ReceiveMessageWaitTimeSeconds: 20` — bật **Long Polling**, giảm số lần poll và chi phí.

### 3.2 Cấu hình order-service publish lên SQS

Order service cần publish event `order.created`, `order.canceled`, `order.status_updated` lên SQS thay vì RabbitMQ. Xem thêm code trong `services/order-service/`.

### 3.3 K8s config cho notification-service

ConfigMap `k8s/base/configmaps/notification-configmap.yaml`:

```yaml
data:
  SQS_QUEUE_URL: "https://sqs.ap-southeast-1.amazonaws.com/<account-id>/notification-queue"
  AWS_REGION: "ap-southeast-1"
```

Patch đúng URL trong `k8s/overlays/prod/kustomization.yaml`:

```yaml
- patch: |-
    apiVersion: v1
    kind: ConfigMap
    metadata:
      name: lms-notification-config
    data:
      SQS_QUEUE_URL: "https://sqs.ap-southeast-1.amazonaws.com/<account-id>/notification-queue"
```

### 3.4 Auth cho SQS

**Cách 1 — IAM Role (khuyến nghị)**: Gắn IAM Role vào EC2 instance (xem mục 1.1). Không cần set `AWS_ACCESS_KEY_ID`/`AWS_SECRET_ACCESS_KEY`.

**Cách 2 — Credentials trong Secret** (nếu không dùng IAM Role):

```bash
kubectl create secret generic lms-notification-secret \
  --from-literal=AWS_ACCESS_KEY_ID="..." \
  --from-literal=AWS_SECRET_ACCESS_KEY="..." \
  --namespace lms --dry-run=client -o yaml | kubectl apply -f -
```

---

## 4. Docker Hub Registry

### 4.1 Tạo Access Token

1. Đăng nhập Docker Hub → **Account Settings** → **Personal access tokens**
2. Tạo token với quyền `Read, Write, Delete`
3. Lưu vào GitHub Secrets:
   - `DOCKERHUB_USERNAME` = `DgHnG36`
   - `DOCKERHUB_TOKEN` = token vừa tạo

### 4.2 Tên image convention

```
docker.io/DgHnG36/lms-<service>:<git-sha-7>
```

Ví dụ: `docker.io/DgHnG36/lms-gateway-service:a1b2c3d`

Cấu hình trong `k8s/overlays/prod/kustomization.yaml`:

```yaml
images:
  - name: lms/gateway-service
    newName: docker.io/DgHnG36/lms-gateway-service
    newTag: "latest" # CD pipeline sẽ override bằng git SHA
```

---

## 5. GitHub Actions CD Pipeline

### 5.1 Secrets cần thiết

Vào **GitHub repo → Settings → Secrets and variables → Actions**, tạo các secrets sau:

| Secret                       | Mô tả                                                                      |
| ---------------------------- | -------------------------------------------------------------------------- |
| `DOCKERHUB_USERNAME`         | Docker Hub username (`DgHnG36`)                                            |
| `DOCKERHUB_TOKEN`            | Docker Hub access token                                                    |
| `KUBECONFIG`                 | Nội dung file kubeconfig của k3s, **base64-encoded**                       |
| `PROD_DB_PASSWORD`           | Mật khẩu RDS PostgreSQL                                                    |
| `PROD_JWT_SECRET`            | JWT signing secret                                                         |
| `PROD_REDIS_PASSWORD`        | Redis password (để trống nếu không auth)                                   |
| `PROD_AWS_ACCESS_KEY_ID`     | _(Bỏ qua nếu dùng IAM Role)_                                               |
| `PROD_AWS_SECRET_ACCESS_KEY` | _(Bỏ qua nếu dùng IAM Role)_                                               |
| `PROD_SQS_QUEUE_URL`         | `https://sqs.ap-southeast-1.amazonaws.com/<account-id>/notification-queue` |
| `PROD_SES_SENDER_EMAIL`      | Email đã verify trong AWS SES                                              |

### 5.2 Lấy KUBECONFIG từ EC2

```bash
# Trên EC2
sudo cat /etc/rancher/k3s/k3s.yaml | \
  sed 's/127.0.0.1/<EC2_PUBLIC_IP>/g' | \
  base64 -w 0
# Copy output → KUBECONFIG secret trên GitHub
```

### 5.3 Flow CD pipeline

```
push to main
  └─▶ CI passes (unit tests, lint)
        └─▶ CD triggers
              ├─ 1. Checkout code
              ├─ 2. Build Docker images (5 services)
              ├─ 3. Push to Docker Hub với tag = git SHA (7 chars)
              ├─ 4. kustomize edit set image (override tag)
              ├─ 5. Inject secrets via kubectl create secret
              ├─ 6. kubectl apply -k k8s/overlays/prod
              ├─ 7. kubectl rollout status (wait for deploy)
              └─ 8. Rollback nếu fail
```

Xem file [`.github/workflows/cd.yml`](../.github/workflows/cd.yml) để xem implementation đầy đủ.

---

## 6. Chi phí tham khảo (ap-southeast-1)

| Service        | Config                                    | Chi phí ước tính  |
| -------------- | ----------------------------------------- | ----------------- |
| EC2            | `t3.medium`, On-Demand                    | ~$30/tháng        |
| EC2            | `t3.medium`, 1-year Reserved              | ~$19/tháng        |
| RDS PostgreSQL | `db.t3.micro`, 20GB gp3, Single-AZ        | ~$15–18/tháng     |
| SQS            | Standard queue, ~1M messages/tháng        | ~$0.40/tháng      |
| SES            | 1,000 emails/tháng                        | ~$0.10/tháng      |
| Docker Hub     | Free tier (1 private repo) / Pro $5/tháng | $0–5/tháng        |
| **Total**      |                                           | **~$46–56/tháng** |

---

## 7. Checklist deploy lần đầu

```
[ ] Tạo EC2, gắn IAM Role với quyền SQS + SES
[ ] Cài k3s, NGINX Ingress
[ ] Tạo RDS instance, tạo 3 databases
[ ] Tạo SQS queue notification-queue
[ ] Verify SES sender email (AWS Console → SES → Verified identities)
[ ] Cập nhật RDS endpoint trong k8s/overlays/prod/kustomization.yaml
[ ] Cập nhật SQS_QUEUE_URL trong k8s/overlays/prod/kustomization.yaml
[ ] Tạo Docker Hub token, thêm vào GitHub Secrets
[ ] Lấy kubeconfig từ EC2, base64-encode, thêm vào GitHub Secrets
[ ] Thêm tất cả secrets còn lại vào GitHub Secrets
[ ] Push to main → trigger CD
[ ] Kiểm tra: kubectl get pods -n lms
```

---

## Tóm tắt: Có cần tạo Deployment/Service trong K8s không?

| Service    | Self-hosted (k8s)     | AWS Managed                                               |
| ---------- | --------------------- | --------------------------------------------------------- |
| PostgreSQL | StatefulSet + Service | ❌ Không cần — chỉ cần cập nhật `DB_HOST` trong ConfigMap |
| RabbitMQ   | StatefulSet + Service | ✅ Giữ nguyên self-hosted — không thay đổi                |

AWS RDS chạy bên ngoài cluster. Các pods kết nối qua DNS endpoint do AWS cấp
— không cần bất kỳ K8s resource nào cho RDS.

---

## 1. AWS RDS — PostgreSQL

### 1.1 Tạo RDS instance (AWS Console / CLI)

```bash
aws rds create-db-instance \
  --db-instance-identifier lms-postgres \
  --db-instance-class db.t3.micro \
  --engine postgres \
  --engine-version "16.3" \
  --master-username postgres \
  --master-user-password "$PROD_DB_PASSWORD" \
  --allocated-storage 20 \
  --storage-type gp3 \
  --no-publicly-accessible \
  --vpc-security-group-ids sg-xxxxxxxx \  # Security group cho phép k3s node kết nối port 5432
  --db-subnet-group-name lms-subnet-group \
  --backup-retention-period 7 \
  --region ap-southeast-1
```

> **Lưu ý về network:**
>
> - K3s node và RDS phải cùng VPC, hoặc dùng VPC Peering.
> - Security group của RDS phải allow inbound TCP `5432` từ Security group / IP của k3s nodes.

### 1.2 Tạo databases

Sau khi RDS sẵn sàng, kết nối và tạo databases:

```bash
# Lấy endpoint từ AWS Console hoặc CLI
RDS_ENDPOINT=$(aws rds describe-db-instances \
  --db-instance-identifier lms-postgres \
  --query 'DBInstances[0].Endpoint.Address' \
  --output text)

psql -h "$RDS_ENDPOINT" -U postgres -c "CREATE DATABASE lms_user_db;"
psql -h "$RDS_ENDPOINT" -U postgres -c "CREATE DATABASE lms_book_db;"
psql -h "$RDS_ENDPOINT" -U postgres -c "CREATE DATABASE lms_order_db;"
```

### 1.3 Cập nhật K8s — Chỉ cần sửa ConfigMaps và Secrets

Không có StatefulSet hay Service nào cần tạo. Chỉ cần patch trong
`k8s/overlays/prod/kustomization.yaml`:

```yaml
patches:
  # ── Trỏ tất cả services đến RDS endpoint ─────────────────────────────────────
  - patch: |-
      apiVersion: v1
      kind: ConfigMap
      metadata:
        name: lms-user-config
      data:
        DB_HOST: "lms-postgres.xxxxxxxxxx.ap-southeast-1.rds.amazonaws.com"
        DB_SSL_MODE: "require"   # RDS yêu cầu SSL
  - patch: |-
      apiVersion: v1
      kind: ConfigMap
      metadata:
        name: lms-book-config
      data:
        DB_HOST: "lms-postgres.xxxxxxxxxx.ap-southeast-1.rds.amazonaws.com"
        DB_SSL_MODE: "require"
  - patch: |-
      apiVersion: v1
      kind: ConfigMap
      metadata:
        name: lms-order-config
      data:
        DB_HOST: "lms-postgres.xxxxxxxxxx.ap-southeast-1.rds.amazonaws.com"
        DB_SSL_MODE: "require"

  # ── Inject DB_PASSWORD từ Secret ─────────────────────────────────────────────
  # (tạo Secret này qua CI/CD, không commit giá trị thật)
  - patch: |-
      apiVersion: v1
      kind: Secret
      metadata:
        name: lms-user-secret
      stringData:
        DB_PASSWORD: "PLACEHOLDER_REPLACED_BY_CICD"
  # ... tương tự cho book-secret, order-secret
  # ── Scale down postgres StatefulSet — không dùng nữa trong prod ──────────────
  - patch: |-
      apiVersion: apps/v1
      kind: StatefulSet
      metadata:
        name: lms-postgres
      spec:
        replicas: 0
```

---

## 2. Chi phí tham khảo (ap-southeast-1)

| Service        | Config                             | Chi phí ước tính |
| -------------- | ---------------------------------- | ---------------- |
| RDS PostgreSQL | `db.t3.micro`, 20GB gp3, Single-AZ | ~$15–18/tháng    |
| RDS PostgreSQL | `db.t3.small`, 20GB gp3, Multi-AZ  | ~$60/tháng       |
| RabbitMQ       | Self-hosted trong K8s cluster      | $0 thêm          |

> RabbitMQ self-hosted không phát sinh chi phí thêm nếu cluster đã chạy.

---

## 3. So sánh: PostgreSQL Self-hosted vs AWS RDS

| Tiêu chí      | PostgreSQL Self-hosted (k8s) | AWS RDS PostgreSQL     |
| ------------- | ---------------------------- | ---------------------- |
| Chi phí       | $0 thêm (nếu cluster đã có)  | ~$15+/tháng            |
| Backup/HA     | Cần tự cấu hình              | Tự động                |
| Maintenance   | Tự nâng cấp, tự patch        | AWS quản lý            |
| Phù hợp       | Dev, staging                 | Production với SLA cao |
| Code thay đổi | Không                        | Không                  |

RabbitMQ tiếp tục chạy self-hosted trong cả dev lẫn prod — không có thay đổi code hay infrastructure.

**Khuyến nghị:**

- **Dev / k3s local** → self-hosted toàn bộ (postgres + rabbitmq StatefulSet) ✅
- **Prod với ít traffic** → RDS t3.micro + RabbitMQ self-hosted ✅
- **Prod cần SLA cao cho DB** → RDS Multi-AZ + RabbitMQ self-hosted (hoặc xem xét Amazon MQ nếu cần managed AMQP) ✅
