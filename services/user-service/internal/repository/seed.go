package repository

import (
	"context"
	"time"

	"github.com/DgHnG36/lib-management-system/services/user-service/internal/models"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type seedUser struct {
	Username string
	Password string
	Email    string
	Role     models.UserRole
}

func SeedDefaultUsers(db *gorm.DB) error {
	seeds := []seedUser{
		{
			Username: "lms-manager",
			Password: "manager@413",
			Email:    "lms-manager@lms.local",
			Role:     models.RoleManager,
		},
		{
			Username: "lms-admin",
			Password: "@dm1n79",
			Email:    "lms-admin@lms.local",
			Role:     models.RoleAdmin,
		},
	}

	ctx := context.Background()
	for _, s := range seeds {
		var existing models.User
		err := db.WithContext(ctx).First(&existing, "username = ?", s.Username).Error
		if err == nil {
			// already exists, skip
			continue
		}
		if err != gorm.ErrRecordNotFound {
			return err
		}

		hashed, err := bcrypt.GenerateFromPassword([]byte(s.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		user := &models.User{
			ID:          uuid.New().String(),
			Username:    s.Username,
			Password:    string(hashed),
			Email:       s.Email,
			Role:        s.Role,
			IsVip:       false,
			IsActive:    true,
			CreatedAt:   time.Now().UTC(),
			LastUpdated: time.Now().UTC(),
		}
		if err := db.WithContext(ctx).Create(user).Error; err != nil {
			return err
		}
	}
	return nil
}
