package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/DgHnG36/lib-management-system/services/user-service/internal/applications"
	"github.com/DgHnG36/lib-management-system/services/user-service/internal/handlers"
	"github.com/DgHnG36/lib-management-system/services/user-service/pkg/interceptor"
	userv1 "github.com/DgHnG36/lib-management-system/shared/go/v1/user"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

/* HANDLER TEST HELPERS */

func newHandlerSetup() (*handlers.UserHandler, *MockUserRepository) {
	repo := NewMockUserRepository()
	svc := applications.NewUserService(repo, []byte("test-secret"), "HS256", 60*time.Minute, testLog)
	h := handlers.NewUserHandler(svc, testLog)
	return h, repo
}

// ctxWithRole injects X-User-Role into a background context using the typed key.
func ctxWithRole(role string) context.Context {
	return context.WithValue(context.Background(), interceptor.ContextKeyUserRole, role)
}

/* TestUserHandler_Register */
/* Verifies: required-field validation (username/password/email), correct Status/Message/UserId
   in response, duplicate username detection. */

func TestUserHandler_Register(t *testing.T) {
	tests := []struct {
		name        string
		req         *userv1.RegisterRequest
		preRegister *userv1.RegisterRequest // register this first to trigger duplicate
		wantErrCode codes.Code
		check       func(t *testing.T, r *userv1.RegisterResponse)
	}{
		{
			name: "valid request — Status 201, Message 'Registered successfully', UserId non-empty",
			req:  &userv1.RegisterRequest{Username: "alice", Password: "pass", Email: "alice@x.com", PhoneNumber: "+1"},
			check: func(t *testing.T, r *userv1.RegisterResponse) {
				assert.Equal(t, int32(201), r.GetStatus())
				assert.Equal(t, "Registered successfully", r.GetMessage())
				assert.NotEmpty(t, r.GetUserId())
			},
		},
		{
			name:        "missing username — InvalidArgument",
			req:         &userv1.RegisterRequest{Username: "", Password: "pass", Email: "a@x.com"},
			wantErrCode: codes.InvalidArgument,
		},
		{
			name:        "missing password — InvalidArgument",
			req:         &userv1.RegisterRequest{Username: "alice", Password: "", Email: "a@x.com"},
			wantErrCode: codes.InvalidArgument,
		},
		{
			name:        "missing email — InvalidArgument",
			req:         &userv1.RegisterRequest{Username: "alice", Password: "pass", Email: ""},
			wantErrCode: codes.InvalidArgument,
		},
		{
			name:        "duplicate username — AlreadyExists",
			req:         &userv1.RegisterRequest{Username: "alice", Password: "pass2", Email: "dup@x.com"},
			preRegister: &userv1.RegisterRequest{Username: "alice", Password: "pass", Email: "alice@x.com"},
			wantErrCode: codes.AlreadyExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newHandlerSetup()
			ctx := context.Background()
			if tt.preRegister != nil {
				_, err := h.Register(ctx, tt.preRegister)
				assert.NoError(t, err)
			}
			resp, err := h.Register(ctx, tt.req)
			if tt.wantErrCode != codes.OK {
				assert.Nil(t, resp)
				st, _ := status.FromError(err)
				assert.Equal(t, tt.wantErrCode, st.Code())
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, resp)
			if tt.check != nil {
				tt.check(t, resp)
			}
		})
	}
}

/* TestUserHandler_Login */
/* Verifies: oneof identifier (username vs email) is routed correctly, Status/Message/
   tokens/User fields in response, required-field validation, wrong credential errors. */

