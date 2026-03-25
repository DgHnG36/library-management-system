package models

import internal "github.com/DgHnG36/lib-management-system/services/user-service/internal/models"

type UserRole = internal.UserRole

type User = internal.User

const (
	RoleGuest          = internal.RoleGuest
	RoleRegisteredUser = internal.RoleRegisteredUser
	RoleManager        = internal.RoleManager
	RoleAdmin          = internal.RoleAdmin
)
