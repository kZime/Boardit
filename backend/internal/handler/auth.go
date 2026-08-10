// internal/handler/auth.go

// api: register, login, refresh token

package handler

import (
	"crypto/rand"
	"encoding/hex"
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

func signAuthToken(userID uint, tokenType string, ttl time.Duration) (string, error) {
	tokenID, err := newTokenID()
	if err != nil {
		return "", err
	}
	now := time.Now()
	claims := jwt.MapClaims{
		"sub": userID,
		"typ": tokenType,
		"jti": tokenID,
		"iat": now.Unix(),
		"exp": now.Add(ttl).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(os.Getenv("JWT_SECRET")))
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

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_in":    int(accessTokenTTL.Seconds()),
	})
}

// ------------------------------------------------------------
// refresh token
// ------------------------------------------------------------

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	// 1. parse and verify refresh token
	var jwtKey = []byte(os.Getenv("JWT_SECRET"))

	token, err := jwt.Parse(req.RefreshToken, func(t *jwt.Token) (interface{}, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return jwtKey, nil
	})
	if err != nil || !token.Valid {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid refresh token")
		return
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid token claims")
		return
	}
	if tokenType, ok := claims["typ"].(string); !ok || tokenType != refreshTokenType {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid refresh token type")
		return
	}
	sub, ok := claims["sub"].(float64)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid sub claim")
		return
	}
	userID := uint(sub)
	var user model.User
	if err := database.DB.Select("id").First(&user, userID).Error; err != nil {
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

	c.JSON(http.StatusOK, gin.H{
		"access_token":  newAccessToken,
		"refresh_token": newRefreshToken,
		"expires_in":    int(accessTokenTTL.Seconds()),
	})
}
