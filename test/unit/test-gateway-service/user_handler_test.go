package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DgHnG36/lib-management-system/services/gateway-service/internal/delivery/http/v1/user_handler"
	"github.com/DgHnG36/lib-management-system/services/gateway-service/internal/dto/mapper"
	"github.com/DgHnG36/lib-management-system/services/gateway-service/pkg/logger"
	userv1 "github.com/DgHnG36/lib-management-system/shared/go/v1/user"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

/* MOCK USER SERVICE CLIENT — implements user_handler.UserClientInterface */

type Mock_UserServiceClient struct {
	mock.Mock
}

func NewMock_UserServiceClient() *Mock_UserServiceClient {
	return &Mock_UserServiceClient{}
}

func (m *Mock_UserServiceClient) Register(ctx context.Context, req *userv1.RegisterRequest) (*userv1.RegisterResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*userv1.RegisterResponse), args.Error(1)
}

func (m *Mock_UserServiceClient) Login(ctx context.Context, req *userv1.LoginRequest) (*userv1.LoginResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*userv1.LoginResponse), args.Error(1)
}

func (m *Mock_UserServiceClient) GetProfile(ctx context.Context, req *userv1.GetProfileRequest) (*userv1.UserProfileResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*userv1.UserProfileResponse), args.Error(1)
}

func (m *Mock_UserServiceClient) UpdateProfile(ctx context.Context, req *userv1.UpdateProfileRequest) (*userv1.UserProfileResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*userv1.UserProfileResponse), args.Error(1)
}

func (m *Mock_UserServiceClient) UpdateVIPAccount(ctx context.Context, req *userv1.UpdateVIPAccountRequest) (*userv1.UpdateVIPAccountResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*userv1.UpdateVIPAccountResponse), args.Error(1)
}

func (m *Mock_UserServiceClient) ListUsers(ctx context.Context, req *userv1.ListUsersRequest) (*userv1.ListUsersResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*userv1.ListUsersResponse), args.Error(1)
}

func (m *Mock_UserServiceClient) DeleteUsers(ctx context.Context, req *userv1.DeleteUsersRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *Mock_UserServiceClient) GetConnection() *grpc.ClientConn {
	return nil
}

/* SHARED FIXTURES */

var (
	testLog    = logger.DefaultNewLogger()
	testMapper = mapper.NewMapper()
)

func newTestHandler(mc *Mock_UserServiceClient) *user_handler.UserHandler {
	return user_handler.NewUserHandlerWithClient(mc, testMapper, testLog)
}

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	m.Run()
}

/* REGISTER */

