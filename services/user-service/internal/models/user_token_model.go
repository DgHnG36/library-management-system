package models

import "time"

type UserToken struct {
	ID               string    `gorm:"type:uuid,primaryKey"`
	UserID           string    `gorm:"type:uuid,not null"`
	RefreshTokenHash string    `gorm:"uniqueIndex;not null"`
	ExpiresAt        time.Time `gorm:"index"`
	CreatedAt        time.Time
}

func (UserToken) TableName() string {
	return "user_tokens"
}
