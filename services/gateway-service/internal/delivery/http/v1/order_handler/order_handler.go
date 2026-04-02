package order_handler

import (
	"fmt"

	"github.com/DgHnG36/lib-management-system/services/gateway-service/internal/clients/order_service_client"
	"github.com/DgHnG36/lib-management-system/services/gateway-service/internal/dto/mapper"
	"github.com/DgHnG36/lib-management-system/services/gateway-service/internal/dto/order_service_dto"
	pkgerrors "github.com/DgHnG36/lib-management-system/services/gateway-service/pkg/errors"
	"github.com/DgHnG36/lib-management-system/services/gateway-service/pkg/logger"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/connectivity"
)

type OrderHandler struct {
	orderServiceClient *order_service_client.OrderServiceClient
	mapper             mapper.MapperInterface
	logger             *logger.Logger
}

func NewOrderHandler(addr string, log *logger.Logger) *OrderHandler {
	orderServiceClient, err := order_service_client.NewOrderServiceClient(addr, log)
	if err != nil {
		log.Fatal("Failed to create order service client", err, logger.Fields{
			"address": addr,
		})
		return nil
	}

	mapper := mapper.NewMapper()

	return &OrderHandler{
		orderServiceClient: orderServiceClient,
		mapper:             mapper,
		logger:             log,
	}
}

func (h *OrderHandler) Close() {
	if h.orderServiceClient != nil && h.orderServiceClient.GetConnection() != nil {
		if err := h.orderServiceClient.GetConnection().Close(); err != nil {
			h.logger.Error("Failed to close order service client connection", err)
		}
	}
}

func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var req order_service_dto.CreateOrderRequestDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind create order request", err)
		c.JSON(400, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	req.UserID = c.GetString("X-User-ID")

	grpcReq := h.mapper.MapPbCreateOrderRequest(&req)
	resp, err := h.orderServiceClient.CreateOrder(c.Request.Context(), grpcReq)
	if err != nil {
		h.logger.Error("Failed to create order", err, logger.Fields{
			"user_id":  req.UserID,
			"book_ids": req.BookIDs,
		})
		appErr := pkgerrors.FromGRPCError(err)
		c.JSON(int(appErr.HTTPStatus), appErr)
		return
	}

	httpResp := h.mapper.MapDTOCreateOrderResponse(resp)
	c.JSON(201, httpResp)
}

func (h *OrderHandler) GetOrder(c *gin.Context) {
	var req order_service_dto.GetOrderRequestDTO
	if err := c.ShouldBindUri(&req); err != nil {
		h.logger.Error("Failed to bind get order request", err)
		c.JSON(400, gin.H{
			"error": "Invalid request parameters",
		})
		return
	}

	grpcReq := h.mapper.MapPbGetOrderRequest(&req)
	resp, err := h.orderServiceClient.GetOrder(c.Request.Context(), grpcReq)
	if err != nil {
		h.logger.Error("Failed to get order", err, logger.Fields{
			"order_id": req.OrderID,
		})
		appErr := pkgerrors.FromGRPCError(err)
		c.JSON(int(appErr.HTTPStatus), appErr)
		return
	}

	httpResp := h.mapper.MapDTOGetOrderResponse(resp)
	c.JSON(200, httpResp)
}

func (h *OrderHandler) ListMyOrders(c *gin.Context) {
	var req order_service_dto.ListMyOrdersRequestDTO
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Error("Failed to bind list my orders request", err)
		c.JSON(400, gin.H{
			"error": "Invalid query parameters",
		})
		return
	}

	req.UserID = c.GetString("X-User-ID")

	grpcReq := h.mapper.MapPbListMyOrdersRequest(&req)
	resp, err := h.orderServiceClient.ListMyOrders(c.Request.Context(), grpcReq)
	if err != nil {
		h.logger.Error("Failed to list my orders", err, logger.Fields{
			"user_id":       req.UserID,
			"filter_status": req.FilterStatus,
		})
		appErr := pkgerrors.FromGRPCError(err)
		c.JSON(int(appErr.HTTPStatus), appErr)
		return
	}

	httpResp := h.mapper.MapDTOListMyOrdersResponse(resp)
	c.JSON(200, httpResp)
}

func (h *OrderHandler) CancelOrder(c *gin.Context) {
	var req order_service_dto.CancelOrderRequestDTO
	if err := c.ShouldBindUri(&req); err != nil {
		h.logger.Error("Failed to bind cancel order request (uri)", err)
		c.JSON(400, gin.H{
			"error": "Invalid request parameters",
		})
		return
	}

	// cancel_reason is optional — ignore body binding errors
	_ = c.ShouldBindJSON(&req)

	req.UserID = c.GetString("X-User-ID")

	grpcReq := h.mapper.MapPbCancelOrderRequest(&req)
	resp, err := h.orderServiceClient.CancelOrder(c.Request.Context(), grpcReq)
	if err != nil {
		h.logger.Error("Failed to cancel order", err, logger.Fields{
			"order_id": req.OrderID,
			"user_id":  req.UserID,
		})
		appErr := pkgerrors.FromGRPCError(err)
		c.JSON(int(appErr.HTTPStatus), appErr)
		return
	}

	httpResp := h.mapper.MapDTOCancelOrderResponse(resp)
	c.JSON(200, httpResp)
}

func (h *OrderHandler) ListAllOrders(c *gin.Context) {
	var req order_service_dto.ListAllOrdersRequestDTO
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Error("Failed to bind list all orders request", err)
		c.JSON(400, gin.H{
			"error": "Invalid query parameters",
		})
		return
	}

	grpcReq := h.mapper.MapPbListAllOrdersRequest(&req)
	resp, err := h.orderServiceClient.ListAllOrders(c.Request.Context(), grpcReq)
	if err != nil {
		h.logger.Error("Failed to list all orders", err, logger.Fields{
			"filter_status":  req.FilterStatus,
			"search_user_id": req.SearchUserID,
		})
		appErr := pkgerrors.FromGRPCError(err)
		c.JSON(int(appErr.HTTPStatus), appErr)
		return
	}

	httpResp := h.mapper.MapDTOListAllOrdersResponse(resp)
	c.JSON(200, httpResp)
}

func (h *OrderHandler) UpdateOrderStatus(c *gin.Context) {
	var req order_service_dto.UpdateOrderStatusRequestDTO
	if err := c.ShouldBindUri(&req); err != nil {
		h.logger.Error("Failed to bind update order status request (uri)", err)
		c.JSON(400, gin.H{
			"error": "Invalid request parameters",
		})
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind update order status request (body)", err)
		c.JSON(400, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	grpcReq := h.mapper.MapPbUpdateOrderStatusRequest(&req)
	resp, err := h.orderServiceClient.UpdateOrderStatus(c.Request.Context(), grpcReq)
	if err != nil {
		h.logger.Error("Failed to update order status", err, logger.Fields{
			"order_id":   req.OrderID,
			"new_status": req.NewStatus,
		})
		appErr := pkgerrors.FromGRPCError(err)
		c.JSON(int(appErr.HTTPStatus), appErr)
		return
	}

	httpResp := h.mapper.MapDTOUpdateOrderStatusResponse(resp)
	c.JSON(200, httpResp)
}

func (h *OrderHandler) CheckConnection() (bool, error) {
	if h.orderServiceClient == nil || h.orderServiceClient.GetConnection() == nil {
		return false, fmt.Errorf("order service client is not initialized")
	}

	if h.orderServiceClient.GetConnection().GetState() != connectivity.Ready {
		return false, fmt.Errorf("order service client connection is not ready")
	}

	return true, nil
}
