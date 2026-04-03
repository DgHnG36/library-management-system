FROM golang:1.25-alpine AS build

RUN apk add --no-cache git

WORKDIR /app

COPY go.work go.work.sum ./

COPY shared/go/go.mod ./shared/go/
COPY services/gateway-service/go.mod ./services/gateway-service/
COPY services/book-service/go.mod ./services/book-service/
COPY services/order-service/go.mod ./services/order-service/
COPY services/user-service/go.mod ./services/user-service/
COPY test/unit/test-gateway-service/go.mod ./test/unit/test-gateway-service/
COPY test/unit/test-user-service/go.mod ./test/unit/test-user-service/
COPY test/unit/test-book-service/go.mod ./test/unit/test-book-service/
COPY test/unit/test-order-service/go.mod ./test/unit/test-order-service/
COPY test/integration/go.mod ./test/integration/

RUN go work sync

COPY shared/ ./shared/
COPY services/gateway-service/ ./services/gateway-service/

WORKDIR /app/services/gateway-service
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/gateway-service ./cmd/api/main.go

FROM alpine:3.23
RUN apk add --no-cache ca-certificates tzdata curl
ENV TZ=Asia/Ho_Chi_Minh

WORKDIR /app
COPY --from=build /app/bin/gateway-service .
RUN chmod +x /app/gateway-service

HEALTHCHECK --interval=30s --timeout=30s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/healthy || exit 1

EXPOSE 8080
CMD [ "./gateway-service" ]