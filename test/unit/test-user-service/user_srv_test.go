package main

import (
	"context"
	"testing"
	"time"

	"github.com/DgHnG36/lib-management-system/services/user-service/internal/applications"
	"github.com/DgHnG36/lib-management-system/services/user-service/internal/models"
	"github.com/DgHnG36/lib-management-system/services/user-service/pkg/logger"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MockUserRepository struct {
	users map[string]*models.User
}

func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		users: make(map[string]*models.User),
	}
}

func (m *MockUserRepository) Create(ctx context.Context, user *models.User) error {
	m.users[user.ID] = user
	return nil
}

func (m *MockUserRepository) FindByID(ctx context.Context, id string) (*models.User, error) {
	return m.users[id], nil
}

func (m *MockUserRepository) FindByUsername(ctx context.Context, username string) (*models.User, error) {
	for _, user := range m.users {
		if user.Username == username {
			return user, nil
		}
	}
	return nil, nil
}

func (m *MockUserRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, nil
}

func (m *MockUserRepository) Update(ctx context.Context, user *models.User) error {
	m.users[user.ID] = user
	return nil
}

func (m *MockUserRepository) UpdateVIPStatus(ctx context.Context, id string, isVip bool) error {
	if user, exists := m.users[id]; exists {
		user.IsVip = isVip
		return nil
	}
	return nil
}

func (m *MockUserRepository) Delete(ctx context.Context, ids []string) error {
	for _, id := range ids {
		delete(m.users, id)
	}
	return nil
}

func (m *MockUserRepository) List(ctx context.Context, page, limit int32, sortBy string, isDesc bool, role models.UserRole) ([]*models.User, int32, error) {
	var users []*models.User
	if role == "" || role == "ROLE_UNSPECIFIED" {
		for _, user := range m.users {
			users = append(users, user)
		}
	} else {
		for _, user := range m.users {
			if user.Role == role {
				users = append(users, user)
			}
		}
	}
	return users, int32(len(users)), nil
}

func TestUserService_Register_Success(t *testing.T) {
	repo := NewMockUserRepository()
	svc := applications.NewUserService(repo, []byte("test-secret"), "HS256", 60*time.Minute, logger.DefaultNewLogger())

	ctx := context.Background()
	user, err := svc.Register(ctx, "testuser", "password123", "test@example.com", "+1234567890")

	assert.NoError(t, err, "expected no error during registration")
	assert.NotNil(t, user, "expected user to be returned")
	assert.Equal(t, "testuser", user.Username)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "+1234567890", user.PhoneNumber)
	assert.Equal(t, models.RoleRegisteredUser, user.Role, "expected role to be REGISTERED_USER")
	assert.True(t, user.IsActive, "expected user to be active")
	assert.False(t, user.IsVip, "expected user to not be VIP")
	assert.NotEmpty(t, user.ID, "expected user ID to be generated")
	assert.NotEmpty(t, user.Password, "expected password to be hashed")
	assert.NotEqual(t, "password123", user.Password, "expected password to be hashed, not plain text")
}

func TestUserService_Register_PasswordHashed(t *testing.T) {
	repo := NewMockUserRepository()
	svc := applications.NewUserService(repo, []byte("test-secret"), "HS256", 60*time.Minute, logger.DefaultNewLogger())

	ctx := context.Background()
	plainPassword := "securePassword123!"
	user, err := svc.Register(ctx, "testuser", plainPassword, "test@example.com", "+1234567890")

	assert.NoError(t, err)
	assert.NotNil(t, user)

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(plainPassword))
	assert.NoError(t, err, "expected password hash to match provided password")
}

func TestUserService_Register_DuplicateUsername(t *testing.T) {
	repo := NewMockUserRepository()
	svc := applications.NewUserService(repo, []byte("test-secret"), "HS256", 60*time.Minute, logger.DefaultNewLogger())

	ctx := context.Background()

	_, err := svc.Register(ctx, "testuser", "password123", "test1@example.com", "+1234567890")
	assert.NoError(t, err)

	user, err := svc.Register(ctx, "testuser", "password456", "test2@example.com", "+9876543210")
	assert.Error(t, err, "expected error when registering duplicate username")
	assert.Nil(t, user, "expected user to be nil")

	st, ok := status.FromError(err)
	assert.True(t, ok, "expected gRPC status error")
	assert.Equal(t, codes.AlreadyExists, st.Code(), "expected AlreadyExists error code")
}

func TestUserService_Register_DuplicateEmail(t *testing.T) {
	repo := NewMockUserRepository()
	svc := applications.NewUserService(repo, []byte("test-secret"), "HS256", 60*time.Minute, logger.DefaultNewLogger())

	ctx := context.Background()

	_, err := svc.Register(ctx, "user1", "password123", "shared@example.com", "+1234567890")
	assert.NoError(t, err)

	user, err := svc.Register(ctx, "user2", "password456", "shared@example.com", "+9876543210")
	assert.Error(t, err, "expected error when registering duplicate email")
	assert.Nil(t, user, "expected user to be nil")

	st, ok := status.FromError(err)
	assert.True(t, ok, "expected gRPC status error")
	assert.Equal(t, codes.AlreadyExists, st.Code(), "expected AlreadyExists error code")
}

func TestUserService_Register_TimestampsSet(t *testing.T) {
	repo := NewMockUserRepository()
	svc := applications.NewUserService(repo, []byte("test-secret"), "HS256", 60*time.Minute, logger.DefaultNewLogger())

	ctx := context.Background()
	beforeTime := time.Now().UTC()
	user, err := svc.Register(ctx, "testuser", "password123", "test@example.com", "+1234567890")
	afterTime := time.Now().UTC()

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.False(t, user.CreatedAt.IsZero(), "expected CreatedAt to be set")
	assert.False(t, user.LastUpdated.IsZero(), "expected LastUpdated to be set")
	assert.True(t, user.CreatedAt.After(beforeTime) || user.CreatedAt.Equal(beforeTime), "expected CreatedAt to be after or equal to before time")
	assert.True(t, user.CreatedAt.Before(afterTime) || user.CreatedAt.Equal(afterTime), "expected CreatedAt to be before or equal to after time")
}

func TestUserService_Register_EmptyUsername(t *testing.T) {
	repo := NewMockUserRepository()
	svc := applications.NewUserService(repo, []byte("test-secret"), "HS256", 60*time.Minute, logger.DefaultNewLogger())

	ctx := context.Background()
	user, err := svc.Register(ctx, "", "password123", "test@example.com", "+1234567890")

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "", user.Username, "noting that empty username is currently allowed (should be fixed)")
}

func TestUserService_Register_PhoneNumberOptional(t *testing.T) {
	repo := NewMockUserRepository()
	svc := applications.NewUserService(repo, []byte("test-secret"), "HS256", 60*time.Minute, logger.DefaultNewLogger())

	ctx := context.Background()
	user, err := svc.Register(ctx, "testuser", "password123", "test@example.com", "")

	assert.NoError(t, err, "expected registration to succeed without phone number")
	assert.NotNil(t, user)
	assert.Equal(t, "", user.PhoneNumber, "expected phone number to be empty")
}
