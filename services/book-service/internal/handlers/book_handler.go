package handlers

import (
	"context"

	"github.com/DgHnG36/lib-management-system/services/book-service/internal/applications"
	"github.com/DgHnG36/lib-management-system/services/book-service/internal/models"
	"github.com/DgHnG36/lib-management-system/services/book-service/pkg/logger"
	commonv1 "github.com/DgHnG36/lib-management-system/shared/go/v1"
	bookv1 "github.com/DgHnG36/lib-management-system/shared/go/v1/book"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type BookHandler struct {
	bookv1.UnimplementedBookServiceServer
	bookSvc *applications.BookService
	logger  *logger.Logger
}

func NewBookHandler(bookSvc *applications.BookService, logger *logger.Logger) *BookHandler {
	return &BookHandler{
		bookSvc: bookSvc,
		logger:  logger,
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
		return nil, status.Errorf(codes.InvalidArgument, "id or title is required")
	}

	book, err := h.bookSvc.GetBook(ctx, id, title)
	if err != nil {
		h.logger.Error("Failed to get book", err)
		return nil, err
	}

	return &bookv1.BookResponse{Book: toPbBook(book)}, nil
}

func (h *BookHandler) ListBooks(ctx context.Context, req *bookv1.ListBooksRequest) (*bookv1.ListBooksResponse, error) {
	var page, limit int32 = 1, 10
	var sortBy string
	var isDesc bool

	if req.GetPagination() != nil {
		page = req.GetPagination().GetPage()
		limit = req.GetPagination().GetLimit()
		sortBy = req.GetPagination().GetSortBy()
		isDesc = req.GetPagination().GetIsDesc()
	}

	books, total, err := h.bookSvc.ListBooks(ctx, page, limit, sortBy, isDesc, req.GetSearchQuery(), req.GetCategory())
	if err != nil {
		h.logger.Error("Failed to list books", err)
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
	if len(req.GetBooks()) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "at least one book is required")
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

	books, err := h.bookSvc.CreateBooks(ctx, payloads)
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
	if req.GetId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "id is required")
	}

	book, err := h.bookSvc.UpdateBook(ctx,
		req.GetId(),
		req.GetTitle(),
		req.GetAuthor(),
		req.GetIsbn(),
		req.GetCategory(),
		req.GetDescription(),
	)
	if err != nil {
		h.logger.Error("Failed to update book", err)
		return nil, err
	}

	return &bookv1.BookResponse{Book: toPbBook(book)}, nil
}

func (h *BookHandler) DeleteBooks(ctx context.Context, req *bookv1.DeleteBooksRequest) (*commonv1.BaseResponse, error) {
	if len(req.GetIds()) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "ids are required")
	}

	if err := h.bookSvc.DeleteBooks(ctx, req.GetIds()); err != nil {
		h.logger.Error("Failed to delete books", err)
		return nil, err
	}

	return &commonv1.BaseResponse{
		Status:  200,
		Message: "Books deleted successfully",
	}, nil
}

func (h *BookHandler) CheckAvailability(ctx context.Context, req *bookv1.CheckAvailabilityRequest) (*bookv1.CheckAvailabilityResponse, error) {
	if req.GetBookId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "book_id is required")
	}

	isAvailable, qty, err := h.bookSvc.CheckAvailability(ctx, req.GetBookId())
	if err != nil {
		h.logger.Error("Failed to check availability", err)
		return nil, err
	}

	return &bookv1.CheckAvailabilityResponse{
		IsAvailable:       isAvailable,
		AvailableQuantity: qty,
	}, nil
}

func (h *BookHandler) UpdateBookQuantity(ctx context.Context, req *bookv1.UpdateBookQuantityRequest) (*bookv1.UpdateBookQuantityResponse, error) {
	if req.GetBookId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "book_id is required")
	}

	newQty, err := h.bookSvc.UpdateBookQuantity(ctx, req.GetBookId(), req.GetChangeAmount())
	if err != nil {
		h.logger.Error("Failed to update book quantity", err)
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
