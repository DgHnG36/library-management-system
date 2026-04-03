package interceptor

import (
	"context"
	"fmt"
	"time"

	"github.com/DgHnG36/lib-management-system/services/user-service/pkg/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func MetadataInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if vals := md.Get("x-user-role"); len(vals) > 0 {
				ctx = context.WithValue(ctx, "X-User-Role", vals[0])
			}
			if vals := md.Get("x-user-id"); len(vals) > 0 {
				ctx = context.WithValue(ctx, "X-User-ID", vals[0])
			}
		}
		return handler(ctx, req)
	}
}

func LoggingInterceptor(log *logger.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		log.Info("gRPC call", logger.Fields{
			"method":  info.FullMethod,
			"latency": time.Since(start).String(),
			"error":   err,
		})
		return resp, err
	}
}

func RecoveryInterceptor(log *logger.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("Panic recovered", fmt.Errorf("%v", r), logger.Fields{
					"method": info.FullMethod,
				})
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()
		return handler(ctx, req)
	}
}