func TestUserHandler_Login(t *testing.T) {
	tests := []struct {
		name        string
		req         *userv1.LoginRequest
		wantErrCode codes.Code
		check       func(t *testing.T, r *userv1.LoginResponse)
	}{
		{
			name: "login by username — Status 200, 'Login successfully', tokens non-empty, user.Username matches",
			req: &userv1.LoginRequest{
				Identifier: &userv1.LoginRequest_Username{Username: "alice"},
				Password:   "secret",
			},
			check: func(t *testing.T, r *userv1.LoginResponse) {
				assert.Equal(t, int32(200), r.GetStatus())
				assert.Equal(t, "Login successfully", r.GetMessage())
				assert.NotEmpty(t, r.GetAccessToken())
				assert.NotEmpty(t, r.GetRefreshToken())
				assert.Equal(t, "alice", r.GetUser().GetUsername())
			},
		},
		{
			name: "login by email — user.Email matches identifier",
			req: &userv1.LoginRequest{
				Identifier: &userv1.LoginRequest_Email{Email: "alice@x.com"},
				Password:   "secret",
			},
			check: func(t *testing.T, r *userv1.LoginResponse) {
				assert.Equal(t, "alice@x.com", r.GetUser().GetEmail())
				assert.NotEmpty(t, r.GetAccessToken())
			},
		},
		{
			name: "missing password — InvalidArgument",
			req: &userv1.LoginRequest{
				Identifier: &userv1.LoginRequest_Username{Username: "alice"},
				Password:   "",
			},
			wantErrCode: codes.InvalidArgument,
		},
		{
			name:        "missing identifier (nil) — InvalidArgument",
			req:         &userv1.LoginRequest{Password: "secret", Identifier: nil},
			wantErrCode: codes.InvalidArgument,
		},
		{
			name: "wrong password — Unauthenticated",
			req: &userv1.LoginRequest{
				Identifier: &userv1.LoginRequest_Username{Username: "alice"},
				Password:   "wrongpass",
			},
			wantErrCode: codes.Unauthenticated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newHandlerSetup()
			ctx := context.Background()
			// Pre-register alice
			_, err := h.Register(ctx, &userv1.RegisterRequest{
				Username: "alice", Password: "secret", Email: "alice@x.com",
			})
			assert.NoError(t, err)

			resp, err := h.Login(ctx, tt.req)
			if tt.wantErrCode != codes.OK {
				assert.Nil(t, resp)
				st, _ := status.FromError(err)
				assert.Equal(t, tt.wantErrCode, st.Code())
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, resp)
			if tt.check != nil {
				tt.check(t, resp)
			}
		})
	}
}

/* TestUserHandler_GetProfile */
/* Verifies: user proto fields are correctly mapped from model, ID required, not-found. */

func TestUserHandler_GetProfile(t *testing.T) {
	tests := []struct {
		name        string
		useRegID    bool   // true: use the ID returned by pre-registration
		id          string // used when useRegID is false
		wantErrCode codes.Code
		check       func(t *testing.T, r *userv1.UserProfileResponse)
	}{
		{
			name:     "existing user — proto User fields correctly mapped",
			useRegID: true,
			check: func(t *testing.T, r *userv1.UserProfileResponse) {
				u := r.GetUser()
				assert.NotNil(t, u)
				assert.NotEmpty(t, u.GetId())
				assert.Equal(t, "alice", u.GetUsername())
				assert.Equal(t, "alice@x.com", u.GetEmail())
				assert.Equal(t, "+1", u.GetPhoneNumber())
				assert.Equal(t, userv1.UserRole_REGISTERED_USER, u.GetRole())
				assert.True(t, u.GetIsActive())
			},
		},
		{
			name:        "missing ID — InvalidArgument",
			id:          "",
			wantErrCode: codes.InvalidArgument,
		},
		{
			name:        "non-existing ID — NotFound",
			id:          "does-not-exist",
			wantErrCode: codes.NotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newHandlerSetup()
			ctx := context.Background()
			reqID := tt.id
			if tt.useRegID {
				regResp, err := h.Register(ctx, &userv1.RegisterRequest{
					Username: "alice", Password: "pass", Email: "alice@x.com", PhoneNumber: "+1",
				})
				assert.NoError(t, err)
				reqID = regResp.GetUserId()
			}

			resp, err := h.GetProfile(ctx, &userv1.GetProfileRequest{Id: reqID})
			if tt.wantErrCode != codes.OK {
				assert.Nil(t, resp)
				st, _ := status.FromError(err)
				assert.Equal(t, tt.wantErrCode, st.Code())
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, resp)
			if tt.check != nil {
				tt.check(t, resp)
			}
		})
	}
}

