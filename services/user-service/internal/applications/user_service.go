package applications

import (
	"context"
	"time"

	"github.com/DgHnG36/lib-management-system/services/user-service/internal/models"
	"github.com/DgHnG36/lib-management-system/services/user-service/internal/repository"
	"github.com/DgHnG36/lib-management-system/services/user-service/pkg/logger"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UserService struct {
	userRepo   repository.UserRepository
	jwtService *JWTService
	logger     *logger.Logger
}

func NewUserService(
	userRepo repository.UserRepository,
	jwtSecret []byte,
	jwtAlgorithm string,
	jwtExpMins time.Duration,
	logger *logger.Logger,
) *UserService {
	return &UserService{
		userRepo:   userRepo,
		jwtService: NewJWTService(jwtSecret, jwtAlgorithm, jwtExpMins),
		logger:     logger,
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
		return nil, status.Errorf(codes.AlreadyExists, "username already exists: %s", username)
	}

	existing, err = s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check email: %v", err)
	}
	if existing != nil {
		return nil, status.Errorf(codes.AlreadyExists, "email already exists: %s", email)
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

func (s *UserService) Login(ctx context.Context, identifier, password string, byEmail bool) (*models.User, *TokenPair, error) {
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
		return nil, &TokenPair{}, status.Errorf(codes.Internal, "failed to find user: %v", err)
	}
	if loginUser == nil {
		return nil, &TokenPair{}, status.Error(codes.NotFound, "invalid credentials")
	}
	if !loginUser.IsActive {
		return nil, &TokenPair{}, status.Error(codes.PermissionDenied, "account is inactive")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(loginUser.Password), []byte(password)); err != nil {
		return nil, &TokenPair{}, status.Error(codes.Unauthenticated, "invalid credentials")
	}

	tokenPair, tokenHashed, expiresAt, err := s.jwtService.GenerateTokenPair(loginUser)
	if err != nil {
		return nil, &TokenPair{}, status.Errorf(codes.Internal, "failed to generate token: %v", err)
	}

	err = s.userRepo.StoreRefreshToken(ctx, loginUser.ID, tokenHashed, expiresAt)
	if err != nil {
		return nil, &TokenPair{}, status.Errorf(codes.Internal, "failed to store refresh token: %v", err)
	}

	s.logger.Info("User logged in", logger.Fields{
		"user_id": loginUser.ID,
	})
	return loginUser, tokenPair, nil
}

func (s *UserService) GetProfile(ctx context.Context, userID string) (*models.User, error) {
	s.logger.Debug("Getting user profile", logger.Fields{
		"user_id": userID,
	})

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find user: %v", err)
	}
	if user == nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	s.logger.Debug("User profile retrieved", logger.Fields{
		"user_id": userID,
	})

	return user, nil
}

func (s *UserService) UpdateProfile(ctx context.Context, userID, username, email, phoneNumber string) (*models.User, error) {
	s.logger.Debug("Updating user profile", logger.Fields{
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

	s.logger.Debug("User profile updated", logger.Fields{
		"user_id": user.ID,
	})

	return user, nil
}

func (s *UserService) UpdateVIPAccount(ctx context.Context, id string, isVip bool) (bool, error) {
	s.logger.Info("Updating VIP status", logger.Fields{
		"user_id": id,
		"is_vip":  isVip,
	})

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

	s.logger.Info("VIP status updated", logger.Fields{
		"user_id": id,
		"is_vip":  isVip,
	})

	return isVip, nil
}

func (s *UserService) ListUsers(ctx context.Context, callerRole models.UserRole, page, limit int32, sortBy string, isDesc bool, targetRole models.UserRole) ([]*models.User, int32, error) {
	if callerRole == models.RoleGuest || callerRole == models.RoleRegisteredUser {
		return nil, 0, status.Error(codes.PermissionDenied, "insufficient permissions to list users")
	}

	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}

	s.logger.Debug("Listing users", logger.Fields{
		"page":  page,
		"limit": limit,
	})

	users, total, err := s.userRepo.List(ctx, page, limit, sortBy, isDesc, targetRole)
	if err != nil {
		return nil, 0, status.Errorf(codes.Internal, "failed to list users: %v", err)
	}

	s.logger.Debug("Users listed", logger.Fields{
		"page":  page,
		"limit": limit,
		"total": total,
	})

	return users, total, nil
}

func (s *UserService) DeleteUsers(ctx context.Context, callerRole models.UserRole, ids []string) error {
	if callerRole != models.RoleAdmin {
		return status.Error(codes.PermissionDenied, "insufficient permissions to delete users")
	}

	s.logger.Info("Deleting users", logger.Fields{
		"user_ids": ids,
	})

	if err := s.userRepo.Delete(ctx, ids); err != nil {
		return status.Errorf(codes.Internal, "failed to delete users: %v", err)
	}

	s.logger.Info("Users deleted", logger.Fields{
		"user_ids": ids,
	})

	return nil
}

func (s *UserService) RefreshToken(ctx context.Context, userID, refreshToken string) (*TokenPair, error) {
	s.logger.Debug("Refreshing token", logger.Fields{
		"user_id": userID,
	})

	tokenHashed := s.jwtService.HashRefreshToken(refreshToken)
	userToken, err := s.userRepo.FindRefreshToken(ctx, tokenHashed)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find user: %v", err)
	}
	if userToken == nil || userToken.UserID != userID {
		return nil, status.Errorf(codes.NotFound, "user %s not found", userID)
	}

	if time.Now().After(userToken.ExpiresAt) {
		if err := s.userRepo.DeleteRefreshToken(ctx, tokenHashed); err != nil {
			s.logger.Error("Failed to delete expired refresh token", err, logger.Fields{"user_id": userID})
		}
		s.logger.Debug("Refresh token expired", logger.Fields{
			"user_id": userID,
		})
		return nil, status.Errorf(codes.Unauthenticated, "refresh token expired")
	}

	user, err := s.userRepo.FindByID(ctx, userToken.UserID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find user: %v", err)
	}
	if user == nil {
		return nil, status.Errorf(codes.NotFound, "user %s not found", userToken.UserID)
	}

	tokenPair, tokenHashed, expiresAt, err := s.jwtService.GenerateTokenPair(user)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to refresh token: %v", err)
	}

	err = s.userRepo.StoreRefreshToken(ctx, userID, tokenHashed, expiresAt)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to store refresh token: %v", err)
	}

	s.logger.Debug("Token refreshed", logger.Fields{
		"user_id": userID,
	})

	return tokenPair, nil
}
