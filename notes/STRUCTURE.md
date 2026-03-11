lib-management-system/
├── .github/workflows/
│ ├── build-and-push.yml
│ └── deploy-to-eks.yml
│
├── infrastructure/ (Terraform)
│ ├── main.tf
│ ├── vpc.tf
│ ├── eks.tf
│ ├── rds.tf
│ ├── ecr.tf
│ └── variables.tf
│
├── k8s/ (Kubernetes Manifests)
│ ├── ingress/alb-ingress.yaml
│ ├── deployments/
│ │ ├── gateway-deployment.yaml
│ │ ├── auth-deployment.yaml
│ │ ├── book-deployment.yaml
│ │ ├── order-deployment.yaml
│ │ └── notif-deployment.yaml
│ ├── services/
│ │ ├── gateway-service.yaml
│ │ └── ...-service.yaml
│ ├── configmaps/app-config.yaml
│ └── secrets/db-secret.yaml
│
├── proto/
│ ├── Makefile # Tooling to generate Go/Python code
│ ├── common.proto
│ ├── user/
│ │ └── user.proto
│ ├── book/
│ │ └── book.proto
│ └── order/
│ └── order.proto
│
├── shared/
│ ├── go/v1/
│ │ ├── common/common.pb.go
│ │ ├── user/
│ │ │ ├── user.pb.go
│ │ │ └── user_grpc.pb.go
│ │ ├── book/
│ │ │ ├── book.pb.go
│ │ │ └── book_grpc.pb.go
│ │ └── order/
│ │ ├── order.pb.go
│ │ └── order_grpc.pb.go
│ └── python/v1/
│ ├── common_pb2.py
│ ├── user_pb2.py
│ ├── user_pb2_grpc.py
│ └── ... (Other stubs)
│
├── services/
│ ├── gateway-service/ (Golang 1.25.6)
│ │ ├── cmd/api/main.go # Entry point
│ │ ├── internal/
│ │ │ ├── middleware/
│ │ │ │ ├── auth.go # JWT validation
│ │ │ │ ├── cors.go
│ │ │ │ └── logger.go
│ │ │ ├── proxy/
│ │ │ │ └── reverse_proxy.go # REST to gRPC logic
│ │ │ └── router/
│ │ │ └── router.go # Route definitions
│ │ ├── Dockerfile
│ │ ├── go.mod
│ │ └── go.sum
│ │
│ ├── user-service/ (Golang 1.25.6)
│ │ ├── cmd/grpc/main.go # gRPC Server
│ │ ├── internal/
│ │ │ ├── handlers/
│ │ │ │ └── auth_handler.go # Implements UserService gRPC
│ │ │ ├── models/
│ │ │ │ └── user_model.go # GORM DB Entity
│ │ │ ├── repository/
│ │ │ │ ├── db.go # RDS Connection
│ │ │ │ └── user_repo.go # SQL Queries
│ │ │ └── applications/
│ │ │ └── auth_service.go # Business logic (Bcrypt, JWT)
│ │ ├── Dockerfile
│ │ ├── go.mod
│ │ └── go.sum
│ │
│ ├── book-service/ (Golang 1.25.6)
│ │ ├── cmd/grpc/main.go
│ │ ├── internal/
│ │ │ ├── handlers/
│ │ │ │ └── book_handler.go
│ │ │ ├── repository/
│ │ │ │ └── book_repo.go
│ │ │ ├── models/
│ │ │ │ └── book_model.go
│ │ │ └── applications/
│ │ │ └── book_service.go # Inventory logic
│ │ ├── Dockerfile
│ │ ├── go.mod
│ │ └── go.sum
│ │
│ ├── order-service/ (Golang 1.25.6)
│ │ ├── cmd/grpc/main.go
│ │ ├── internal/
│ │ │ ├── handlers/
│ │ │ │ └── order_handler.go
│ │ │ ├── repository/
│ │ │ │ └── order_repo.go
│ │ │ ├── models/
│ │ │ │ └── order_model.go
│ │ │ ├── applications/
│ │ │ │ └── order_service.go # Orchestration logic
│ │ │ └── broker/
│ │ │ └── rabbitmq_pub.go # Push events to MQ
│ │ ├── Dockerfile
│ │ ├── go.mod
│ │ └── go.sum
│ │
│ └── notification-service/ (Python 3.12)
│ ├── main.py # RabbitMQ Consumer
│ ├── requirements.txt
│ ├── Dockerfile
│ └── src/
│ ├── **init**.py
│ ├── email_service.py # AWS SES integration
│ ├── client/
│ │ ├── **init**.py
│ │ ├── user_client.py # Calls User-Service via gRPC
│ │ └── book_client.py # Calls Book-Service via gRPC
│ └── utils/
│ ├── **init**.py
│ ├── logger.py
│ └── config.py
│
├── go.work # Multi-module management
├── docker-compose.yml # Local orchestration
├── Makefile # Global automation
└── README.md

# Tech stack and versions

- Backend Services (Gateway, Auth, Book, Order): Golang 1.25.6
- Background Worker (Notification Service): Python 3.12
- Message Broker: RabbitMQ 4.0-management (UI integrations)
- Relational Database: Amazon RDS (PostgreSQL 16.x or 15.x)
- Cloud Orchestration: Amazon EKS (Kubernetes 1.30 or 1.29)
- Containerization: Docker Engine (Latest version)
