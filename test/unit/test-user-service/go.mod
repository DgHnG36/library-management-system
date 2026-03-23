module github.com/DgHnG36/lib-management-system/services/user-service/test/unit/test-user-service

go 1.25.6

require (
	github.com/DgHnG36/lib-management-system/services/user-service v0.0.0
	github.com/stretchr/testify v1.11.1
	golang.org/x/crypto v0.32.0
	google.golang.org/grpc v1.65.0
)

replace github.com/DgHnG36/lib-management-system/services/user-service => ../../../services/user-service
