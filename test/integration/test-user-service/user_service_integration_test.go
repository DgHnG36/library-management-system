package integration

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/DgHnG36/lib-management-system/services/user-service/internal/applications"
	"github.com/DgHnG36/lib-management-system/services/user-service/internal/handlers"
	"github.com/DgHnG36/lib-management-system/services/user-service/internal/models"
	"github.com/DgHnG36/lib-management-system/services/user-service/internal/repository"
	"github.com/DgHnG36/lib-management-system/services/user-service/pkg/logger"
	commonv1 "github.com/DgHnG36/lib-management-system/shared/go/v1"
	userv1 "github.com/DgHnG36/lib-management-system/shared/go/v1/user"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

/* HELPER METHODS */
func setupTestDB(t *testing.T) *gorm.DB {
	host := os.Getenv("TEST_USER_DB_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("TEST_USER_DB_PORT")
	if port == "" {
		port = "15432"
	}
	dbName := os.Getenv("TEST_USER_DB_NAME")
	if dbName == "" {
		dbName = "user_db"
	}
	user := os.Getenv("TEST_USER_DB_USER")
	if user == "" {
		user = "postgres"
	}
	password := os.Getenv("TEST_USER_DB_PASSWORD")
	if password == "" {
		password = "postgres"
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable", host, user, password, dbName, port)

	var db *gorm.DB
	var err error

	for i := 0; i < 5; i++ {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}
	if !assert.NoError(t, err, "failed to connect to test database. check docker is running and TEST_USER_DB_* envs") {
		t.FailNow()
	}

	err = db.AutoMigrate(&models.User{})
	if !assert.NoError(t, err, "failed to migrate test database") {
		t.FailNow()
	}

	return db
}

func setupUserService(t *testing.T) (*handlers.UserHandler, repository.UserRepository) {
	db := setupTestDB(t)

	tx := db.Begin()
	if !assert.NoError(t, tx.Error, "failed to begin test transaction") {
		t.FailNow()
	}
	if !assert.NoError(t, tx.Exec("TRUNCATE TABLE users RESTART IDENTITY CASCADE").Error, "failed to reset users table for test transaction") {
		t.FailNow()
	}
	t.Cleanup(func() {
		_ = tx.Rollback().Error
	})

	log := logger.DefaultNewLogger()
	repo := repository.NewUserRepo(tx)
	srv := applications.NewUserService(repo, []byte("integration-test-secret"), "HS256", 60*time.Minute, log)
	handler := handlers.NewUserHandler(srv, log)
	return handler, repo
}

func TestUserService_Register_Success(t *testing.T) {
	handler, repo := setupUserService(t)
	ctx := context.Background()

	username := "integration_register_user"
	email := "integration_register_user@example.com"

	req := &userv1.RegisterRequest{
		Username:    username,
		Password:    "password123",
		Email:       email,
		PhoneNumber: "+84901234567",
	}

	resp, err := handler.Register(ctx, req)
	if !assert.NoError(t, err) {
		return
	}
	assert.NotNil(t, resp)
	assert.Equal(t, int32(http.StatusOK), resp.GetStatus())
	assert.Equal(t, "User registered successfully", resp.GetMessage())
	assert.NotEmpty(t, resp.GetUserId())

	createdUser, findErr := repo.FindByUsername(ctx, username)
	assert.NoError(t, findErr)
	assert.NotNil(t, createdUser)
	assert.Equal(t, resp.GetUserId(), createdUser.ID)
	assert.Equal(t, email, createdUser.Email)
	assert.NotEqual(t, "password123", createdUser.Password)
}

func TestUserService_Register_DuplicateUsername(t *testing.T) {
	handler, _ := setupUserService(t)
	ctx := context.Background()

	firstReq := &userv1.RegisterRequest{
		Username:    "dup_user",
		Password:    "password123",
		Email:       "dup_user_1@example.com",
		PhoneNumber: "+84909990001",
	}
	_, firstErr := handler.Register(ctx, firstReq)
	assert.NoError(t, firstErr)

	secondReq := &userv1.RegisterRequest{
		Username:    "dup_user",
		Password:    "password456",
		Email:       "dup_user_2@example.com",
		PhoneNumber: "+84909990002",
	}
	resp, err := handler.Register(ctx, secondReq)
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestUserService_Register_WithoutPhoneNumber_Success(t *testing.T) {
	handler, repo := setupUserService(t)
	ctx := context.Background()

	username := "register_without_phone"
	email := "register_without_phone@example.com"

	req := &userv1.RegisterRequest{
		Username: username,
		Password: "password123",
		Email:    email,
	}

	resp, err := handler.Register(ctx, req)
	if !assert.NoError(t, err) {
		return
	}
	if !assert.NotNil(t, resp) {
		return
	}

	assert.Equal(t, int32(http.StatusOK), resp.GetStatus())
	assert.Equal(t, "User registered successfully", resp.GetMessage())
	assert.NotEmpty(t, resp.GetUserId())

	createdUser, findErr := repo.FindByUsername(ctx, username)
	assert.NoError(t, findErr)
	if !assert.NotNil(t, createdUser) {
		return
	}
	assert.Equal(t, email, createdUser.Email)
	assert.Equal(t, "", createdUser.PhoneNumber)
}

func TestUserService_Register_MissingField(t *testing.T) {
	handler, _ := setupUserService(t)
	ctx := context.Background()

	req := &userv1.RegisterRequest{
		Username:    "",
		Password:    "password123",
		Email:       "missing_field@example.com",
		PhoneNumber: "+84909990003",
	}
	resp, err := handler.Register(ctx, req)
	assert.Error(t, err, "expected error when registering with missing username")
	assert.Nil(t, resp)

	req = &userv1.RegisterRequest{
		Username:    "missing_password",
		Password:    "",
		Email:       "missing_password@example.com",
		PhoneNumber: "+84909990004",
	}
	resp, err = handler.Register(ctx, req)
	assert.Error(t, err, "expected error when registering with missing password")
	assert.Nil(t, resp)

	req = &userv1.RegisterRequest{
		Username:    "missing_email",
		Password:    "password123",
		Email:       "",
		PhoneNumber: "+84909990005",
	}
	resp, err = handler.Register(ctx, req)
	assert.Error(t, err, "expected error when registering with missing email")
	assert.Nil(t, resp)
}

func TestUserService_Login_FailedUsername(t *testing.T) {
	handler, _ := setupUserService(t)
	ctx := context.Background()

	req := &userv1.RegisterRequest{
		Username:    "regis-user",
		Password:    "password123",
		Email:       "regis-user@test.com",
		PhoneNumber: "+84909990006",
	}
	_, err := handler.Register(ctx, req)
	assert.NoError(t, err)

	loginReq := &userv1.LoginRequest{
		Identifier: &userv1.LoginRequest_Username{
			Username: "Regis-user",
		},
		Password: "password123",
	}
	resp, err := handler.Login(ctx, loginReq)
	assert.Error(t, err, "expected error when logging in with non-existent username")
	assert.Nil(t, resp)
}

func TestUserService_Login_FailedEmail(t *testing.T) {
	handler, _ := setupUserService(t)
	ctx := context.Background()

	req := &userv1.RegisterRequest{
		Username:    "regis-email-user",
		Password:    "password123",
		Email:       "regis_user@test.com",
		PhoneNumber: "+84909990007",
	}

	_, err := handler.Register(ctx, req)
	assert.NoError(t, err)

	loginReq := &userv1.LoginRequest{
		Identifier: &userv1.LoginRequest_Email{
			Email: "Regis_user@test.com",
		},
		Password: "password123",
	}
	resp, err := handler.Login(ctx, loginReq)
	assert.Error(t, err, "expected error when logging in with non-existent email")
	assert.Nil(t, resp)
}

func TestUserService_Login_FailedPassword(t *testing.T) {
	handler, _ := setupUserService(t)

	ctx := context.Background()

	req := &userv1.RegisterRequest{
		Username:    "regis-password-user",
		Password:    "password123",
		Email:       "regis_password_user@test.com",
		PhoneNumber: "+84909990008",
	}

	_, err := handler.Register(ctx, req)
	assert.NoError(t, err)

	loginReq := &userv1.LoginRequest{
		Identifier: &userv1.LoginRequest_Username{
			Username: "regis-password-user",
		},
		Password: "wrong_password",
	}
	resp, err := handler.Login(ctx, loginReq)
	assert.Error(t, err, "expected error when logging in with incorrect password")
	assert.Nil(t, resp)
}

func TestUserService_Login_SQLInjection(t *testing.T) {
	handler, _ := setupUserService(t)
	ctx := context.Background()

	loginReq := &userv1.LoginRequest{
		Identifier: &userv1.LoginRequest_Username{
			Username: "' OR '1'='1 - --",
		},
		Password: "password123",
	}
	resp, err := handler.Login(ctx, loginReq)
	assert.Error(t, err, "expected error when logging in with SQL injection")
	assert.Nil(t, resp)
}

func TestUserService_GetProfile_Success(t *testing.T) {
	handler, _ := setupUserService(t)
	ctx := context.Background()

	registerReq := &userv1.RegisterRequest{
		Username:    "get_profile_user",
		Password:    "password123",
		Email:       "get_profile_user@example.com",
		PhoneNumber: "+84909990111",
	}

	registerResp, registerErr := handler.Register(ctx, registerReq)
	if !assert.NoError(t, registerErr) {
		return
	}
	if !assert.NotNil(t, registerResp) {
		return
	}

	profileResp, err := handler.GetProfile(ctx, &userv1.GetProfileRequest{Id: registerResp.GetUserId()})
	if !assert.NoError(t, err) {
		return
	}
	if !assert.NotNil(t, profileResp) {
		return
	}
	if !assert.NotNil(t, profileResp.GetUser()) {
		return
	}

	assert.Equal(t, registerResp.GetUserId(), profileResp.GetUser().GetId())
	assert.Equal(t, registerReq.GetUsername(), profileResp.GetUser().GetUsername())
	assert.Equal(t, registerReq.GetEmail(), profileResp.GetUser().GetEmail())
	assert.Equal(t, registerReq.GetPhoneNumber(), profileResp.GetUser().GetPhoneNumber())
}

func TestUserService_GetProfile_NotFound(t *testing.T) {
	handler, _ := setupUserService(t)
	ctx := context.Background()

	profileResp, err := handler.GetProfile(ctx, &userv1.GetProfileRequest{Id: "81c11570-c367-44c7-8253-9a46c45d3b67"})
	assert.Error(t, err)
	assert.Nil(t, profileResp)

	st, ok := status.FromError(err)
	assert.True(t, ok, "expected gRPC status error")
	assert.Equal(t, codes.NotFound, st.Code(), "expected NotFound error code")
}

func TestUserService_UpdateProfile_MissingField(t *testing.T) {
	handler, _ := setupUserService(t)

	ctx := context.Background()

	userRegister := &userv1.RegisterRequest{
		Username: "update_profile_user",
		Password: "password123",
		Email:    "updateUser@test.com",
	}

	resp, err := handler.Register(ctx, userRegister)
	assert.NoError(t, err)
	assert.NotNil(t, resp)

	updateReq := &userv1.UpdateProfileRequest{
		Id:       "",
		Username: "",
		Email:    "",
	}

	_, err = handler.UpdateProfile(ctx, updateReq)
	st, ok := status.FromError(err)
	assert.True(t, ok, "expected gRPC status error")
	assert.Equal(t, codes.InvalidArgument, st.Code(), "expected InvalidArgument code")
}

func TestUserService_UpdateProfile_Success(t *testing.T) {
	handler, _ := setupUserService(t)

	ctx := context.Background()

	userRegister := &userv1.RegisterRequest{
		Username: "update_profile_user2",
		Password: "password123",
		Email:    "update_profile_user2@example.com",
	}

	resp, err := handler.Register(ctx, userRegister)
	assert.NoError(t, err)
	assert.NotNil(t, resp)

	userLogin := &userv1.LoginRequest{
		Identifier: &userv1.LoginRequest_Username{
			Username: "update_profile_user2",
		},
		Password: "password123",
	}
	loginResp, err := handler.Login(ctx, userLogin)
	assert.NoError(t, err)
	assert.NotNil(t, loginResp)
	assert.NotEmpty(t, loginResp.GetToken())

	updateReq := &userv1.UpdateProfileRequest{
		Id:       resp.GetUserId(),
		Username: "updated_username",
		Email:    "updated_email@example.com",
	}

	updateResp, err := handler.UpdateProfile(ctx, updateReq)
	assert.NoError(t, err)
	assert.NotNil(t, updateResp)
	assert.NotNil(t, updateResp.GetUser())
	assert.Equal(t, resp.GetUserId(), updateResp.GetUser().GetId())
	assert.Equal(t, updateReq.GetUsername(), updateResp.GetUser().GetUsername())
	assert.Equal(t, updateReq.GetEmail(), updateResp.GetUser().GetEmail())
}

func TestUserService_UpdateVIPAccount_Success(t *testing.T) {
	handler, _ := setupUserService(t)
	ctx := context.Background()

	userRegister := &userv1.RegisterRequest{
		Username: "vip_user",
		Password: "password123",
		Email:    "vip_user@example.com",
	}
	resp, err := handler.Register(ctx, userRegister)
	assert.NoError(t, err)
	assert.NotNil(t, resp)

	updateReq := &userv1.UpdateVIPAccountRequest{
		Id:    resp.GetUserId(),
		IsVip: true,
	}

	updateResp, err := handler.UpdateVIPAccount(ctx, updateReq)
	assert.NoError(t, err)
	assert.NotNil(t, updateResp)
	assert.True(t, updateResp.GetCurrentVipStatus(), "expected VIP status to be true after update")
}

func TestUserService_ListUsers_Success(t *testing.T) {
	handler, _ := setupUserService(t)
	ctx := context.Background()

	// Create multiple users
	for i := 1; i <= 3; i++ {
		req := &userv1.RegisterRequest{
			Username: fmt.Sprintf("list_user_%d", i),
			Password: "password123",
			Email:    fmt.Sprintf("list_user_%d@example.com", i),
		}
		_, err := handler.Register(ctx, req)
		assert.NoError(t, err)
	}

	listReq := &userv1.ListUsersRequest{
		Pagination: &commonv1.PaginationRequest{
			Page:   1,
			Limit:  10,
			SortBy: "created_at",
		},
	}

	listResp, err := handler.ListUsers(ctx, listReq)
	assert.NoError(t, err)
	assert.NotNil(t, listResp)
	assert.Len(t, listResp.GetUsers(), 3, "expected to retrieve 3 users")
}

func TestUserService_ListUsers_Empty(t *testing.T) {
	handler, _ := setupUserService(t)
	ctx := context.Background()

	listReq := &userv1.ListUsersRequest{
		Pagination: &commonv1.PaginationRequest{
			Page:   1,
			Limit:  10,
			SortBy: "created_at",
		},
	}

	listResp, err := handler.ListUsers(ctx, listReq)
	assert.NoError(t, err)
	assert.NotNil(t, listResp)
	assert.Len(t, listResp.GetUsers(), 0, "expected to retrieve 0 users from empty database")
}

func TestUserService_Delete_Success(t *testing.T) {
	handler, _ := setupUserService(t)
	ctx := context.Background()

	userRegister := &userv1.RegisterRequest{
		Username: "delete_user",
		Password: "password123",
		Email:    "delete_user@example.com",
	}
	resp, err := handler.Register(ctx, userRegister)
	assert.NoError(t, err)
	assert.NotNil(t, resp)

	deleteReq := &userv1.DeleteUsersRequest{
		Ids: []string{resp.GetUserId()},
	}

	_, err = handler.DeleteUsers(ctx, deleteReq)
	assert.NoError(t, err)
}
