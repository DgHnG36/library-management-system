package applications

import (
	"context"
	"time"

	"github.com/DgHnG36/lib-management-system/services/user-service/internal/models"
	"github.com/DgHnG36/lib-management-system/services/user-service/internal/repository"
	"github.com/DgHnG36/lib-management-system/services/user-service/pkg/logger"
	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type JWTClaims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	Email  string `json:"email"`
	jwt.StandardClaims
}

type UserService struct {
	userRepo     repository.UserRepository
	jwtSecret    []byte
	jwtAlgorithm string
	jwtExpMins   time.Duration
	logger       *logger.Logger
}

func NewUserService(
	userRepo repository.UserRepository,
	jwtSecret []byte,
	jwtAlgorithm string,
	jwtExpMins time.Duration,
	logger *logger.Logger,
) *UserService {
	return &UserService{
		userRepo:     userRepo,
		jwtSecret:    jwtSecret,
		jwtAlgorithm: jwtAlgorithm,
		jwtExpMins:   jwtExpMins,
		logger:       logger,
	}
}

func (s *UserService) Register(ctx context.Context, username, password, email, phoneNumber string) (*models.User, error) {
	s.logger.Info("Registering user", logger.Fields{
		"username": username,
		"email":    email,
	})

	existing, err := s.userRepo.FindByUsername(ctx, username)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check username: %v", err)
	}
	if existing != nil {
		return nil, status.Errorf(codes.AlreadyExists, "username already exists")
	}

	existing, err = s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check email: %v", err)
	}
	if existing != nil {
		return nil, status.Errorf(codes.AlreadyExists, "email already exists")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to hash password: %v", err)
	}

	newUser := &models.User{
		ID:          uuid.New().String(),
		Username:    username,
		Password:    string(hashedPassword),
		Email:       email,
		PhoneNumber: phoneNumber,
		Role:        models.RoleRegisteredUser,
		IsVip:       false,
		IsActive:    true,
		CreatedAt:   time.Now().UTC(),
		LastUpdated: time.Now().UTC(),
	}

	if err := s.userRepo.Create(ctx, newUser); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create new user: %v", err)
	}

	s.logger.Info("User registered", logger.Fields{
		"user_id":  newUser.ID,
		"username": newUser.Username,
	})
	return newUser, nil
}

func (s *UserService) Login(ctx context.Context, identifier, password string, byEmail bool) (*models.User, string, error) {
	s.logger.Info("User login attempt", logger.Fields{
		"identifier": identifier,
		"by_email":   byEmail,
	})

	var loginUser *models.User
	var err error
	if byEmail {
		loginUser, err = s.userRepo.FindByEmail(ctx, identifier)
	} else {
		loginUser, err = s.userRepo.FindByUsername(ctx, identifier)
	}

	if err != nil {
		return nil, "", status.Errorf(codes.Internal, "failed to find user: %v", err)
	}
	if loginUser == nil {
		return nil, "", status.Errorf(codes.NotFound, "invalid credentials")
	}
	if !loginUser.IsActive {
		return nil, "", status.Errorf(codes.PermissionDenied, "account is inactive")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(loginUser.Password), []byte(password)); err != nil {
		return nil, "", status.Errorf(codes.Unauthenticated, "invalid credentials")
	}

	token, err := s.generateToken(loginUser)
	if err != nil {
		return nil, "", status.Errorf(codes.Internal, "failed to generate token: %v", err)
	}

	s.logger.Info("User logged in", logger.Fields{
		"user_id": loginUser.ID,
	})
	return loginUser, token, nil
}

func (s *UserService) GetProfile(ctx context.Context, userID string) (*models.User, error) {
	s.logger.Info("Getting user profile", logger.Fields{
		"user_id": userID,
	})

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find user: %v", err)
	}
	if user == nil {
		return nil, status.Errorf(codes.NotFound, "user not found")
	}

	return user, nil
}

func (s *UserService) UpdateProfile(ctx context.Context, userID, username, email, phoneNumber string) (*models.User, error) {
	s.logger.Info("Updating user profile", logger.Fields{
		"user_id": userID,
	})

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find user: %v", err)
	}
	if user == nil {
		return nil, status.Errorf(codes.NotFound, "user not found")
	}

	if username != "" && username != user.Username {
		existing, err := s.userRepo.FindByUsername(ctx, username)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to check username: %v", err)
		}
		if existing != nil {
			return nil, status.Errorf(codes.AlreadyExists, "username '%s' already taken", username)
		}
		user.Username = username
	}

	if email != "" && email != user.Email {
		existing, err := s.userRepo.FindByEmail(ctx, email)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to check email: %v", err)
		}
		if existing != nil {
			return nil, status.Errorf(codes.AlreadyExists, "email '%s' already registered", email)
		}
		user.Email = email
	}

	if phoneNumber != "" {
		user.PhoneNumber = phoneNumber
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update user: %v", err)
	}

	return user, nil
}

func (s *UserService) UpdateVIPAccount(ctx context.Context, id string, isVip bool) (bool, error) {
	s.logger.Info("Updating VIP status", logger.Fields{"user_id": id, "is_vip": isVip})

	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		return false, status.Errorf(codes.Internal, "failed to find user: %v", err)
	}
	if user == nil {
		return false, status.Errorf(codes.NotFound, "user %s not found", id)
	}

	if err := s.userRepo.UpdateVIPStatus(ctx, id, isVip); err != nil {
		return false, status.Errorf(codes.Internal, "failed to update VIP status: %v", err)
	}

	return isVip, nil
}

func (s *UserService) ListUsers(ctx context.Context, page, limit int32, sortBy string, isDesc bool, role models.UserRole) ([]*models.User, int32, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}

	users, total, err := s.userRepo.List(ctx, page, limit, sortBy, isDesc, role)
	if err != nil {
		return nil, 0, status.Errorf(codes.Internal, "failed to list users: %v", err)
	}
	return users, total, nil
}

func (s *UserService) DeleteUsers(ctx context.Context, ids []string) error {
	s.logger.Info("Deleting users", logger.Fields{"ids": ids})

	if err := s.userRepo.Delete(ctx, ids); err != nil {
		return status.Errorf(codes.Internal, "failed to delete users: %v", err)
	}
	return nil
}

/* HELPER METHODS */
func (s *UserService) generateToken(user *models.User) (string, error) {
	expMins := int(s.jwtExpMins.Minutes())
	if expMins <= 0 {
		expMins = 60
	}

	claims := JWTClaims{
		UserID: user.ID,
		Role:   string(user.Role),
		Email:  user.Email,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Duration(expMins) * time.Minute).Unix(),
			IssuedAt:  time.Now().Unix(),
			Issuer:    "lib-management-system",
			Subject:   user.ID,
			Audience:  "gateway-service",
		},
	}

	var signingMethod jwt.SigningMethod
	switch s.jwtAlgorithm {
	case "HS512":
		signingMethod = jwt.SigningMethodHS512
	default:
		signingMethod = jwt.SigningMethodHS256
	}

	token := jwt.NewWithClaims(signingMethod, claims)
	return token.SignedString(s.jwtSecret)
}
