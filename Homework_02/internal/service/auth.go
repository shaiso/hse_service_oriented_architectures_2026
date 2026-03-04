package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shaiso/marketplace/internal/model"
	"github.com/shaiso/marketplace/internal/repo"
	"golang.org/x/crypto/bcrypt"
)

const (
	ErrCodeTokenExpired        = "TOKEN_EXPIRED"
	ErrCodeTokenInvalid        = "TOKEN_INVALID"
	ErrCodeRefreshTokenInvalid = "REFRESH_TOKEN_INVALID"
	ErrCodeAccessDenied        = "ACCESS_DENIED"
	ErrCodeUserAlreadyExists   = "USER_ALREADY_EXISTS"
	ErrCodeInvalidCredentials  = "INVALID_CREDENTIALS"
)

type AuthService struct {
	userRepo        *repo.UserRepo
	jwtSecret       []byte
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

type Claims struct {
	UserID uuid.UUID      `json:"user_id"`
	Role   model.UserRole `json:"role"`
	jwt.RegisteredClaims
}

type AuthTokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type UserInfo struct {
	ID        uuid.UUID      `json:"id"`
	Username  string         `json:"username"`
	Role      model.UserRole `json:"role"`
	CreatedAt time.Time      `json:"created_at"`
}

func NewAuthService(userRepo *repo.UserRepo, jwtSecret string) *AuthService {
	return &AuthService{
		userRepo:        userRepo,
		jwtSecret:       []byte(jwtSecret),
		accessTokenTTL:  15 * time.Minute,
		refreshTokenTTL: 7 * 24 * time.Hour,
	}
}

func (s *AuthService) RegisterRaw(ctx context.Context, username, password, role string) (*UserInfo, error) {
	fields := make(map[string]string)
	if len(username) < 3 || len(username) > 100 {
		fields["username"] = "must be between 3 and 100 characters"
	}
	if len(password) < 6 || len(password) > 100 {
		fields["password"] = "must be between 6 and 100 characters"
	}
	r := model.UserRole(role)
	if r != model.RoleUser && r != model.RoleSeller && r != model.RoleAdmin {
		fields["role"] = "must be one of: USER, SELLER, ADMIN"
	}
	if len(fields) > 0 {
		return nil, &ValidationError{Fields: fields}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := model.User{
		Username:     username,
		PasswordHash: string(hash),
		Role:         r,
	}

	if err := s.userRepo.Create(ctx, &user); err != nil {
		if isDuplicateKey(err) {
			return nil, &BusinessError{Code: ErrCodeUserAlreadyExists, Message: "user already exists", Status: http.StatusConflict}
		}
		return nil, fmt.Errorf("create user: %w", err)
	}

	return &UserInfo{
		ID:        user.ID,
		Username:  user.Username,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
	}, nil
}

func (s *AuthService) LoginRaw(ctx context.Context, username, password string) (*AuthTokens, error) {
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isNoRows(err) {
			return nil, &BusinessError{Code: ErrCodeInvalidCredentials, Message: "invalid credentials", Status: http.StatusUnauthorized}
		}
		return nil, fmt.Errorf("get user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, &BusinessError{Code: ErrCodeInvalidCredentials, Message: "invalid credentials", Status: http.StatusUnauthorized}
	}

	return s.generateTokenPair(user.ID, user.Role)
}

func (s *AuthService) RefreshRaw(ctx context.Context, refreshToken string) (*AuthTokens, error) {
	claims, err := s.ParseToken(refreshToken)
	if err != nil {
		return nil, &BusinessError{Code: ErrCodeRefreshTokenInvalid, Message: "invalid refresh token", Status: http.StatusUnauthorized}
	}

	return s.generateTokenPair(claims.UserID, claims.Role)
}

func (s *AuthService) ParseToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

func (s *AuthService) generateTokenPair(userID uuid.UUID, role model.UserRole) (*AuthTokens, error) {
	accessToken, err := s.generateToken(userID, role, s.accessTokenTTL)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, err := s.generateToken(userID, role, s.refreshTokenTTL)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	return &AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *AuthService) generateToken(userID uuid.UUID, role model.UserRole, ttl time.Duration) (string, error) {
	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func isDuplicateKey(err error) bool {
	return err != nil && strings.Contains(err.Error(), "23505")
}
