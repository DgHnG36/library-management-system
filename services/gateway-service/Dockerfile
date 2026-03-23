FROM golang:1.25-alpine AS build

RUN apk add --no-cache git

WORKDIR /app
COPY services/gateway-service/go.mod services/gateway-service/go.sum ./services/gateway-service/

WORKDIR /app/services/gateway-service
COPY services/gateway-service/ ./

RUN go mod download
RUN go mod tidy

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