func TestUserHandler_Register(t *testing.T) {
	tests := []struct {
		name           string
		inputBody      string
		setupMock      func(*Mock_UserServiceClient)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Successful registration — bind and response successfully",
			inputBody: `{
				"username": "user-register-1",
				"password": "pass-register-1",
				"email": "user-register-1@test.com",
				"phone_number": "0985761231"
			}`,
			setupMock: func(mc *Mock_UserServiceClient) {
				mc.On("Register", mock.Anything,
					mock.MatchedBy(func(req *userv1.RegisterRequest) bool {
						return req.Username == "user-register-1" &&
							req.Password == "pass-register-1" &&
							req.Email == "user-register-1@test.com" &&
							req.PhoneNumber == "0985761231"
					}),
				).Return(&userv1.RegisterResponse{UserId: "uid-user-register-1"}, nil)
			},
			expectedStatus: http.StatusCreated,
			expectedBody:   "uid-user-register-1",
		},
		{
			name: "Phone number is optional — bind successfully when not provided",
			inputBody: `{
				"username": "user-register-2",
				"password": "pass-register-2",
				"email": "user-register-2@test.com"
			}`,
			setupMock: func(mc *Mock_UserServiceClient) {
				mc.On("Register", mock.Anything,
					mock.MatchedBy(func(req *userv1.RegisterRequest) bool {
						return req.Username == "user-register-2" &&
							req.Email == "user-register-2@test.com" &&
							req.PhoneNumber == ""
					}),
				).Return(&userv1.RegisterResponse{UserId: "uid-user-register-2"}, nil)
			},
			expectedStatus: http.StatusCreated,
			expectedBody:   "uid-user-register-2",
		},
		{
			name: "Invalid JSON — bind fails and returns 400",
			inputBody: `{
				"username": "user-register-3",
				"password": "pass-register-3",
				"email": "
			}`,
			setupMock:      func(mc *Mock_UserServiceClient) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request payload",
		},
		{
			name: "Missing email (required) — bind validation fails and returns 400",
			inputBody: `{
				"username": "user-register-4",
				"password": "pass-register-4"
			}`,
			setupMock:      func(mc *Mock_UserServiceClient) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request payload",
		},
		{
			name: "Missing password (required) — bind validation fails and returns 400",
			inputBody: `{
				"username": "user-register-5",
				"email": "user-register-5@test.com"
			}`,
			setupMock:      func(mc *Mock_UserServiceClient) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request payload",
		},
		{
			name: "User already exists — gRPC AlreadyExists returns 409",
			inputBody: `{
				"username": "user-register-6",
				"password": "pass-register-6",
				"email": "user-register-6@test.com"
			}`,
			setupMock: func(mc *Mock_UserServiceClient) {
				mc.On("Register", mock.Anything, mock.Anything).
					Return(nil, status.Error(codes.AlreadyExists, "User already exists"))
			},
			expectedStatus: http.StatusConflict,
			expectedBody:   "User already exists",
		},
		{
			name: "Internal gRPC error — returns 500",
			inputBody: `{
				"username": "user-register-7",
				"password": "pass-register-7",
				"email": "user-register-7@test.com"
			}`,
			setupMock: func(mc *Mock_UserServiceClient) {
				mc.On("Register", mock.Anything, mock.Anything).
					Return(nil, status.Error(codes.Internal, "Internal server error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := NewMock_UserServiceClient()
			tt.setupMock(mc)
			h := newTestHandler(mc)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/users/register", strings.NewReader(tt.inputBody))
			c.Request.Header.Set("Content-Type", "application/json")

			h.Register(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedBody)
			mc.AssertExpectations(t)
		})
	}
}

/* LOGIN */

func TestUserHandler_Login(t *testing.T) {
	tests := []struct {
		name           string
		inputBody      string
		setupMock      func(*Mock_UserServiceClient)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Login with username — bind identifier correctly to LoginRequest_Username",
			inputBody: `{
				"identifier": "user-login-1",
				"password": "pass-login-1"
			}`,
			setupMock: func(mc *Mock_UserServiceClient) {
				mc.On("Login", mock.Anything,
					mock.MatchedBy(func(req *userv1.LoginRequest) bool {
						_, isUsername := req.Identifier.(*userv1.LoginRequest_Username)
						return isUsername &&
							req.GetUsername() == "user-login-1" &&
							req.GetPassword() == "pass-login-1"
					}),
				).Return(&userv1.LoginResponse{
					AccessToken:  "access-token-login-1",
					RefreshToken: "refresh-token-login-1",
					User: &userv1.User{
						Id:       "uid-user-login-1",
						Username: "user-login-1",
						Email:    "user-login-1@test.com",
					},
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "uid-user-login-1",
		},
		{
			name: "Login with email — bind identifier correctly to LoginRequest_Email",
			inputBody: `{
				"identifier": "user-login-2@test.com",
				"password": "pass-login-2"
			}`,
			setupMock: func(mc *Mock_UserServiceClient) {
				mc.On("Login", mock.Anything,
					mock.MatchedBy(func(req *userv1.LoginRequest) bool {
						_, isEmail := req.Identifier.(*userv1.LoginRequest_Email)
						return isEmail &&
							req.GetEmail() == "user-login-2@test.com" &&
							req.GetPassword() == "pass-login-2"
					}),
				).Return(&userv1.LoginResponse{
					AccessToken:  "access-token-login-2",
					RefreshToken: "refresh-token-login-2",
					User: &userv1.User{
						Id:       "uid-user-login-2",
						Username: "user-login-2",
						Email:    "user-login-2@test.com",
					},
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "uid-user-login-2",
		},
		{
			name: "Missing identifier (required) — bind validation fails and returns 400",
			inputBody: `{
				"password": "pass-login-3"
			}`,
			setupMock:      func(mc *Mock_UserServiceClient) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request payload",
		},
		{
			name: "Missing password (required) — bind validation fails and returns 400",
			inputBody: `{
				"identifier": "user-login-4"
			}`,
			setupMock:      func(mc *Mock_UserServiceClient) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request payload",
		},
		{
			name:           "Invalid JSON — bind fails and returns 400",
			inputBody:      `{invalid`,
			setupMock:      func(mc *Mock_UserServiceClient) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request payload",
		},
		{
			name: "Incorrect password — gRPC Unauthenticated returns 401",
			inputBody: `{
				"identifier": "user-login-5",
				"password": "wrong-pass"
			}`,
			setupMock: func(mc *Mock_UserServiceClient) {
				mc.On("Login", mock.Anything, mock.Anything).
					Return(nil, status.Error(codes.Unauthenticated, "Invalid credentials"))
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Invalid credentials",
		},
		{
			name: "User not found — gRPC NotFound returns 404",
			inputBody: `{
				"identifier": "user-login-6",
				"password": "pass-login-6"
			}`,
			setupMock: func(mc *Mock_UserServiceClient) {
				mc.On("Login", mock.Anything, mock.Anything).
					Return(nil, status.Error(codes.NotFound, "User not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   "User not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := NewMock_UserServiceClient()
			tt.setupMock(mc)
			h := newTestHandler(mc)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/users/login", strings.NewReader(tt.inputBody))
			c.Request.Header.Set("Content-Type", "application/json")

			h.Login(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedBody)
			mc.AssertExpectations(t)
		})
	}
}

/* ===== GET PROFILE ===== */

func TestUserHandler_GetProfile(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		setupMock      func(*Mock_UserServiceClient)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:   "Thành công — X-User-ID bind đúng vào GetProfileRequest.Id",
			userID: "uid-user-getprofile-1",
			setupMock: func(mc *Mock_UserServiceClient) {
				mc.On("GetProfile", mock.Anything,
					mock.MatchedBy(func(req *userv1.GetProfileRequest) bool {
						return req.GetId() == "uid-user-getprofile-1"
					}),
				).Return(&userv1.UserProfileResponse{
					User: &userv1.User{
						Id:       "uid-user-getprofile-1",
						Username: "user-getprofile-1",
						Email:    "user-getprofile-1@test.com",
					},
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "uid-user-getprofile-1",
		},
		{
			name:   "Thành công — response trả đúng username và email",
			userID: "uid-user-getprofile-2",
			setupMock: func(mc *Mock_UserServiceClient) {
				mc.On("GetProfile", mock.Anything, mock.Anything).
					Return(&userv1.UserProfileResponse{
						User: &userv1.User{
							Id:          "uid-user-getprofile-2",
							Username:    "user-getprofile-2",
							Email:       "user-getprofile-2@test.com",
							PhoneNumber: "0985761232",
						},
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "user-getprofile-2@test.com",
		},
		{
			name:   "User not found — gRPC NotFound returns 404",
			userID: "uid-user-getprofile-3",
			setupMock: func(mc *Mock_UserServiceClient) {
				mc.On("GetProfile", mock.Anything, mock.Anything).
					Return(nil, status.Error(codes.NotFound, "User not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   "User not found",
		},
		{
			name:   "Internal gRPC error — returns 500",
			userID: "uid-user-getprofile-4",
			setupMock: func(mc *Mock_UserServiceClient) {
				mc.On("GetProfile", mock.Anything, mock.Anything).
					Return(nil, status.Error(codes.Internal, "Internal server error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := NewMock_UserServiceClient()
			tt.setupMock(mc)
			h := newTestHandler(mc)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/v1/users/profile", nil)
			c.Set("X-User-ID", tt.userID)

			h.GetProfile(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedBody)
			mc.AssertExpectations(t)
		})
	}
}

/* UPDATE PROFILE */

func TestUserHandler_UpdateProfile(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		inputBody      string
		setupMock      func(*Mock_UserServiceClient)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:   "Update full profile — body and X-User-ID bind correctly to UpdateProfileRequest",
			userID: "uid-user-updateprofile-1",
			inputBody: `{
				"username": "user-updateprofile-1",
				"email": "user-updateprofile-1@test.com",
				"phone_number": "0985761241"
			}`,
			setupMock: func(mc *Mock_UserServiceClient) {
				mc.On("UpdateProfile", mock.Anything,
					mock.MatchedBy(func(req *userv1.UpdateProfileRequest) bool {
						return req.GetId() == "uid-user-updateprofile-1" &&
							req.GetUsername() == "user-updateprofile-1" &&
							req.GetEmail() == "user-updateprofile-1@test.com" &&
							req.GetPhoneNumber() == "0985761241"
					}),
				).Return(&userv1.UserProfileResponse{
					User: &userv1.User{
						Id:          "uid-user-updateprofile-1",
						Username:    "user-updateprofile-1",
						Email:       "user-updateprofile-1@test.com",
						PhoneNumber: "0985761241",
					},
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "uid-user-updateprofile-1",
		},
		{
			name:   "Update partial profile — only change username",
			userID: "uid-user-updateprofile-2",
			inputBody: `{
				"username": "user-updateprofile-2"
			}`,
			setupMock: func(mc *Mock_UserServiceClient) {
				mc.On("UpdateProfile", mock.Anything,
					mock.MatchedBy(func(req *userv1.UpdateProfileRequest) bool {
						return req.GetId() == "uid-user-updateprofile-2" &&
							req.GetUsername() == "user-updateprofile-2"
					}),
				).Return(&userv1.UserProfileResponse{
					User: &userv1.User{
						Id:       "uid-user-updateprofile-2",
						Username: "user-updateprofile-2",
					},
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "uid-user-updateprofile-2",
		},
		{
			name:           "Invalid JSON — bind fails and returns 400",
			userID:         "uid-user-updateprofile-3",
			inputBody:      `{invalid`,
			setupMock:      func(mc *Mock_UserServiceClient) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request payload",
		},
		{
			name:   "User not found — gRPC NotFound returns 404",
			userID: "uid-user-updateprofile-4",
			inputBody: `{
				"username": "user-updateprofile-4"
			}`,
			setupMock: func(mc *Mock_UserServiceClient) {
				mc.On("UpdateProfile", mock.Anything, mock.Anything).
					Return(nil, status.Error(codes.NotFound, "User not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   "User not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := NewMock_UserServiceClient()
			tt.setupMock(mc)
			h := newTestHandler(mc)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPut, "/v1/users/profile", strings.NewReader(tt.inputBody))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Set("X-User-ID", tt.userID)

			h.UpdateProfile(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedBody)
			mc.AssertExpectations(t)
		})
	}
}

/* ===== UPDATE VIP ACCOUNT ===== */

func TestUserHandler_UpdateVIPAccount(t *testing.T) {
	tests := []struct {
		name           string
		uriID          string
		inputBody      string
		setupMock      func(*Mock_UserServiceClient)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:      "Cấp VIP — uri id và is_vip bind đúng vào UpdateVIPAccountRequest",
			uriID:     "uid-user-updatevip-1",
			inputBody: `{"is_vip": true}`,
			setupMock: func(mc *Mock_UserServiceClient) {
				mc.On("UpdateVIPAccount", mock.Anything,
					mock.MatchedBy(func(req *userv1.UpdateVIPAccountRequest) bool {
						return req.GetId() == "uid-user-updatevip-1" && req.GetIsVip() == true
					}),
				).Return(&userv1.UpdateVIPAccountResponse{CurrentVipStatus: true}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"current_vip_status":true`,
		},
		{
			name:      "Revoke VIP — is_vip=false bind correctly and response returns current_vip_status false",
			uriID:     "uid-user-updatevip-2",
			inputBody: `{"is_vip": false}`,
			setupMock: func(mc *Mock_UserServiceClient) {
				mc.On("UpdateVIPAccount", mock.Anything,
					mock.MatchedBy(func(req *userv1.UpdateVIPAccountRequest) bool {
						return req.GetId() == "uid-user-updatevip-2" && req.GetIsVip() == false
					}),
				).Return(&userv1.UpdateVIPAccountResponse{CurrentVipStatus: false}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"current_vip_status":false`,
		},
		{
			name:           "Missing URI param id (required) — ShouldBindUri fails and returns 400",
			uriID:          "",
			inputBody:      `{"is_vip": true}`,
			setupMock:      func(mc *Mock_UserServiceClient) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request parameters",
		},
		{
			name:      "User not found — gRPC NotFound returns 404",
			uriID:     "uid-user-updatevip-4",
			inputBody: `{"is_vip": true}`,
			setupMock: func(mc *Mock_UserServiceClient) {
				mc.On("UpdateVIPAccount", mock.Anything, mock.Anything).
					Return(nil, status.Error(codes.NotFound, "User not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   "User not found",
		},
		{
			name:      "Permission denied — gRPC PermissionDenied returns 403",
			uriID:     "uid-user-updatevip-5",
			inputBody: `{"is_vip": true}`,
			setupMock: func(mc *Mock_UserServiceClient) {
				mc.On("UpdateVIPAccount", mock.Anything, mock.Anything).
					Return(nil, status.Error(codes.PermissionDenied, "Permission denied"))
			},
			expectedStatus: http.StatusForbidden,
			expectedBody:   "Permission denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := NewMock_UserServiceClient()
			tt.setupMock(mc)
			h := newTestHandler(mc)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPatch, "/v1/users/"+tt.uriID+"/vip", strings.NewReader(tt.inputBody))
			c.Request.Header.Set("Content-Type", "application/json")
			if tt.uriID != "" {
				c.Params = gin.Params{{Key: "id", Value: tt.uriID}}
			}

			h.UpdateVIPAccount(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedBody)
			mc.AssertExpectations(t)
		})
	}
}

/* LIST USERS */

func TestUserHandler_ListUsers(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    string
		setupMock      func(*Mock_UserServiceClient)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:        "List users with pagination — query params bind correctly",
			queryParams: "?page=1&limit=10",
			setupMock: func(mc *Mock_UserServiceClient) {
				mc.On("ListUsers", mock.Anything,
					mock.MatchedBy(func(req *userv1.ListUsersRequest) bool {
						return req.GetPagination().GetPage() == 1 &&
							req.GetPagination().GetLimit() == 10
					}),
				).Return(&userv1.ListUsersResponse{
					Users: []*userv1.User{
						{Id: "uid-user-listusers-1", Username: "user-listusers-1", Email: "user-listusers-1@test.com"},
						{Id: "uid-user-listusers-2", Username: "user-listusers-2", Email: "user-listusers-2@test.com"},
					},
					TotalCount: 2,
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "uid-user-listusers-1",
		},
		{
			name:        "Response returns correct total_count and all users",
			queryParams: "?page=1&limit=5",
			setupMock: func(mc *Mock_UserServiceClient) {
				mc.On("ListUsers", mock.Anything, mock.Anything).
					Return(&userv1.ListUsersResponse{
						Users: []*userv1.User{
							{Id: "uid-user-listusers-3", Username: "user-listusers-3"},
						},
						TotalCount: 1,
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"total_count":1`,
		},
		{
			name:        "Empty result — returns []",
			queryParams: "?page=1&limit=10",
			setupMock: func(mc *Mock_UserServiceClient) {
				mc.On("ListUsers", mock.Anything, mock.Anything).
					Return(&userv1.ListUsersResponse{Users: nil, TotalCount: 0}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"total_count":0`,
		},
		{
			name:        "Internal gRPC error — returns 500",
			queryParams: "?page=1&limit=10",
			setupMock: func(mc *Mock_UserServiceClient) {
				mc.On("ListUsers", mock.Anything, mock.Anything).
					Return(nil, status.Error(codes.Internal, "Internal server error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := NewMock_UserServiceClient()
			tt.setupMock(mc)
			h := newTestHandler(mc)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/v1/users"+tt.queryParams, nil)

			h.ListUsers(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedBody)
			mc.AssertExpectations(t)
		})
	}
}

/* DELETE USERS */

func TestUserHandler_DeleteUsers(t *testing.T) {
	tests := []struct {
		name           string
		inputBody      string
		setupMock      func(*Mock_UserServiceClient)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:      "Delete a user — user_ids bind correctly to DeleteUsersRequest.Ids",
			inputBody: `{"user_ids": ["uid-user-deleteusers-1"]}`,
			setupMock: func(mc *Mock_UserServiceClient) {
				mc.On("DeleteUsers", mock.Anything,
					mock.MatchedBy(func(req *userv1.DeleteUsersRequest) bool {
						return len(req.GetIds()) == 1 && req.GetIds()[0] == "uid-user-deleteusers-1"
					}),
				).Return(nil)
			},
			expectedStatus: http.StatusNoContent,
			expectedBody:   "",
		},
		{
			name: "Delete multiple users — all IDs bind correctly to DeleteUsersRequest.Ids",
			inputBody: `{
				"user_ids": [
					"uid-user-deleteusers-2",
					"uid-user-deleteusers-3"
				]
			}`,
			setupMock: func(mc *Mock_UserServiceClient) {
				mc.On("DeleteUsers", mock.Anything,
					mock.MatchedBy(func(req *userv1.DeleteUsersRequest) bool {
						return len(req.GetIds()) == 2 &&
							req.GetIds()[0] == "uid-user-deleteusers-2" &&
							req.GetIds()[1] == "uid-user-deleteusers-3"
					}),
				).Return(nil)
			},
			expectedStatus: http.StatusNoContent,
			expectedBody:   "",
		},
		{
			name:           "Missing user_ids (required) — bind validation returns 400",
			inputBody:      `{}`,
			setupMock:      func(mc *Mock_UserServiceClient) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request payload",
		},
		{
			name:           "Invalid JSON — bind fails and returns 400",
			inputBody:      `{invalid`,
			setupMock:      func(mc *Mock_UserServiceClient) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request payload",
		},
		{
			name:      "User not found — gRPC NotFound returns 404",
			inputBody: `{"user_ids": ["uid-user-deleteusers-5"]}`,
			setupMock: func(mc *Mock_UserServiceClient) {
				mc.On("DeleteUsers", mock.Anything, mock.Anything).
					Return(status.Error(codes.NotFound, "User not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   "User not found",
		},
		{
			name:      "Permission denied — gRPC PermissionDenied returns 403",
			inputBody: `{"user_ids": ["uid-user-deleteusers-6"]}`,
			setupMock: func(mc *Mock_UserServiceClient) {
				mc.On("DeleteUsers", mock.Anything, mock.Anything).
					Return(status.Error(codes.PermissionDenied, "Permission denied"))
			},
			expectedStatus: http.StatusForbidden,
			expectedBody:   "Permission denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := NewMock_UserServiceClient()
			tt.setupMock(mc)
			h := newTestHandler(mc)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodDelete, "/v1/users", strings.NewReader(tt.inputBody))
			c.Request.Header.Set("Content-Type", "application/json")

			h.DeleteUsers(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}
			mc.AssertExpectations(t)
		})
	}
}
