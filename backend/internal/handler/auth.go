// internal/handler/auth.go

// api: register, login, refresh token

package handler

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"backend/internal/database"
	"backend/internal/model"

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
	Password string `json:"password" binding:"required,min=6"`
}

func Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		fmt.Println("Register error:", err.Error())
		return
	}

	// hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "hash password error"})
		fmt.Println("Hash password error:", err.Error())
		return
	}

	// insert database
	user := model.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hash),
	}
	if err := database.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username or email already exists"})
		fmt.Println("Database error:", err.Error())
		return
	}

	// return basic info (without password)
	c.JSON(http.StatusCreated, gin.H{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
	})
}

// ------------------------------------------------------------
// login
// ------------------------------------------------------------

var (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 7 * 24 * time.Hour
)

type loginRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 1. find user
	var user model.User
	if err := database.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	// 2. compare password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// 3. sign token

	var jwtKey = []byte(os.Getenv("JWT_SECRET"))

	atClaims := jwt.MapClaims{"sub": user.ID, "exp": time.Now().Add(accessTokenTTL).Unix()}
	at := jwt.NewWithClaims(jwt.SigningMethodHS256, atClaims)
	accessToken, _ := at.SignedString(jwtKey)

	rtClaims := jwt.MapClaims{"sub": user.ID, "exp": time.Now().Add(refreshTokenTTL).Unix()}
	rt := jwt.NewWithClaims(jwt.SigningMethodHS256, rtClaims)
	refreshToken, _ := rt.SignedString(jwtKey)

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 1. parse and verify refresh token
	var jwtKey = []byte(os.Getenv("JWT_SECRET"))

	token, err := jwt.Parse(req.RefreshToken, func(t *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}
	claims := token.Claims.(jwt.MapClaims)
	userID := uint(claims["sub"].(float64))

	// 2. sign new token
	atClaims := jwt.MapClaims{"sub": userID, "exp": time.Now().Add(accessTokenTTL).Unix()}
	at := jwt.NewWithClaims(jwt.SigningMethodHS256, atClaims)
	newAccessToken, _ := at.SignedString(jwtKey)

	rtClaims := jwt.MapClaims{"sub": userID, "exp": time.Now().Add(refreshTokenTTL).Unix()}
	rt := jwt.NewWithClaims(jwt.SigningMethodHS256, rtClaims)
	newRefreshToken, _ := rt.SignedString(jwtKey)

	c.JSON(http.StatusOK, gin.H{
		"access_token":  newAccessToken,
		"refresh_token": newRefreshToken,
	})
}
