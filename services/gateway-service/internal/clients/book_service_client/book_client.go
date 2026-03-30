package book_service_client

import (
	"context"
	"fmt"

	"time"

	"github.com/DgHnG36/lib-management-system/services/gateway-service/pkg/logger"
	bookv1 "github.com/DgHnG36/lib-management-system/shared/go/v1/book"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// BookServiceClient wraps the generated BookServiceClient with additional functionality
type BookServiceClient struct {
	client bookv1.BookServiceClient
	conn   *grpc.ClientConn
	logger *logger.Logger
}

// NewBookServiceClient creates a new book service client connection
func NewBookServiceClient(addr string, logger *logger.Logger) (*BookServiceClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024*1024*100)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to book service at %s: %w", addr, err)
	}

	return &BookServiceClient{
		client: bookv1.NewBookServiceClient(conn),
		conn:   conn,
		logger: logger,
	}, nil
}

// GetBook retrieves a single book by ID or title
func (bc *BookServiceClient) GetBook(ctx context.Context, req *bookv1.GetBookRequest) (*bookv1.BookResponse, error) {
	bc.logger.Info("GetBook called", map[string]interface{}{
		"identifier_type": fmt.Sprintf("%T", req.Identifier),
	})

	resp, err := bc.client.GetBook(ctx, req)
	if err != nil {
		bc.logger.Error("GetBook failed", err, map[string]interface{}{
			"identifier_type": fmt.Sprintf("%T", req.Identifier),
		})
		return nil, fmt.Errorf("failed to get book: %w", err)
	}

	return resp, nil
}

// ListBooks retrieves a list of books with filtering and pagination
func (bc *BookServiceClient) ListBooks(ctx context.Context, req *bookv1.ListBooksRequest) (*bookv1.ListBooksResponse, error) {
	bc.logger.Info("ListBooks called", map[string]interface{}{
		"search_query": req.SearchQuery,
		"category":     req.Category,
	})

	resp, err := bc.client.ListBooks(ctx, req)
	if err != nil {
		bc.logger.Error("ListBooks failed", err, map[string]interface{}{
			"search_query": req.SearchQuery,
			"category":     req.Category,
		})
		return nil, fmt.Errorf("failed to list books: %w", err)
	}

	return resp, nil
}

// CreateBooks creates multiple books
func (bc *BookServiceClient) CreateBooks(ctx context.Context, req *bookv1.CreateBooksRequest) (*bookv1.CreateBooksResponse, error) {
	bc.logger.Info("CreateBooks called", map[string]interface{}{
		"count": len(req.Books),
	})

	resp, err := bc.client.CreateBooks(ctx, req)
	if err != nil {
		bc.logger.Error("CreateBooks failed", err, map[string]interface{}{
			"count": len(req.Books),
		})
		return nil, fmt.Errorf("failed to create books: %w", err)
	}

	return resp, nil
}

// UpdateBook updates an existing book
func (bc *BookServiceClient) UpdateBook(ctx context.Context, req *bookv1.UpdateBookRequest) (*bookv1.BookResponse, error) {
	bc.logger.Info("UpdateBook called", map[string]interface{}{
		"book_id": req.Book.Id,
	})

	resp, err := bc.client.UpdateBook(ctx, req)
	if err != nil {
		bc.logger.Error("UpdateBook failed", err, map[string]interface{}{
			"book_id": req.Book.Id,
		})
		return nil, fmt.Errorf("failed to update book: %w", err)
	}

	return resp, nil
}

// DeleteBooks deletes multiple books by ID
func (bc *BookServiceClient) DeleteBooks(ctx context.Context, req *bookv1.DeleteBooksRequest) error {
	bc.logger.Info("DeleteBooks called", map[string]interface{}{
		"ids": req.Ids,
	})

	_, err := bc.client.DeleteBooks(ctx, req)
	if err != nil {
		bc.logger.Error("DeleteBooks failed", err, map[string]interface{}{
			"ids": req.Ids,
		})
		return fmt.Errorf("failed to delete books: %w", err)
	}

	return nil
}

// CheckAvailability checks if a book is available for borrowing
func (bc *BookServiceClient) CheckAvailability(ctx context.Context, req *bookv1.CheckAvailabilityRequest) (*bookv1.CheckAvailabilityResponse, error) {
	bc.logger.Info("CheckAvailability called", map[string]interface{}{
		"book_id": req.BookId,
	})

	resp, err := bc.client.CheckAvailability(ctx, req)
	if err != nil {
		bc.logger.Error("CheckAvailability failed", err, map[string]interface{}{
			"book_id": req.BookId,
		})
		return nil, fmt.Errorf("failed to check availability: %w", err)
	}

	return resp, nil
}

// UpdateBookQuantity updates the quantity of a book
func (bc *BookServiceClient) UpdateBookQuantity(ctx context.Context, req *bookv1.UpdateBookQuantityRequest) (*bookv1.UpdateBookQuantityResponse, error) {
	bc.logger.Info("UpdateBookQuantity called", map[string]interface{}{
		"book_id":  req.BookId,
		"quantity": req.QuantityChange,
	})

	resp, err := bc.client.UpdateBookQuantity(ctx, req)
	if err != nil {
		bc.logger.Error("UpdateBookQuantity failed", err, map[string]interface{}{
			"book_id":  req.BookId,
			"quantity": req.QuantityChange,
		})
		return nil, fmt.Errorf("failed to update book quantity: %w", err)
	}

	return resp, nil
}

// Close closes the connection to the book service
func (bc *BookServiceClient) Close() error {
	if bc.conn != nil {
		return bc.conn.Close()
	}
	return nil
}

// GetConnection returns the underlying gRPC connection
func (bc *BookServiceClient) GetConnection() *grpc.ClientConn {
	return bc.conn
}

// GetClient returns the underlying generated client
func (bc *BookServiceClient) GetClient() bookv1.BookServiceClient {
	return bc.client
}
