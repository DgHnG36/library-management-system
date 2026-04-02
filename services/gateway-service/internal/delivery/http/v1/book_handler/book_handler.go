package book_handler

import (
	"fmt"

	"github.com/DgHnG36/lib-management-system/services/gateway-service/internal/clients/book_service_client"
	"github.com/DgHnG36/lib-management-system/services/gateway-service/internal/dto/book_service_dto"
	"github.com/DgHnG36/lib-management-system/services/gateway-service/internal/dto/mapper"
	pkgerrors "github.com/DgHnG36/lib-management-system/services/gateway-service/pkg/errors"
	"github.com/DgHnG36/lib-management-system/services/gateway-service/pkg/logger"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/connectivity"
)

type BookHandler struct {
	bookServiceClient *book_service_client.BookServiceClient
	mapper            mapper.MapperInterface
	logger            *logger.Logger
}

func NewBookHandler(addr string, log *logger.Logger) *BookHandler {
	bookServiceClient, err := book_service_client.NewBookServiceClient(addr, log)
	if err != nil {
		log.Fatal("Failed to create book service client", err, logger.Fields{
			"address": addr,
		})
		return nil
	}

	mapper := mapper.NewMapper()

	return &BookHandler{
		bookServiceClient: bookServiceClient,
		mapper:            mapper,
		logger:            log,
	}
}

func (h *BookHandler) Close() {
	if h.bookServiceClient != nil && h.bookServiceClient.GetConnection() != nil {
		if err := h.bookServiceClient.GetConnection().Close(); err != nil {
			h.logger.Error("Failed to close book service client connection", err)
		}
	}
}

func (h *BookHandler) GetBook(c *gin.Context) {
	var req book_service_dto.GetBookRequestDTO
	if err := c.ShouldBindUri(&req); err != nil {
		h.logger.Error("Failed to bind get book request", err)
		c.JSON(400, gin.H{
			"error": "Invalid request parameters",
		})
		return
	}

	isTitle := h.identifyTitle(req.Identifier)
	grpcReq := h.mapper.MapPbGetBookRequest(&req, isTitle)
	resp, err := h.bookServiceClient.GetBook(c.Request.Context(), grpcReq)
	if err != nil {
		h.logger.Error("Failed to get book", err, logger.Fields{
			"identifier": req.Identifier,
		})
		appErr := pkgerrors.FromGRPCError(err)
		c.JSON(int(appErr.HTTPStatus), appErr)
		return
	}

	httpResp := h.mapper.MapDTOGetBookResponse(resp)
	c.JSON(200, httpResp)
}

func (h *BookHandler) ListBooks(c *gin.Context) {
	var req book_service_dto.ListBooksRequestDTO
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Error("Failed to bind list books request", err)
		c.JSON(400, gin.H{
			"error": "Invalid query parameters",
		})
		return
	}

	grpcReq := h.mapper.MapPbListBooksRequest(&req)
	resp, err := h.bookServiceClient.ListBooks(c.Request.Context(), grpcReq)
	if err != nil {
		h.logger.Error("Failed to list books", err, logger.Fields{
			"search_query": req.SearchQuery,
			"category":     req.Category,
		})
		appErr := pkgerrors.FromGRPCError(err)
		c.JSON(int(appErr.HTTPStatus), appErr)
		return
	}

	httpResp := h.mapper.MapDTOListBooksResponse(resp)
	c.JSON(200, httpResp)
}

