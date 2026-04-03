package main

import (
	"context"
	"os"
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

// ─── Mock Repository ────────────────────────────────────────────────────────

// MockUserRepository implements repository.UserRepository for unit tests.
// Shared across user_service_test.go and user_handler_test.go (same package).
type MockUserRepository struct {
	users  map[string]*models.User
	tokens map[string]*models.UserToken
}

func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		users:  make(map[string]*models.User),
		tokens: make(map[string]*models.UserToken),
	}
}

func (m *MockUserRepository) Create(_ context.Context, user *models.User) error {
	m.users[user.ID] = user
	return nil
}

func (m *MockUserRepository) FindByID(_ context.Context, id string) (*models.User, error) {
	return m.users[id], nil
}

func (m *MockUserRepository) FindByUsername(_ context.Context, username string) (*models.User, error) {
	for _, u := range m.users {
		if u.Username == username {
			return u, nil
		}
	}
	return nil, nil
}

func (m *MockUserRepository) FindByEmail(_ context.Context, email string) (*models.User, error) {
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, nil
}

func (m *MockUserRepository) Update(_ context.Context, user *models.User) error {
	m.users[user.ID] = user
	return nil
}

func (m *MockUserRepository) UpdateVIPStatus(_ context.Context, id string, isVip bool) error {
	if u, ok := m.users[id]; ok {
		u.IsVip = isVip
	}
	return nil
}

// List returns all users when role is empty or GUEST (proto zero-value = "no filter"),
// otherwise filters by the specified role.
func (m *MockUserRepository) List(_ context.Context, _, _ int32, _ string, _ bool, role models.UserRole) ([]*models.User, int32, error) {
	result := make([]*models.User, 0)
	for _, u := range m.users {
		if role == "" || role == models.RoleGuest {
			result = append(result, u)
		} else if u.Role == role {
			result = append(result, u)
		}
	}
	return result, int32(len(result)), nil
}

func (m *MockUserRepository) Delete(_ context.Context, ids []string) error {
	for _, id := range ids {
		delete(m.users, id)
	}
	return nil
}

func (m *MockUserRepository) StoreRefreshToken(_ context.Context, userID, hash string, expiresAt time.Time) error {
	m.tokens[hash] = &models.UserToken{
		UserID:           userID,
		RefreshTokenHash: hash,
		ExpiresAt:        expiresAt,
	}
	return nil
}

func (m *MockUserRepository) FindRefreshToken(_ context.Context, hash string) (*models.UserToken, error) {
	if tok, ok := m.tokens[hash]; ok {
		return tok, nil
	}
	return nil, nil
}

func (m *MockUserRepository) DeleteRefreshToken(_ context.Context, hash string) error {
	delete(m.tokens, hash)
	return nil
}

// ─── Shared test helpers ─────────────────────────────────────────────────────

var testLog = logger.DefaultNewLogger()

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func newTestSvc() (*applications.UserService, *MockUserRepository) {
	repo := NewMockUserRepository()
	svc := applications.NewUserService(repo, []byte("test-secret"), "HS256", 60*time.Minute, testLog)
	return svc, repo
}

// ─── TestUserService_Register ─────────────────────────────────────────────────
// Verifies: password hashing, default field values, duplicate detection, timestamps.

