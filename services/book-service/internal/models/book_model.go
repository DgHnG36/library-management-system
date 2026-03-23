package models

import "time"

type Book struct {
	ID                string `gorm:"primaryKey,type:uuid"`
	Title             string `gorm:"type:varchar(255);not null;index"`
	Author            string `gorm:"type:varchar(255);not null"`
	ISBN              string `gorm:"type:varchar(20);uniqueIndex"`
	Category          string `gorm:"type:varchar(100);index"`
	Description       string `gorm:"type:text"`
	TotalQuantity     int32  `gorm:"not null;default:0"`
	AvailableQuantity int32  `gorm:"not null;default:0"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (Book) TableName() string {
	return "books"
}
