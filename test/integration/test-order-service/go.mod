module github.com/DgHnG36/lib-management-system/services/order-service/test/integration

go 1.25.6

require (
	github.com/DgHnG36/lib-management-system/services/order-service v0.0.0
	github.com/stretchr/testify v1.11.1
	google.golang.org/grpc v1.79.2
	gorm.io/driver/postgres v1.6.0
	gorm.io/gorm v1.31.1
)

replace github.com/DgHnG36/lib-management-system/services/order-service => ../../../services/order-service
