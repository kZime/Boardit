// internal/handler/auth.go

// api: register, login, refresh token

package handler

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"backend/internal/database"
	"backend/internal/model"
	"backend/internal/response"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// ------------------------------------------------------------
// register
// ------------------------------------------------------------

type registerRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{2,31}$`)

func Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Warn("register validation failed", "err", err)
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(req.Email)
	if !usernamePattern.MatchString(req.Username) {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "username must be 3-32 URL-safe characters")
		return
	}
	var existingCount int64
	if err := database.DB.Model(&model.User{}).
		Where("username = ? OR email = ?", req.Username, req.Email).
		Count(&existingCount).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL", "failed to validate account uniqueness")
		return
	}
	if existingCount > 0 {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "username or email already exists")
		return
	}

	// hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("hash password failed", "err", err)
		response.Error(c, http.StatusInternalServerError, "INTERNAL", "hash password error")
		return
	}

	// insert database
	user := model.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hash),
	}
	if err := database.DB.Create(&user).Error; err != nil {
		slog.Warn("register db create failed", "err", err)
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "username or email already exists")
		return
	}

	// return User shape (OpenAPI): id, username, email, created_at
	c.JSON(http.StatusCreated, gin.H{
		"id":         user.ID,
		"username":   user.Username,
		"email":      user.Email,
		"created_at": user.CreatedAt.Format(time.RFC3339),
	})
}

// ------------------------------------------------------------
// login
// ------------------------------------------------------------

var (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 7 * 24 * time.Hour
)

const (
	accessTokenType  = "access"
	refreshTokenType = "refresh"
)

func newTokenID() (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}

type signedAuthToken struct {
	Value     string
	ID        string
	ExpiresAt time.Time
}

func signAuthToken(userID uint, tokenType string, ttl time.Duration) (signedAuthToken, error) {
	tokenID, err := newTokenID()
	if err != nil {
		return signedAuthToken{}, err
	}
	now := time.Now().UTC()
	expiresAt := now.Add(ttl)
	claims := jwt.MapClaims{
		"sub": userID,
		"typ": tokenType,
		"jti": tokenID,
		"iat": now.Unix(),
		"exp": expiresAt.Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	value, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		return signedAuthToken{}, err
	}
	return signedAuthToken{Value: value, ID: tokenID, ExpiresAt: expiresAt}, nil
}

type loginRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	req.Email = strings.TrimSpace(req.Email)

	// 1. find user
	var user model.User
	if err := database.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid credentials")
		return
	}
	// 2. compare password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid credentials")
		return
	}

	// 3. sign token

	accessToken, err := signAuthToken(user.ID, accessTokenType, accessTokenTTL)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL", "sign access token error")
		return
	}

	refreshToken, err := signAuthToken(user.ID, refreshTokenType, refreshTokenTTL)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL", "sign refresh token error")
		return
	}
	if err := database.DB.Create(&model.RefreshSession{
		UserID: user.ID, TokenID: refreshToken.ID, ExpiresAt: refreshToken.ExpiresAt,
		CreatedAt: time.Now().UTC(),
	}).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL", "create refresh session error")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken.Value,
		"refresh_token": refreshToken.Value,
		"expires_in":    int(accessTokenTTL.Seconds()),
	})
}

// ------------------------------------------------------------
// refresh token
// ------------------------------------------------------------

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func parseRefreshToken(raw string) (uint, string, bool) {
	token, err := jwt.Parse(raw, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(os.Getenv("JWT_SECRET")), nil
	})
	if err != nil || !token.Valid {
		return 0, "", false
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, "", false
	}
	if tokenType, ok := claims["typ"].(string); !ok || tokenType != refreshTokenType {
		return 0, "", false
	}
	sub, subOK := claims["sub"].(float64)
	tokenID, tokenIDOK := claims["jti"].(string)
	if !subOK || !tokenIDOK || tokenID == "" {
		return 0, "", false
	}
	return uint(sub), tokenID, true
}

var errInvalidRefreshSession = errors.New("invalid refresh session")

func Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	userID, tokenID, ok := parseRefreshToken(req.RefreshToken)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid refresh token")
		return
	}

	// 2. sign new token
	newAccessToken, err := signAuthToken(userID, accessTokenType, accessTokenTTL)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL", "sign access token error")
		return
	}

	newRefreshToken, err := signAuthToken(userID, refreshTokenType, refreshTokenTTL)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL", "sign refresh token error")
		return
	}
	now := time.Now().UTC()
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.RefreshSession{}).
			Where("user_id = ? AND token_id = ? AND revoked_at IS NULL AND expires_at > ?", userID, tokenID, now).
			Updates(map[string]any{"revoked_at": now, "replaced_by_token_id": newRefreshToken.ID})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errInvalidRefreshSession
		}
		return tx.Create(&model.RefreshSession{
			UserID: userID, TokenID: newRefreshToken.ID, ExpiresAt: newRefreshToken.ExpiresAt,
			CreatedAt: now,
		}).Error
	})
	if errors.Is(err, errInvalidRefreshSession) {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid refresh token")
		return
	}
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL", "rotate refresh token error")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  newAccessToken.Value,
		"refresh_token": newRefreshToken.Value,
		"expires_in":    int(accessTokenTTL.Seconds()),
	})
}

func Logout(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	userID, tokenID, ok := parseRefreshToken(req.RefreshToken)
	if ok {
		now := time.Now().UTC()
		database.DB.Model(&model.RefreshSession{}).
			Where("user_id = ? AND token_id = ? AND revoked_at IS NULL", userID, tokenID).
			Update("revoked_at", now)
	}
	c.Status(http.StatusNoContent)
}
