package applications

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/DgHnG36/lib-management-system/services/order-service/internal/broker"
	"github.com/DgHnG36/lib-management-system/services/order-service/internal/models"
	"github.com/DgHnG36/lib-management-system/services/order-service/internal/repository"
	"github.com/DgHnG36/lib-management-system/services/order-service/pkg/logger"
	bookv1 "github.com/DgHnG36/lib-management-system/shared/go/v1/book"
	userv1 "github.com/DgHnG36/lib-management-system/shared/go/v1/user"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type OrderService struct {
	orderRepo  repository.OrderRepository
	bookClient bookv1.BookServiceClient
	userClient userv1.UserServiceClient
	publisher  broker.Publisher
	logger     *logger.Logger
}

func NewOrderService(orderRepo repository.OrderRepository, bookClient bookv1.BookServiceClient, userClient userv1.UserServiceClient, publisher broker.Publisher, logger *logger.Logger) *OrderService {
	return &OrderService{
		orderRepo:  orderRepo,
		bookClient: bookClient,
		userClient: userClient,
		publisher:  publisher,
		logger:     logger,
	}
}

func (s *OrderService) CreateOrder(ctx context.Context, userID string, bookIDs []string, borrowDays int32) (*models.Order, error) {
	s.logger.Info("Creating order", logger.Fields{
		"user_id":     userID,
		"book_ids":    bookIDs,
		"borrow_days": borrowDays,
	})
	// Check availability of books
	type availResult struct {
		bookID string
		err    error
	}
	results := make(chan availResult, len(bookIDs))
	bookCtx := svcCtx(ctx)
	for _, bookID := range bookIDs {
		go func(bID string) {
			resp, err := s.bookClient.CheckAvailability(bookCtx, &bookv1.CheckAvailabilityRequest{
				BookId: bID,
			})
			if err != nil || resp == nil {
				results <- availResult{
					bookID: bID,
					err:    fmt.Errorf("book %s not available", bID),
				}
				return
			}
			results <- availResult{
				bookID: bID,
				err:    nil,
			}
		}(bookID)
	}

	for i := 0; i < len(bookIDs); i++ {
		r := <-results
		if r.err != nil {
			return nil, status.Error(codes.FailedPrecondition, r.err.Error())
		}
	}

	// Create order
	now := time.Now().UTC()
	order := &models.Order{
		ID:         uuid.New().String(),
		UserID:     userID,
		Status:     models.StatusPending,
		BorrowDate: now,
		DueDate:    now.AddDate(0, 0, int(borrowDays)),
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	orderBooks := make([]models.OrderBook, len(bookIDs))
	for i, bookID := range bookIDs {
		orderBooks[i] = models.OrderBook{
			OrderID: order.ID,
			BookID:  bookID,
		}
	}

	order.Books = orderBooks

	if err := s.orderRepo.Create(ctx, order); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create order: %v", err)
	}

	// Update book quantity
	for _, bookID := range bookIDs {
		go func(bID string) {
			_, err := s.bookClient.UpdateBookQuantity(bookCtx, &bookv1.UpdateBookQuantityRequest{
				BookId:       bID,
				ChangeAmount: -1,
			})
			if err != nil {
				s.logger.Error("Failed to update book quantity", err, logger.Fields{
					"book_id": bID,
				})
			}
		}(bookID)
	}

	// Publish event
	go s.publisher.Publish("order.created", map[string]interface{}{
		"order_id": order.ID,
		"user_id":  userID,
		"book_ids": bookIDs,
	})

	s.logger.Info("Order created successfully", logger.Fields{
		"order_id": order.ID,
	})
	return order, nil
}

func (s *OrderService) GetOrder(ctx context.Context, orderID string) (*models.Order, *userv1.User, []*bookv1.Book, error) {
	s.logger.Debug("Getting order", logger.Fields{
		"order_id": orderID,
	})
	order, err := s.orderRepo.FindByID(ctx, orderID)
	if err != nil {
		return nil, nil, nil, status.Errorf(codes.Internal, "failed to get order: %v", err)
	}
	if order == nil {
		return nil, nil, nil, status.Errorf(codes.NotFound, "order %s not found", orderID)
	}

	// Fetch books and user details
	var (
		user     *userv1.User
		books    []*bookv1.Book
		userErr  error
		booksErr error
		wg       sync.WaitGroup
		mu       sync.Mutex
	)

	wg.Add(2)
	go func() {
		defer wg.Done()
		resp, err := s.userClient.GetProfile(ctx, &userv1.GetProfileRequest{
			Id: order.UserID,
		})
		if err != nil {
			userErr = err
			s.logger.Error("Failed to get user profile", err, logger.Fields{
				"user_id": order.UserID,
			})
			return
		}
		user = resp.User
	}()

	go func() {
		defer wg.Done()
		bookList := make([]*bookv1.Book, 0, len(order.Books))
		for _, ob := range order.Books {
			resp, err := s.bookClient.GetBook(ctx, &bookv1.GetBookRequest{
				Identifier: &bookv1.GetBookRequest_Id{
					Id: ob.BookID,
				},
			})
			if err != nil {
				booksErr = err
				s.logger.Error("Failed to get book details", err, logger.Fields{
					"book_id": ob.BookID,
				})
				return
			}
			mu.Lock()
			bookList = append(bookList, resp.Book)
			mu.Unlock()
		}
		books = bookList
	}()

	wg.Wait()
	if userErr != nil {
		return nil, nil, nil, status.Errorf(codes.Internal, "failed to get user details: %v", userErr)
	}
	if booksErr != nil {
		return nil, nil, nil, status.Errorf(codes.Internal, "failed to get book details: %v", booksErr)
	}
	return order, user, books, nil
}

