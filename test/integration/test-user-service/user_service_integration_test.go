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
	userv1 "github.com/DgHnG36/lib-management-system/shared/go/v1/user"
	"github.com/stretchr/testify/assert"
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