/* TestUserHandler_UpdateProfile */
/* Verifies: updated fields reflected in response, unchanged fields preserved, ID required. */

func TestUserHandler_UpdateProfile(t *testing.T) {
	tests := []struct {
		name        string
		update      *userv1.UpdateProfileRequest // Id will be filled in at runtime (useRegID=true)
		useRegID    bool
		wantErrCode codes.Code
		check       func(t *testing.T, r *userv1.UserProfileResponse)
	}{
		{
			name:     "update phone — new phone reflected in response, username preserved",
			useRegID: true,
			update:   &userv1.UpdateProfileRequest{PhoneNumber: "+9999"},
			check: func(t *testing.T, r *userv1.UserProfileResponse) {
				assert.Equal(t, "+9999", r.GetUser().GetPhoneNumber())
				assert.Equal(t, "alice", r.GetUser().GetUsername(), "username should be unchanged")
			},
		},
		{
			name:     "update username — new username reflected in response",
			useRegID: true,
			update:   &userv1.UpdateProfileRequest{Username: "new_alice"},
			check: func(t *testing.T, r *userv1.UserProfileResponse) {
				assert.Equal(t, "new_alice", r.GetUser().GetUsername())
			},
		},
		{
			name:        "missing ID — InvalidArgument",
			update:      &userv1.UpdateProfileRequest{Id: "", Username: "x"},
			wantErrCode: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newHandlerSetup()
			ctx := context.Background()
			req := tt.update
			if tt.useRegID {
				regResp, err := h.Register(ctx, &userv1.RegisterRequest{
					Username: "alice", Password: "pass", Email: "alice@x.com", PhoneNumber: "+1",
				})
				assert.NoError(t, err)
				req = &userv1.UpdateProfileRequest{
					Id:          regResp.GetUserId(),
					Username:    tt.update.GetUsername(),
					Email:       tt.update.GetEmail(),
					PhoneNumber: tt.update.GetPhoneNumber(),
				}
			}

			resp, err := h.UpdateProfile(ctx, req)
			if tt.wantErrCode != codes.OK {
				assert.Nil(t, resp)
				st, _ := status.FromError(err)
				assert.Equal(t, tt.wantErrCode, st.Code())
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, resp)
			if tt.check != nil {
				tt.check(t, resp)
			}
		})
	}
}

/* TestUserHandler_UpdateVIPAccount */
/* Verifies: admin role required (from "X-User-Role" context), VIP status reflected in
   response, non-admin denied, ID required. */

func TestUserHandler_UpdateVIPAccount(t *testing.T) {
	tests := []struct {
		name        string
		ctx         context.Context
		isVip       bool
		useRegID    bool
		fixedID     string // used when useRegID is false
		wantErrCode codes.Code
		check       func(t *testing.T, r *userv1.UpdateVIPAccountResponse)
	}{
		{
			name:     "admin sets VIP true — Status 200, CurrentVipStatus true",
			ctx:      ctxWithRole("ADMIN"),
			isVip:    true,
			useRegID: true,
			check: func(t *testing.T, r *userv1.UpdateVIPAccountResponse) {
				assert.Equal(t, int32(200), r.GetStatus())
				assert.Equal(t, "VIP status updated successfully", r.GetMessage())
				assert.True(t, r.GetCurrentVipStatus())
			},
		},
		{
			name:     "admin sets VIP false — CurrentVipStatus false",
			ctx:      ctxWithRole("ADMIN"),
			isVip:    false,
			useRegID: true,
			check: func(t *testing.T, r *userv1.UpdateVIPAccountResponse) {
				assert.False(t, r.GetCurrentVipStatus())
			},
		},
		{
			name:        "manager caller — PermissionDenied",
			ctx:         ctxWithRole("MANAGER"),
			isVip:       true,
			fixedID:     "some-id",
			wantErrCode: codes.PermissionDenied,
		},
		{
			name:        "admin, missing ID — InvalidArgument",
			ctx:         ctxWithRole("ADMIN"),
			isVip:       true,
			fixedID:     "",
			wantErrCode: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newHandlerSetup()
			userID := tt.fixedID
			if tt.useRegID {
				regResp, err := h.Register(context.Background(), &userv1.RegisterRequest{
					Username: "alice", Password: "pass", Email: "alice@x.com",
				})
				assert.NoError(t, err)
				userID = regResp.GetUserId()
			}

			resp, err := h.UpdateVIPAccount(tt.ctx, &userv1.UpdateVIPAccountRequest{
				Id:    userID,
				IsVip: tt.isVip,
			})
			if tt.wantErrCode != codes.OK {
				assert.Nil(t, resp)
				st, _ := status.FromError(err)
				assert.Equal(t, tt.wantErrCode, st.Code())
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, resp)
			if tt.check != nil {
				tt.check(t, resp)
			}
		})
	}
}