func (s *OrderService) ListMyOrders(ctx context.Context, userID string, page, limit int32, sortBy string, isDesc bool, filterStatus models.OrderStatus) ([]*models.Order, int32, error) {
	s.logger.Debug("Listing user orders", logger.Fields{
		"user_id": userID,
		"page":    page,
		"limit":   limit,
	})
	if page <= 0 {
		page = 1
	}

	if limit <= 0 {
		limit = 10
	}

	orders, total, err := s.orderRepo.FindByUserID(ctx, userID, page, limit, sortBy, isDesc, filterStatus)
	if err != nil {
		return nil, 0, status.Errorf(codes.Internal, "failed to list orders: %v", err)
	}

	return orders, total, nil
}

func (s *OrderService) ListAllOrders(ctx context.Context, page, limit int32, sortBy string, isDesc bool, filterStatus models.OrderStatus, searchUserID string) ([]*models.Order, int32, error) {
	s.logger.Debug("Listing all orders", logger.Fields{
		"page":  page,
		"limit": limit,
	})

	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}
	return s.orderRepo.FindAll(ctx, page, limit, sortBy, isDesc, filterStatus, searchUserID)
}

func (s *OrderService) UpdateOrderStatus(ctx context.Context, orderID string, newStatus models.OrderStatus, note string) (*models.Order, error) {
	s.logger.Info("Update new order status", logger.Fields{
		"order_id":   orderID,
		"new_status": newStatus,
	})

	if newStatus == models.StatusReturned {
		order, err := s.orderRepo.FindByID(ctx, orderID)
		if err != nil || order == nil {
			return nil, status.Errorf(codes.NotFound, "order not found")
		}

		penalty := s.calculatePenalty(order)
		if err := s.orderRepo.UpdateReturnInfo(ctx, orderID, penalty); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to update return info: %v", err)
		}

		bookCtx := svcCtx(ctx)
		for _, ob := range order.Books {
			go func(bID string) {
				_, err := s.bookClient.UpdateBookQuantity(bookCtx, &bookv1.UpdateBookQuantityRequest{
					BookId:       bID,
					ChangeAmount: +1,
				})
				if err != nil {
					s.logger.Error("Failed to update book quantity", err, logger.Fields{
						"book_id": bID,
					})
				}
			}(ob.BookID)
		}
	} else {
		if err := s.orderRepo.UpdateStatus(ctx, orderID, newStatus, note); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to update order status: %v", err)
		}
	}

	go s.publisher.Publish("order.status_updated", map[string]interface{}{
		"order_id":   orderID,
		"new_status": newStatus,
		"note":       note,
	})

	return s.orderRepo.FindByID(ctx, orderID)
}

func (s *OrderService) CancelOrder(ctx context.Context, orderID string, userID string, reason string) (*models.Order, error) {
	s.logger.Info("Canceling order", logger.Fields{
		"order_id": orderID,
		"user_id":  userID,
		"reason":   reason,
	})

	if err := s.orderRepo.Cancel(ctx, orderID, userID, reason); err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "cannot cancel order: not found, not yours, or not in PENDING status")
	}

	s.logger.Info("Order canceled successfully", logger.Fields{
		"order_id": orderID,
		"user_id":  userID,
	})

	order, _ := s.orderRepo.FindByID(ctx, orderID)

	bookCtx := svcCtx(ctx)
	for _, ob := range order.Books {
		go func(bID string) {
			_, err := s.bookClient.UpdateBookQuantity(bookCtx, &bookv1.UpdateBookQuantityRequest{
				BookId:       bID,
				ChangeAmount: +1,
			})
			if err != nil {
				s.logger.Error("Failed to update book quantity", err, logger.Fields{
					"book_id": bID,
				})
			}
		}(ob.BookID)
	}

	go s.publisher.Publish("order.canceled", map[string]interface{}{
		"order_id": orderID,
		"user_id":  userID,
	})

	return order, nil
}

/* HELPER METHODS */

// svcCtx returns a context with outgoing gRPC metadata that identifies this
// call as an internal system request. Backend services (e.g. book-service) use
// this metadata to authorise service-to-service calls without requiring a
// user-level role.
func svcCtx(ctx context.Context) context.Context {
	return metadata.NewOutgoingContext(ctx, metadata.Pairs("X-User-Role", "SYSTEM"))
}

func (s *OrderService) calculatePenalty(order *models.Order) int32 {
	if order.ReturnDate == nil {
		now := time.Now().UTC()
		if now.After(order.DueDate) {
			overdueDays := int(now.Sub(order.DueDate).Hours() / 24)
			return int32(overdueDays) * 5
		}
	}

	return 0
}