func TestUserService_Register(t *testing.T) {
	tests := []struct {
		name        string
		username    string
		password    string
		email       string
		phone       string
		setup       func(*MockUserRepository)
		wantErrCode codes.Code
		check       func(t *testing.T, u *models.User)
	}{
		{
			name:     "valid input — role REGISTERED_USER, isActive true, VIP false, password bcrypt-hashed",
			username: "alice", password: "secret123", email: "alice@x.com", phone: "+1",
			check: func(t *testing.T, u *models.User) {
				assert.Equal(t, "alice", u.Username)
				assert.Equal(t, "alice@x.com", u.Email)
				assert.Equal(t, "+1", u.PhoneNumber)
				assert.Equal(t, models.RoleRegisteredUser, u.Role)
				assert.True(t, u.IsActive)
				assert.False(t, u.IsVip)
				assert.NotEmpty(t, u.ID)
				assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(u.Password), []byte("secret123")),
					"stored password must be the bcrypt hash of the original plain-text password")
			},
		},
		{
			name:     "phone number optional — succeeds with empty phone",
			username: "bob", password: "pass", email: "bob@x.com", phone: "",
			check: func(t *testing.T, u *models.User) {
				assert.Equal(t, "", u.PhoneNumber)
				assert.NotEmpty(t, u.ID)
			},
		},
		{
			name:     "timestamps are set to current time on creation",
			username: "carol", password: "pass", email: "carol@x.com",
			check: func(t *testing.T, u *models.User) {
				assert.False(t, u.CreatedAt.IsZero(), "CreatedAt should be set")
				assert.False(t, u.LastUpdated.IsZero(), "LastUpdated should be set")
				assert.WithinDuration(t, time.Now().UTC(), u.CreatedAt, 5*time.Second)
			},
		},
		{
			name:     "duplicate username — AlreadyExists",
			username: "alice", password: "pass2", email: "alice2@x.com",
			setup: func(r *MockUserRepository) {
				r.users["u1"] = &models.User{ID: "u1", Username: "alice", Email: "other@x.com"}
			},
			wantErrCode: codes.AlreadyExists,
		},
		{
			name:     "duplicate email — AlreadyExists",
			username: "newuser", password: "pass", email: "dup@x.com",
			setup: func(r *MockUserRepository) {
				r.users["u2"] = &models.User{ID: "u2", Username: "other", Email: "dup@x.com"}
			},
			wantErrCode: codes.AlreadyExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo := newTestSvc()
			if tt.setup != nil {
				tt.setup(repo)
			}
			u, err := svc.Register(context.Background(), tt.username, tt.password, tt.email, tt.phone)
			if tt.wantErrCode != codes.OK {
				assert.Nil(t, u)
				st, _ := status.FromError(err)
				assert.Equal(t, tt.wantErrCode, st.Code())
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, u)
			if tt.check != nil {
				tt.check(t, u)
			}
		})
	}
}

// ─── TestUserService_Login ────────────────────────────────────────────────────
// Verifies: login by username, login by email, wrong password, user not found, inactive account.

func TestUserService_Login(t *testing.T) {
	const plainPwd = "p@ssw0rd"

	tests := []struct {
		name        string
		identifier  string
		password    string
		byEmail     bool
		inactive    bool
		wantErrCode codes.Code
		check       func(t *testing.T, u *models.User, tp *applications.TokenPair)
	}{
		{
			name:       "login by username — user and non-empty token pair returned",
			identifier: "alice", password: plainPwd, byEmail: false,
			check: func(t *testing.T, u *models.User, tp *applications.TokenPair) {
				assert.Equal(t, "alice", u.Username)
				assert.Equal(t, "alice@x.com", u.Email)
				assert.NotEmpty(t, tp.AccessToken)
				assert.NotEmpty(t, tp.RefreshToken)
			},
		},
		{
			name:       "login by email — user.Email matches identifier",
			identifier: "alice@x.com", password: plainPwd, byEmail: true,
			check: func(t *testing.T, u *models.User, tp *applications.TokenPair) {
				assert.Equal(t, "alice@x.com", u.Email)
				assert.NotEmpty(t, tp.AccessToken)
			},
		},
		{
			name:       "wrong password — Unauthenticated",
			identifier: "alice", password: "wrongpass", byEmail: false,
			wantErrCode: codes.Unauthenticated,
		},
		{
			name:       "user not found — NotFound",
			identifier: "no-such-user", password: plainPwd, byEmail: false,
			wantErrCode: codes.NotFound,
		},
		{
			name:       "inactive account — PermissionDenied",
			identifier: "alice", password: plainPwd, byEmail: false,
			inactive:    true,
			wantErrCode: codes.PermissionDenied,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo := newTestSvc()
			// Pre-register alice so there is always a valid user to test against.
			registered, err := svc.Register(context.Background(), "alice", plainPwd, "alice@x.com", "")
			assert.NoError(t, err)
			if tt.inactive {
				repo.users[registered.ID].IsActive = false
			}

			u, tp, err := svc.Login(context.Background(), tt.identifier, tt.password, tt.byEmail)
			if tt.wantErrCode != codes.OK {
				st, _ := status.FromError(err)
				assert.Equal(t, tt.wantErrCode, st.Code())
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, u)
			if tt.check != nil {
				tt.check(t, u, tp)
			}
		})
	}
}

// ─── TestUserService_GetProfile ───────────────────────────────────────────────
// Verifies: existing user returns correct fields, missing ID returns NotFound.

