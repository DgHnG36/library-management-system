package handlers

import (
	"context"

	"github.com/DgHnG36/lib-management-system/services/book-service/internal/applications"
	"github.com/DgHnG36/lib-management-system/services/book-service/internal/models"
	"github.com/DgHnG36/lib-management-system/services/book-service/pkg/interceptor"
	"github.com/DgHnG36/lib-management-system/services/book-service/pkg/logger"
	commonv1 "github.com/DgHnG36/lib-management-system/shared/go/v1"
	bookv1 "github.com/DgHnG36/lib-management-system/shared/go/v1/book"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type BookHandler struct {
	bookv1.UnimplementedBookServiceServer
	bookService *applications.BookService
	logger      *logger.Logger
}

func NewBookHandler(bookService *applications.BookService, logger *logger.Logger) *BookHandler {
	return &BookHandler{
		bookService: bookService,
		logger:      logger,
	}
}

func (h *BookHandler) GetBook(ctx context.Context, req *bookv1.GetBookRequest) (*bookv1.BookResponse, error) {
	var id, title string

	switch ident := req.Identifier.(type) {
	case *bookv1.GetBookRequest_Id:
		id = ident.Id
	case *bookv1.GetBookRequest_Title:
		title = ident.Title
	default:
		return nil, status.Error(codes.InvalidArgument, "ID or Title is required")
	}

	book, err := h.bookService.GetBook(ctx, id, title)
	if err != nil {
		h.logger.Error("Failed to get book", err, logger.Fields{
			"id":    id,
			"title": title,
		})
		return nil, err
	}

	return &bookv1.BookResponse{
		Book: toPbBook(book),
	}, nil
}

func (h *BookHandler) ListBooks(ctx context.Context, req *bookv1.ListBooksRequest) (*bookv1.ListBooksResponse, error) {
	var page, limit int32 = 1, 10
	var sortBy string
	var isDesc bool

	if req.GetPagination() != nil {
		page = req.Pagination.GetPage()
		limit = req.Pagination.GetLimit()
		sortBy = req.Pagination.GetSortBy()
		isDesc = req.Pagination.GetIsDesc()
	}

	searchQuery := req.GetSearchQuery()
	category := req.GetCategory()

	books, total, err := h.bookService.ListBooks(ctx, page, limit, sortBy, isDesc, searchQuery, category)
	if err != nil {
		h.logger.Error("Failed to list books", err, logger.Fields{
			"page":  page,
			"limit": limit,
		})
		return nil, err
	}

	pbBooks := make([]*bookv1.Book, len(books))
	for i, b := range books {
		pbBooks[i] = toPbBook(b)
	}

	return &bookv1.ListBooksResponse{
		Books:      pbBooks,
		TotalCount: total,
	}, nil
}

func (h *BookHandler) CreateBooks(ctx context.Context, req *bookv1.CreateBooksRequest) (*bookv1.CreateBooksResponse, error) {
	role := ctx.Value(interceptor.ContextKeyUserRole)
	if role != "ADMIN" && role != "MANAGER" {
		return nil, status.Error(codes.PermissionDenied, "only admins and managers can create books")
	}

	if len(req.GetBooks()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one book is required")
	}

	payloads := make([]applications.CreateBookPayload, len(req.GetBooks()))
	for i, b := range req.GetBooks() {
		payloads[i] = applications.CreateBookPayload{
			Title:         b.GetTitle(),
			Author:        b.GetAuthor(),
			ISBN:          b.GetIsbn(),
			Category:      b.GetCategory(),
			Description:   b.GetDescription(),
			TotalQuantity: b.GetTotalQuantity(),
		}
	}

	books, err := h.bookService.CreateBooks(ctx, payloads)
	if err != nil {
		h.logger.Error("Failed to create books", err)
		return nil, err
	}

	pbBooks := make([]*bookv1.Book, len(books))
	for i, b := range books {
		pbBooks[i] = toPbBook(b)
	}

	return &bookv1.CreateBooksResponse{
		Books:        pbBooks,
		SuccessCount: int32(len(books)),
	}, nil
}

func (h *BookHandler) UpdateBook(ctx context.Context, req *bookv1.UpdateBookRequest) (*bookv1.BookResponse, error) {
	role := ctx.Value(interceptor.ContextKeyUserRole)
	if role != "ADMIN" && role != "MANAGER" {
		return nil, status.Error(codes.PermissionDenied, "only admins and managers can update books")
	}

	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "ID book is required")
	}

	titleBook := req.GetTitle()
	author := req.GetAuthor()
	isbn := req.GetIsbn()
	category := req.GetCategory()
	description := req.GetDescription()

	book, err := h.bookService.UpdateBook(ctx,
		req.Id,
		titleBook,
		author,
		isbn,
		category,
		description,
	)
	if err != nil {
		h.logger.Error("Failed to update book", err, logger.Fields{
			"book_id": req.Id,
		})
		return nil, err
	}

	return &bookv1.BookResponse{
		Book: toPbBook(book),
	}, nil
}

