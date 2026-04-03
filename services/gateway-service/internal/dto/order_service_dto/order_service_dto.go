package order_service_dto

import "github.com/DgHnG36/lib-management-system/services/gateway-service/internal/dto/common_dto"

type OrderDTO struct {
	ID            string  `json:"id"`
	UserID        string  `json:"user_id"`
	Status        string  `json:"status"`
	BorrowDate    string  `json:"borrow_date"`
	DueDate       string  `json:"due_date"`
	ReturnDate    *string `json:"return_date,omitempty"`
	Note          string  `json:"note,omitempty"`
	PenaltyAmount int32   `json:"penalty_amount"`
	CancelReason  string  `json:"cancel_reason,omitempty"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`

	Books []OrderBookDTO `json:"books"`
}

type OrderBookDTO struct {
	OrderID string `json:"order_id"`
	BookID  string `json:"book_id"`
}

type CreateOrderRequestDTO struct {
	UserID     string   // set from auth context
	BookIDs    []string `json:"book_ids" binding:"required,min=1,dive,required"`
	BorrowDays int32    `json:"borrow_days" binding:"required"`
}

type OrderResponseDTO struct {
	Order *OrderDTO `json:"order"`
}

type GetOrderRequestDTO struct {
	OrderID string `uri:"id" binding:"required"`
}

type ListMyOrdersRequestDTO struct {
	Pagination   *common_dto.PaginationDTO `form:",inline"`
	UserID       string                    // set from auth context
	FilterStatus string                    `form:"filter_status"`
}

type ListOrdersResponseDTO struct {
	Orders     []*OrderDTO `json:"orders"`
	TotalCount int32       `json:"total_count"`
}

type CancelOrderRequestDTO struct {
	OrderID      string `uri:"id" binding:"required"`
	UserID       string // set from auth context
	CancelReason string `json:"cancel_reason,omitempty"`
}

type ListAllOrdersRequestDTO struct {
	Pagination   *common_dto.PaginationDTO `form:",inline"`
	FilterStatus string                    `form:"filter_status"`
	SearchUserID string                    `form:"search_user_id"`
}

type UpdateOrderStatusRequestDTO struct {
	OrderID   string `uri:"id" binding:"required"`
	NewStatus string `json:"new_status"`
	Note      string `json:"note,omitempty"`
}
