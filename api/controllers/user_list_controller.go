package controllers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/tuantech/proxy-server/api/database"
	"github.com/tuantech/proxy-server/api/models"
)

// GetUsers xử lý yêu cầu lấy danh sách người dùng có phân trang và tìm kiếm
func GetUsers(c *gin.Context) {
	// Lấy tham số phân trang từ query string
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("pageSize", "10")
	search := c.DefaultQuery("search", "")

	// Chuyển đổi tham số sang kiểu số
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	// Kết nối đến cơ sở dữ liệu
	db, err := database.InitDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection error"})
		return
	}
	defer db.Close()

	// Lấy danh sách người dùng theo phân trang và tìm kiếm
	result, err := models.GetUsersPaginated(db, page, pageSize, search)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get users"})
		return
	}

	c.JSON(http.StatusOK, result)
}