# ─── Build stage ─────────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS build

RUN apk add --no-cache git

WORKDIR /app

COPY go.work go.work.sum ./

COPY shared/go/go.mod ./shared/go/
COPY services/order-service/go.mod services/order-service/go.sum ./services/order-service/

RUN go work sync

COPY shared/ ./shared/
COPY services/order-service/ ./services/order-service/

WORKDIR /app
COPY shared/go/ ./shared/go/
COPY services/order-service/ ./services/order-service/

WORKDIR /app/services/order-service
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/order-service ./cmd/grpc/main.go

FROM alpine:3.23

RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Ho_Chi_Minh

WORKDIR /app
COPY --from=build /app/bin/order-service .
RUN chmod +x /app/order-service

EXPOSE 40043
CMD ["./order-service"]