/* TestUserHandler_ListUsers */
/* Verifies: admin/manager role required (from "X-User-Role" context), TotalCount matches
   number of registered users, non-admin denied, empty repo returns empty list. */

func TestUserHandler_ListUsers(t *testing.T) {
	tests := []struct {
		name        string
		ctx         context.Context
		preRegister int // number of users to pre-register
		wantErrCode codes.Code
		check       func(t *testing.T, r *userv1.ListUsersResponse)
	}{
		{
			name:        "admin caller, 3 users — TotalCount 3 and users slice non-empty",
			ctx:         ctxWithRole("ADMIN"),
			preRegister: 3,
			check: func(t *testing.T, r *userv1.ListUsersResponse) {
				assert.Equal(t, int32(3), r.GetTotalCount())
				assert.Len(t, r.GetUsers(), 3)
			},
		},
		{
			name:        "manager caller — returns users",
			ctx:         ctxWithRole("MANAGER"),
			preRegister: 1,
			check: func(t *testing.T, r *userv1.ListUsersResponse) {
				assert.Equal(t, int32(1), r.GetTotalCount())
			},
		},
		{
			name:        "registered user caller — PermissionDenied",
			ctx:         ctxWithRole("REGISTERED_USER"),
			wantErrCode: codes.PermissionDenied,
		},
		{
			name:        "admin caller, empty repo — empty users slice and TotalCount 0",
			ctx:         ctxWithRole("ADMIN"),
			preRegister: 0,
			check: func(t *testing.T, r *userv1.ListUsersResponse) {
				assert.Empty(t, r.GetUsers())
				assert.Equal(t, int32(0), r.GetTotalCount())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newHandlerSetup()
			for i := 0; i < tt.preRegister; i++ {
				_, err := h.Register(context.Background(), &userv1.RegisterRequest{
					Username: fmt.Sprintf("user%d", i),
					Password: "pass",
					Email:    fmt.Sprintf("user%d@x.com", i),
				})
				assert.NoError(t, err)
			}

			resp, err := h.ListUsers(tt.ctx, &userv1.ListUsersRequest{})
			if tt.wantErrCode != codes.OK {
				assert.Nil(t, resp)
				st, _ := status.FromError(err)
				assert.Equal(t, tt.wantErrCode, st.Code())
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, resp)
			if tt.check != nil {
				tt.check(t, resp)
			}
		})
	}
}

/* TestUserHandler_DeleteUsers */
/* Verifies: admin role required (from "X-User-Role" context), Status/Message in response,
   user removed from repo, non-admin denied, empty IDs rejected. */

