package models

import "time"

type UserRole string

const (
	RoleGuest          UserRole = "GUEST"
	RoleRegisteredUser UserRole = "REGISTERED_USER"
	RoleManager        UserRole = "MANAGER"
	RoleAdmin          UserRole = "ADMIN"
)

type User struct {
	ID          string   `gorm:"type:uuid;primaryKey"`
	Username    string   `gorm:"type:varchar(100);uniqueIndex;not null"`
	Password    string   `gorm:"type:varchar(255);not null"`
	Email       string   `gorm:"type:varchar(255);uniqueIndex;not null"`
	PhoneNumber string   `gorm:"type:varchar(20)"`
	Role        UserRole `gorm:"type:varchar(30);not null;default:'REGISTERED_USER'"`
	IsVip       bool     `gorm:"default:false"`
	IsActive    bool     `gorm:"default:true"`
	CreatedAt   time.Time
	LastUpdated time.Time `gorm:"column:last_updated;autoUpdateTime"`
}

func (User) TableName() string {
	return "users"
}
