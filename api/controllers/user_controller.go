package controllers

import (
	"github.com/gin-gonic/gin"
	"github.com/tuantech/proxy-server/api/database"
	"github.com/tuantech/proxy-server/api/models"
	"log"
	"net/http"
)

// CreateUserRequest đại diện cho yêu cầu tạo người dùng mới
type CreateUserRequest struct {
	Username      string `json:"username" binding:"required"`
	Password      string `json:"password" binding:"required"`
	MaxConnection int    `json:"maxConnection" binding:"required"`
}

// UpdateUserRequest đại diện cho yêu cầu cập nhật thông tin người dùng
type UpdateUserRequest struct {
	MaxConnection int `json:"maxConnection" binding:"required"`
}

// PasswordRequest đại diện cho yêu cầu đổi mật khẩu
type PasswordRequest struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required"`
}

// ResetPasswordRequest đại diện cho yêu cầu đặt lại mật khẩu
type ResetPasswordRequest struct {
	NewPassword string `json:"newPassword" binding:"required"`
}

// CreateUser xử lý yêu cầu tạo người dùng mới
func CreateUser(c *gin.Context) {
	var req CreateUserRequest

	// Parse request body
	if err := c.ShouldBindJSON(&req); err != nil {
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
	req.MaxConnection = 5
	// Tạo đối tượng User mới
	user := &models.User{
		Username:      req.Username,
		Password:      req.Password,
		MaxConnection: req.MaxConnection,
	}

	// Lưu người dùng vào cơ sở dữ liệu
	err = models.CreateUser(db, user)
	if err != nil {
		if err.Error() == "username already exists" {
			c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  "User created successfully",
		"username": user.Username,
	})
}

// UpdateUser xử lý yêu cầu cập nhật thông tin người dùng
func UpdateUser(c *gin.Context) {
	username := c.Param("username")
	var req UpdateUserRequest

	// Parse request body
	if err := c.ShouldBindJSON(&req); err != nil {
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

	// Cập nhật thông tin người dùng
	user := &models.User{
		MaxConnection: req.MaxConnection,
	}

	err = models.UpdateUser(db, user)
	if err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "User updated successfully",
		"username": username,
	})
}

// DeleteUserRequest đại diện cho yêu cầu xóa người dùng
type DeleteUserRequest struct {
	Password string `json:"password" binding:"required"`
}

// DeleteUser xử lý yêu cầu xóa người dùng
func DeleteUser(c *gin.Context) {
	username := c.Param("username")

	usernameLogin, exists := c.Get("username")

	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	log.Println(usernameLogin)

	var req DeleteUserRequest

	// Parse request body
	if err := c.ShouldBindJSON(&req); err != nil {
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

	// xác thực user đang login với password
	user, err := models.GetUserByUsername(db, usernameLogin.(string))
	if err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user"})
		}
		return
	}
	log.Println(user)
	// lấy password của user đang login trong db
	userPassword := user.Password

	// xác thực password với password gửi lên
	if models.HashPassword(req.Password) != userPassword {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Incorrect password"})
		return
	}

	// Xóa người dùng
	err = models.DeleteUser(db, username)
	if err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else if err.Error() == "incorrect password" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Incorrect password"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "User deleted successfully",
		"username": username,
	})
}

// ResetPassword xử lý yêu cầu đặt lại mật khẩu cho người dùng
func ResetPassword(c *gin.Context) {
	username := c.Param("username")
	var req ResetPasswordRequest

	// Parse request body
	if err := c.ShouldBindJSON(&req); err != nil {
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

	// Đặt lại mật khẩu
	err = models.ResetPassword(db, username, req.NewPassword)
	if err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset password"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Password reset successfully",
		"username": username,
	})
}

// ChangePassword xử lý yêu cầu đổi mật khẩu của người dùng đang đăng nhập
func ChangePassword(c *gin.Context) {
	// Lấy username từ context (đã được set bởi middleware)
	username, exists := c.Get("username")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req PasswordRequest

	// Parse request body
	if err := c.ShouldBindJSON(&req); err != nil {
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

	// Đổi mật khẩu
	err = models.ChangePassword(db, username.(string), req.OldPassword, req.NewPassword)
	if err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else if err.Error() == "incorrect password" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Incorrect password"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to change password"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Password changed successfully",
		"username": username,
	})
}