func TestUserService_GetProfile(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		setup       func(*MockUserRepository)
		wantErrCode codes.Code
		check       func(t *testing.T, u *models.User)
	}{
		{
			name: "existing user — all fields returned correctly",
			id:   "u1",
			setup: func(r *MockUserRepository) {
				r.users["u1"] = &models.User{
					ID: "u1", Username: "bob", Email: "bob@x.com",
					Role: models.RoleRegisteredUser, IsActive: true,
				}
			},
			check: func(t *testing.T, u *models.User) {
				assert.Equal(t, "u1", u.ID)
				assert.Equal(t, "bob", u.Username)
				assert.Equal(t, "bob@x.com", u.Email)
				assert.Equal(t, models.RoleRegisteredUser, u.Role)
				assert.True(t, u.IsActive)
			},
		},
		{
			name:        "non-existing ID — NotFound",
			id:          "does-not-exist",
			wantErrCode: codes.NotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo := newTestSvc()
			if tt.setup != nil {
				tt.setup(repo)
			}
			u, err := svc.GetProfile(context.Background(), tt.id)
			if tt.wantErrCode != codes.OK {
				assert.Nil(t, u)
				st, _ := status.FromError(err)
				assert.Equal(t, tt.wantErrCode, st.Code())
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, u)
			if tt.check != nil {
				tt.check(t, u)
			}
		})
	}
}

// ─── TestUserService_UpdateProfile ────────────────────────────────────────────
// Verifies: each field updated independently, unchanged fields preserved,
// duplicate username/email rejected, user not found.

func TestUserService_UpdateProfile(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		username    string
		email       string
		phone       string
		setup       func(*MockUserRepository)
		wantErrCode codes.Code
		check       func(t *testing.T, u *models.User)
	}{
		{
			name: "update phone — new phone applied, other fields preserved",
			id:   "u1", username: "", email: "", phone: "+9999",
			setup: func(r *MockUserRepository) {
				r.users["u1"] = &models.User{ID: "u1", Username: "bob", Email: "bob@x.com", PhoneNumber: "+1111"}
			},
			check: func(t *testing.T, u *models.User) {
				assert.Equal(t, "+9999", u.PhoneNumber)
				assert.Equal(t, "bob", u.Username, "username should be unchanged")
				assert.Equal(t, "bob@x.com", u.Email, "email should be unchanged")
			},
		},
		{
			name: "update username — new username applied",
			id:   "u1", username: "newbob", email: "", phone: "",
			setup: func(r *MockUserRepository) {
				r.users["u1"] = &models.User{ID: "u1", Username: "bob", Email: "bob@x.com"}
			},
			check: func(t *testing.T, u *models.User) {
				assert.Equal(t, "newbob", u.Username)
				assert.Equal(t, "bob@x.com", u.Email, "email should be unchanged")
			},
		},
		{
			name: "update email — new email applied",
			id:   "u1", username: "", email: "new@x.com", phone: "",
			setup: func(r *MockUserRepository) {
				r.users["u1"] = &models.User{ID: "u1", Username: "bob", Email: "bob@x.com"}
			},
			check: func(t *testing.T, u *models.User) {
				assert.Equal(t, "new@x.com", u.Email)
				assert.Equal(t, "bob", u.Username, "username should be unchanged")
			},
		},
		{
			name: "duplicate username already taken — AlreadyExists",
			id:   "u1", username: "taken", email: "", phone: "",
			setup: func(r *MockUserRepository) {
				r.users["u1"] = &models.User{ID: "u1", Username: "bob", Email: "bob@x.com"}
				r.users["u2"] = &models.User{ID: "u2", Username: "taken", Email: "taken@x.com"}
			},
			wantErrCode: codes.AlreadyExists,
		},
		{
			name: "user not found — NotFound",
			id:   "no-such-id", username: "new",
			wantErrCode: codes.NotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo := newTestSvc()
			if tt.setup != nil {
				tt.setup(repo)
			}
			u, err := svc.UpdateProfile(context.Background(), tt.id, tt.username, tt.email, tt.phone)
			if tt.wantErrCode != codes.OK {
				st, _ := status.FromError(err)
				assert.Equal(t, tt.wantErrCode, st.Code())
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, u)
			if tt.check != nil {
				tt.check(t, u)
			}
		})
	}
}

// ─── TestUserService_UpdateVIPAccount ─────────────────────────────────────────
// Verifies: VIP status toggle, returned value matches requested value, user not found.

