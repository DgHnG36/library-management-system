package models

import "time"

type OrderStatus string

const (
	StatusUnspecified OrderStatus = "STATUS_UNSPECIFIED"
	StatusPending     OrderStatus = "PENDING"
	StatusApproved    OrderStatus = "APPROVED"
	StatusBorrowed    OrderStatus = "BORROWED"
	StatusReturned    OrderStatus = "RETURNED"
	StatusCanceled    OrderStatus = "CANCELED"
	StatusOverdue     OrderStatus = "OVERDUE"
)

type Order struct {
	ID            string      `gorm:"type:uuid;primaryKey"`
	UserID        string      `gorm:"type:uuid;not null;index"`
	Status        OrderStatus `gorm:"type:varchar(30);not null;default:'PENDING'"`
	BorrowDate    time.Time   `gorm:"not null"`
	DueDate       time.Time   `gorm:"not null"`
	ReturnDate    *time.Time  `gorm:"default:null"`
	Note          string      `gorm:"type:text"`
	PenaltyAmount int32       `gorm:"default:0"`
	CancelReason  string      `gorm:"type:text"`
	CreatedAt     time.Time
	UpdatedAt     time.Time

	Books []OrderBook `gorm:"foreignKey:OrderID"`
}

type OrderBook struct {
	OrderID string `gorm:"type:uuid;primaryKey"`
	BookID  string `gorm:"type:uuid;primaryKey"`
}

func (Order) TableName() string {
	return "orders"
}

func (OrderBook) TableName() string {
	return "order_books"
}