func TestUserHandler_DeleteUsers(t *testing.T) {
	tests := []struct {
		name        string
		ctx         context.Context
		useRegID    bool
		fixedIDs    []string
		wantErrCode codes.Code
		check       func(t *testing.T, repo *MockUserRepository, userID string)
	}{
		{
			name:     "admin caller — Status 200, user removed from repo",
			ctx:      ctxWithRole("ADMIN"),
			useRegID: true,
			check: func(t *testing.T, repo *MockUserRepository, userID string) {
				_, exists := repo.users[userID]
				assert.False(t, exists, "user should be removed from repo after deletion")
			},
		},
		{
			name:        "manager caller — PermissionDenied",
			ctx:         ctxWithRole("MANAGER"),
			fixedIDs:    []string{"some-id"},
			wantErrCode: codes.PermissionDenied,
		},
		{
			name:        "admin caller, empty IDs — InvalidArgument",
			ctx:         ctxWithRole("ADMIN"),
			fixedIDs:    []string{},
			wantErrCode: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, repo := newHandlerSetup()
			var userID string
			ids := tt.fixedIDs
			if tt.useRegID {
				regResp, err := h.Register(context.Background(), &userv1.RegisterRequest{
					Username: "alice", Password: "pass", Email: "alice@x.com",
				})
				assert.NoError(t, err)
				userID = regResp.GetUserId()
				ids = []string{userID}
			}

			resp, err := h.DeleteUsers(tt.ctx, &userv1.DeleteUsersRequest{Ids: ids})
			if tt.wantErrCode != codes.OK {
				assert.Nil(t, resp)
				st, _ := status.FromError(err)
				assert.Equal(t, tt.wantErrCode, st.Code())
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, resp)
			assert.Equal(t, int32(200), resp.GetStatus())
			assert.Equal(t, "Users deleted successfully", resp.GetMessage())
			if tt.check != nil {
				tt.check(t, repo, userID)
			}
		})
	}
}

/* TestUserHandler_RefreshToken */
/* Verifies: refresh token required, valid refresh token produces new token pair,
   "X-User-ID" context value used for token lookup. */

func TestUserHandler_RefreshToken(t *testing.T) {
	tests := []struct {
		name         string
		setupLogin   bool   // true: register+login to obtain a real refresh token
		refreshToken string // used when setupLogin is false
		wantErrCode  codes.Code
		check        func(t *testing.T, r *userv1.LoginResponse)
	}{
		{
			name:       "valid refresh token — Status 200, new access and refresh tokens returned",
			setupLogin: true,
			check: func(t *testing.T, r *userv1.LoginResponse) {
				assert.Equal(t, int32(200), r.GetStatus())
				assert.NotEmpty(t, r.GetAccessToken())
				assert.NotEmpty(t, r.GetRefreshToken())
			},
		},
		{
			name:         "missing refresh token — InvalidArgument (checked before context access)",
			refreshToken: "",
			wantErrCode:  codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newHandlerSetup()
			ctx := context.Background()

			var refreshToken, userID string
			if tt.setupLogin {
				// Register + Login to obtain a real refresh token stored in the mock.
				regResp, err := h.Register(ctx, &userv1.RegisterRequest{
					Username: "alice", Password: "pass", Email: "alice@x.com",
				})
				assert.NoError(t, err)
				userID = regResp.GetUserId()

				loginResp, err := h.Login(ctx, &userv1.LoginRequest{
					Identifier: &userv1.LoginRequest_Username{Username: "alice"},
					Password:   "pass",
				})
				assert.NoError(t, err)
				refreshToken = loginResp.GetRefreshToken()
			} else {
				refreshToken = tt.refreshToken
			}

			reqCtx := context.WithValue(ctx, interceptor.ContextKeyUserID, userID)
			resp, err := h.RefreshToken(reqCtx, &userv1.RefreshTokenRequest{
				RefreshToken: refreshToken,
			})
			if tt.wantErrCode != codes.OK {
				assert.Nil(t, resp)
				st, _ := status.FromError(err)
				assert.Equal(t, tt.wantErrCode, st.Code())
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, resp)
			if tt.check != nil {
				tt.check(t, resp)
			}
		})
	}
}
