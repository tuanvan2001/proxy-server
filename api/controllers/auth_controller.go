package controllers

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/tuantech/proxy-server/api/config"
	"github.com/tuantech/proxy-server/api/database"
	"github.com/tuantech/proxy-server/api/models"
)

// LoginRequest đại diện cho yêu cầu đăng nhập
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse đại diện cho phản hồi đăng nhập thành công
type LoginResponse struct {
	Token   string `json:"token"`
	Expires string `json:"expires"`
}

// Login xử lý yêu cầu đăng nhập và tạo JWT token
func Login(c *gin.Context) {
	var loginReq LoginRequest

	// Parse request body
	if err := c.ShouldBindJSON(&loginReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Kết nối đến cơ sở dữ liệu
	db, err := database.InitDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection error"})
		return
	}
	defer db.Close()

	// Lấy thông tin người dùng từ cơ sở dữ liệu
	user, err := models.GetUserByUsername(db, loginReq.Username)
	if err != nil {
		if err == sql.ErrNoRows || err.Error() == "user not found" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		}
		return
	}

	// Kiểm tra mật khẩu
	hashedPassword := models.HashPassword(loginReq.Password)
	if hashedPassword != user.Password {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	// Kiểm tra xem người dùng có trong danh sách AdminUsers không
	isAdmin := false
	for _, adminUser := range config.AdminUsers {
		if user.Username == adminUser {
			isAdmin = true
			break
		}
	}

	if !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "User not authorized to access this API"})
		return
	}

	// Tạo JWT token
	expiration := time.Now().Add(time.Hour * time.Duration(config.JWTExpiration))
	claims := jwt.MapClaims{
		"username": user.Username,
		"exp":      expiration.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(config.JWTSecret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Trả về token
	c.JSON(http.StatusOK, LoginResponse{
		Token:   tokenString,
		Expires: expiration.Format(time.RFC3339),
	})
}