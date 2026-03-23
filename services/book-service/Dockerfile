FROM golang:1.25-alpine AS build

RUN apk add --no-cache git

WORKDIR /app

COPY go.work go.work.sum ./

COPY shared/go/go.mod ./shared/go/
COPY services/book-service/go.mod ./services/book-service/
COPY services/gateway-service/go.mod ./services/gateway-service/
COPY services/order-service/go.mod ./services/order-service/
COPY services/user-service/go.mod ./services/user-service/

RUN go work sync

COPY shared/ ./shared/
COPY services/book-service/ ./services/book-service/

WORKDIR /app/services/book-service
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/book-service ./cmd/grpc/main.go

FROM alpine:3.23

RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Ho_Chi_Minh

WORKDIR /app
COPY --from=build /app/bin/book-service .
RUN chmod +x /app/book-service

EXPOSE 40042
CMD ["./book-service"]