func TestUserService_UpdateVIPAccount(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		isVip       bool
		setup       func(*MockUserRepository)
		wantErrCode codes.Code
		wantResult  bool
	}{
		{
			name: "set VIP true — returns true",
			id:   "u1", isVip: true,
			setup: func(r *MockUserRepository) {
				r.users["u1"] = &models.User{ID: "u1", Username: "bob", IsVip: false}
			},
			wantResult: true,
		},
		{
			name: "set VIP false — returns false",
			id:   "u1", isVip: false,
			setup: func(r *MockUserRepository) {
				r.users["u1"] = &models.User{ID: "u1", Username: "bob", IsVip: true}
			},
			wantResult: false,
		},
		{
			name: "user not found — NotFound",
			id:   "no-such-id", isVip: true,
			wantErrCode: codes.NotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo := newTestSvc()
			if tt.setup != nil {
				tt.setup(repo)
			}
			got, err := svc.UpdateVIPAccount(context.Background(), tt.id, tt.isVip)
			if tt.wantErrCode != codes.OK {
				st, _ := status.FromError(err)
				assert.Equal(t, tt.wantErrCode, st.Code())
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.wantResult, got)
		})
	}
}

// ─── TestUserService_ListUsers ────────────────────────────────────────────────
// Verifies: admin/manager can list users; registered user/guest is denied;
// total count reflects the number of users in the repo.

func TestUserService_ListUsers(t *testing.T) {
	tests := []struct {
		name        string
		callerRole  models.UserRole
		setup       func(*MockUserRepository)
		wantErrCode codes.Code
		wantTotal   int32
	}{
		{
			name:       "admin caller — returns all users and correct total count",
			callerRole: models.RoleAdmin,
			setup: func(r *MockUserRepository) {
				r.users["1"] = &models.User{ID: "1", Username: "u1", Role: models.RoleRegisteredUser}
				r.users["2"] = &models.User{ID: "2", Username: "u2", Role: models.RoleManager}
			},
			wantTotal: 2,
		},
		{
			name:       "manager caller — returns users",
			callerRole: models.RoleManager,
			setup: func(r *MockUserRepository) {
				r.users["1"] = &models.User{ID: "1", Username: "u1", Role: models.RoleRegisteredUser}
			},
			wantTotal: 1,
		},
		{
			name:        "registered user caller — PermissionDenied",
			callerRole:  models.RoleRegisteredUser,
			wantErrCode: codes.PermissionDenied,
		},
		{
			name:        "guest caller — PermissionDenied",
			callerRole:  models.RoleGuest,
			wantErrCode: codes.PermissionDenied,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo := newTestSvc()
			if tt.setup != nil {
				tt.setup(repo)
			}
			// models.RoleGuest as targetRole ≡ "no role filter" in mock
			users, total, err := svc.ListUsers(context.Background(), tt.callerRole, 1, 10, "", false, models.RoleGuest)
			if tt.wantErrCode != codes.OK {
				assert.Nil(t, users)
				st, _ := status.FromError(err)
				assert.Equal(t, tt.wantErrCode, st.Code())
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.wantTotal, total)
		})
	}
}

// ─── TestUserService_DeleteUsers ──────────────────────────────────────────────
// Verifies: admin deletes users (and they are removed from repo); non-admin is denied.

func TestUserService_DeleteUsers(t *testing.T) {
	tests := []struct {
		name        string
		callerRole  models.UserRole
		ids         []string
		setup       func(*MockUserRepository)
		wantErrCode codes.Code
		checkRepo   func(t *testing.T, r *MockUserRepository)
	}{
		{
			name:       "admin caller — users removed from repo",
			callerRole: models.RoleAdmin,
			ids:        []string{"u1"},
			setup: func(r *MockUserRepository) {
				r.users["u1"] = &models.User{ID: "u1", Username: "bob"}
			},
			checkRepo: func(t *testing.T, r *MockUserRepository) {
				_, ok := r.users["u1"]
				assert.False(t, ok, "user should be removed from repo after deletion")
			},
		},
		{
			name:        "manager caller — PermissionDenied",
			callerRole:  models.RoleManager,
			ids:         []string{"u1"},
			wantErrCode: codes.PermissionDenied,
		},
		{
			name:        "registered user caller — PermissionDenied",
			callerRole:  models.RoleRegisteredUser,
			ids:         []string{"u1"},
			wantErrCode: codes.PermissionDenied,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo := newTestSvc()
			if tt.setup != nil {
				tt.setup(repo)
			}
			err := svc.DeleteUsers(context.Background(), tt.callerRole, tt.ids)
			if tt.wantErrCode != codes.OK {
				st, _ := status.FromError(err)
				assert.Equal(t, tt.wantErrCode, st.Code())
				return
			}
			assert.NoError(t, err)
			if tt.checkRepo != nil {
				tt.checkRepo(t, repo)
			}
		})
	}
}
