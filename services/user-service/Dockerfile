FROM golang:1.25-alpine AS build

RUN apk add --no-cache git

WORKDIR /app
COPY go.work go.work.sum ./

COPY shared/go/go.mod ./shared/go/
COPY services/user-service/go.mod ./services/user-service/
COPY services/gateway-service/go.mod ./services/gateway-service/
COPY services/book-service/go.mod ./services/book-service/
COPY services/order-service/go.mod ./services/order-service/
COPY test/unit/test-gateway-service/go.mod ./test/unit/test-gateway-service/
COPY test/unit/test-user-service/go.mod ./test/unit/test-user-service/
COPY test/unit/test-book-service/go.mod ./test/unit/test-book-service/
COPY test/unit/test-order-service/go.mod ./test/unit/test-order-service/
COPY test/integration/go.mod ./test/integration/

RUN go work sync

COPY shared ./shared/
COPY services/user-service/ ./services/user-service/

WORKDIR /app/services/user-service
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/user-service ./cmd/grpc/main.go

FROM alpine:3.23

RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Ho_Chi_Minh

WORKDIR /app
COPY --from=build /app/bin/user-service .
RUN chmod +x /app/user-service

EXPOSE 40041
CMD ["./user-service"]