func (h *BookHandler) CreateBooks(c *gin.Context) {
	var req book_service_dto.CreateBooksRequestDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind create books request", err)
		c.JSON(400, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	grpcReq := h.mapper.MapPbCreateBooksRequest(&req)
	resp, err := h.bookServiceClient.CreateBooks(c.Request.Context(), grpcReq)
	if err != nil {
		h.logger.Error("Failed to create books", err)
		appErr := pkgerrors.FromGRPCError(err)
		c.JSON(int(appErr.HTTPStatus), appErr)
		return
	}

	httpResp := h.mapper.MapDTOCreateBooksResponse(resp)
	c.JSON(201, httpResp)
}

func (h *BookHandler) UpdateBook(c *gin.Context) {
	var req book_service_dto.UpdateBookRequestDTO
	if err := c.ShouldBindUri(&req); err != nil {
		h.logger.Error("Failed to bind update book request (uri)", err)
		c.JSON(400, gin.H{
			"error": "Invalid request parameters",
		})
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind update book request (body)", err)
		c.JSON(400, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	grpcReq := h.mapper.MapPbUpdateBookRequest(&req)
	resp, err := h.bookServiceClient.UpdateBook(c.Request.Context(), grpcReq)
	if err != nil {
		h.logger.Error("Failed to update book", err, logger.Fields{
			"book_id": req.ID,
		})
		appErr := pkgerrors.FromGRPCError(err)
		c.JSON(int(appErr.HTTPStatus), appErr)
		return
	}

	httpResp := h.mapper.MapDTOUpdateBookResponse(resp)
	c.JSON(200, httpResp)
}

func (h *BookHandler) DeleteBook(c *gin.Context) {
	bookID := c.Param("id")
	if bookID == "" {
		c.JSON(400, gin.H{
			"error": "Book ID is required",
		})
		return
	}

	req := book_service_dto.DeleteBooksRequestDTO{
		BookIDs: []string{bookID},
	}

	grpcReq := h.mapper.MapPbDeleteBooksRequest(&req)
	err := h.bookServiceClient.DeleteBooks(c.Request.Context(), grpcReq)
	if err != nil {
		h.logger.Error("Failed to delete book", err, logger.Fields{
			"book_id": bookID,
		})
		appErr := pkgerrors.FromGRPCError(err)
		c.JSON(int(appErr.HTTPStatus), appErr)
		return
	}

	c.JSON(204, nil)
}

func (h *BookHandler) CheckBookAvailability(c *gin.Context) {
	var req book_service_dto.CheckAvailabilityRequestDTO
	if err := c.ShouldBindUri(&req); err != nil {
		h.logger.Error("Failed to bind check book availability request", err)
		c.JSON(400, gin.H{
			"error": "Invalid request parameters",
		})
		return
	}

	grpcReq := h.mapper.MapPbCheckBookAvailabilityRequest(&req)
	resp, err := h.bookServiceClient.CheckAvailability(c.Request.Context(), grpcReq)
	if err != nil {
		h.logger.Error("Failed to check book availability", err, logger.Fields{
			"book_id": req.BookID,
		})
		appErr := pkgerrors.FromGRPCError(err)
		c.JSON(int(appErr.HTTPStatus), appErr)
		return
	}

	httpResp := h.mapper.MapDTOCheckBookAvailabilityResponse(resp)
	c.JSON(200, httpResp)
}

func (h *BookHandler) UpdateBookQuantity(c *gin.Context) {
	var req book_service_dto.UpdateBookQuantityRequestDTO
	if err := c.ShouldBindUri(&req); err != nil {
		h.logger.Error("Failed to bind update book quantity request (uri)", err)
		c.JSON(400, gin.H{
			"error": "Invalid request parameters",
		})
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind update book quantity request (body)", err)
		c.JSON(400, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	grpcReq := h.mapper.MapPbUpdateBookQuantityRequest(&req)
	resp, err := h.bookServiceClient.UpdateBookQuantity(c.Request.Context(), grpcReq)
	if err != nil {
		h.logger.Error("Failed to update book quantity", err, logger.Fields{
			"book_id":       req.BookID,
			"change_amount": req.ChangeAmount,
		})
		appErr := pkgerrors.FromGRPCError(err)
		c.JSON(int(appErr.HTTPStatus), appErr)
		return
	}

	httpResp := h.mapper.MapDTOUpdateBookQuantityResponse(resp)
	c.JSON(200, httpResp)
}

func (h *BookHandler) CheckConnection() (bool, error) {
	if h.bookServiceClient == nil || h.bookServiceClient.GetConnection() == nil {
		return false, fmt.Errorf("book service client is not initialized")
	}

	if h.bookServiceClient.GetConnection().GetState() != connectivity.Ready {
		return false, fmt.Errorf("book service connection is not ready")
	}

	return true, nil
}

/* HELPER METHODS */

func (h *BookHandler) identifyTitle(identifier string) bool {
	// TODO: Implement logic to identify if the identifier is a title
	return true // Placeholder, should be replaced with actual logic
}
