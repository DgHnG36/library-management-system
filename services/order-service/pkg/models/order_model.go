package models

import internal "github.com/DgHnG36/lib-management-system/services/order-service/internal/models"

type OrderStatus = internal.OrderStatus

type Order = internal.Order

type OrderBook = internal.OrderBook

const (
	StatusUnspecified = internal.StatusUnspecified
	StatusPending     = internal.StatusPending
	StatusApproved    = internal.StatusApproved
	StatusBorrowed    = internal.StatusBorrowed
	StatusReturned    = internal.StatusReturned
	StatusCanceled    = internal.StatusCanceled
	StatusOverdue     = internal.StatusOverdue
)
