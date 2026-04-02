package book_service_client

import (
	"context"
	"fmt"

	"github.com/DgHnG36/lib-management-system/services/gateway-service/pkg/logger"
	bookv1 "github.com/DgHnG36/lib-management-system/shared/go/v1/book"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type BookServiceClient struct {
	client bookv1.BookServiceClient
	conn   *grpc.ClientConn
	logger *logger.Logger
}

func NewBookServiceClient(addr string, log *logger.Logger) (*BookServiceClient, error) {
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(4*1024*1024)),
	)

	if err != nil {
		log.Error("Failed to connect to book service", err, logger.Fields{
			"address": addr,
		})
		return nil, fmt.Errorf("failed to connect to book service at %s: %w", addr, err)
	}

	log.Info("Successfully connected to book service", logger.Fields{
		"address": addr,
	})

	return &BookServiceClient{
		client: bookv1.NewBookServiceClient(conn),
		conn:   conn,
		logger: log,
	}, nil
}

func (bc *BookServiceClient) GetBook(ctx context.Context, req *bookv1.GetBookRequest) (*bookv1.BookResponse, error) {
	bc.logger.Info("GetBook called to book service", logger.Fields{
		"identifier": req.GetIdentifier(),
	})

	resp, err := bc.client.GetBook(ctx, req)
	if err != nil {
		bc.logger.Error("GetBook failed", err, logger.Fields{
			"identifier": req.GetIdentifier(),
		})
		return nil, fmt.Errorf("failed to get book: %w", err)
	}

	return resp, nil
}

func (bc *BookServiceClient) ListBooks(ctx context.Context, req *bookv1.ListBooksRequest) (*bookv1.ListBooksResponse, error) {
	bc.logger.Info("ListBooks called to book service", logger.Fields{
		"search_query": req.GetSearchQuery(),
		"category":     req.GetCategory(),
	})

	resp, err := bc.client.ListBooks(ctx, req)
	if err != nil {
		bc.logger.Error("ListBooks failed", err, logger.Fields{
			"search_query": req.GetSearchQuery(),
			"category":     req.GetCategory(),
		})
		return nil, fmt.Errorf("failed to list books: %w", err)
	}

	return resp, nil
}

func (bc *BookServiceClient) CreateBooks(ctx context.Context, req *bookv1.CreateBooksRequest) (*bookv1.CreateBooksResponse, error) {
	bc.logger.Info("CreateBooks called to book service", logger.Fields{
		"count": len(req.Books),
	})

	resp, err := bc.client.CreateBooks(ctx, req)
	if err != nil {
		bc.logger.Error("CreateBooks failed", err, logger.Fields{
			"count": len(req.Books),
		})
		return nil, fmt.Errorf("failed to create books: %w", err)
	}

	return resp, nil
}

func (bc *BookServiceClient) UpdateBook(ctx context.Context, req *bookv1.UpdateBookRequest) (*bookv1.BookResponse, error) {
	bc.logger.Info("UpdateBook called to book service", logger.Fields{
		"book_id": req.GetId(),
	})

	resp, err := bc.client.UpdateBook(ctx, req)
	if err != nil {
		bc.logger.Error("UpdateBook failed", err, logger.Fields{
			"book_id": req.GetId(),
		})
		return nil, fmt.Errorf("failed to update book: %w", err)
	}

	return resp, nil
}

func (bc *BookServiceClient) DeleteBooks(ctx context.Context, req *bookv1.DeleteBooksRequest) error {
	bc.logger.Info("DeleteBooks called to book service", logger.Fields{
		"ids": req.GetIds(),
	})

	_, err := bc.client.DeleteBooks(ctx, req)
	if err != nil {
		bc.logger.Error("DeleteBooks failed", err, logger.Fields{
			"ids": req.GetIds(),
		})
		return fmt.Errorf("failed to delete books: %w", err)
	}

	return nil
}

func (bc *BookServiceClient) CheckAvailability(ctx context.Context, req *bookv1.CheckAvailabilityRequest) (*bookv1.CheckAvailabilityResponse, error) {
	bc.logger.Info("CheckAvailability called to book service", logger.Fields{
		"book_id": req.GetBookId(),
	})

	resp, err := bc.client.CheckAvailability(ctx, req)
	if err != nil {
		bc.logger.Error("CheckAvailability failed", err, logger.Fields{
			"book_id": req.GetBookId(),
		})
		return nil, fmt.Errorf("failed to check availability: %w", err)
	}

	return resp, nil
}

func (bc *BookServiceClient) UpdateBookQuantity(ctx context.Context, req *bookv1.UpdateBookQuantityRequest) (*bookv1.UpdateBookQuantityResponse, error) {
	bc.logger.Info("UpdateBookQuantity called to book service", logger.Fields{
		"book_id":       req.GetBookId(),
		"change_amount": req.GetChangeAmount(),
	})

	resp, err := bc.client.UpdateBookQuantity(ctx, req)
	if err != nil {
		bc.logger.Error("UpdateBookQuantity failed", err, logger.Fields{
			"book_id":       req.GetBookId(),
			"change_amount": req.GetChangeAmount(),
		})
		return nil, fmt.Errorf("failed to update book quantity: %w", err)
	}

	return resp, nil
}

func (bc *BookServiceClient) Close() error {
	if bc.conn != nil {
		return bc.conn.Close()
	}
	return nil
}

func (bc *BookServiceClient) GetConnection() *grpc.ClientConn {
	return bc.conn
}