func (h *BookHandler) DeleteBooks(ctx context.Context, req *bookv1.DeleteBooksRequest) (*commonv1.BaseResponse, error) {
	role := ctx.Value(interceptor.ContextKeyUserRole)
	if role != "ADMIN" && role != "MANAGER" {
		return nil, status.Error(codes.PermissionDenied, "only admins and managers can delete books")
	}

	if len(req.GetIds()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "ID books are required")
	}

	if err := h.bookService.DeleteBooks(ctx, req.GetIds()); err != nil {
		h.logger.Error("Failed to delete books", err, logger.Fields{
			"book_ids": req.Ids,
		})
		return nil, err
	}

	return &commonv1.BaseResponse{
		Status:  200,
		Message: "Books deleted successfully",
	}, nil
}

func (h *BookHandler) CheckAvailability(ctx context.Context, req *bookv1.CheckAvailabilityRequest) (*bookv1.CheckAvailabilityResponse, error) {
	role := ctx.Value(interceptor.ContextKeyUserRole)
	if role != "ADMIN" && role != "MANAGER" && role != "SYSTEM" {
		return nil, status.Error(codes.PermissionDenied, "only admins, managers, and users can check book availability")
	}

	if req.GetBookId() == "" {
		return nil, status.Error(codes.InvalidArgument, "ID book is required")
	}

	isAvailable, qty, err := h.bookService.CheckAvailability(ctx, req.BookId)
	if err != nil {
		h.logger.Error("Failed to check availability", err, logger.Fields{
			"book_id": req.BookId,
		})
		return nil, err
	}

	return &bookv1.CheckAvailabilityResponse{
		IsAvailable:       isAvailable,
		AvailableQuantity: qty,
	}, nil
}

func (h *BookHandler) UpdateBookQuantity(ctx context.Context, req *bookv1.UpdateBookQuantityRequest) (*bookv1.UpdateBookQuantityResponse, error) {
	role := ctx.Value(interceptor.ContextKeyUserRole)
	if role != "ADMIN" && role != "MANAGER" && role != "SYSTEM" {
		return nil, status.Error(codes.PermissionDenied, "only admins and managers can update book quantity")
	}

	if req.GetBookId() == "" {
		return nil, status.Error(codes.InvalidArgument, "ID book is required")
	}

	newQty, err := h.bookService.UpdateBookQuantity(ctx, req.GetBookId(), req.GetChangeAmount())
	if err != nil {
		h.logger.Error("Failed to update book quantity", err, logger.Fields{
			"book_id":       req.BookId,
			"change_amount": req.ChangeAmount,
		})
		return nil, err
	}

	return &bookv1.UpdateBookQuantityResponse{
		Success:              true,
		Message:              "Book quantity updated successfully",
		NewAvailableQuantity: newQty,
	}, nil
}

/* HELPER METHODS */
func toPbBook(b *models.Book) *bookv1.Book {
	return &bookv1.Book{
		Id:                b.ID,
		Title:             b.Title,
		Author:            b.Author,
		Isbn:              b.ISBN,
		Category:          b.Category,
		Description:       b.Description,
		TotalQuantity:     b.TotalQuantity,
		AvailableQuantity: b.AvailableQuantity,
		CreatedAt:         timestamppb.New(b.CreatedAt),
		UpdatedAt:         timestamppb.New(b.UpdatedAt),
	}
}
