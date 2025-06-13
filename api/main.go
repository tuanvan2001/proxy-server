package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/tuantech/proxy-server/api/config"
	"github.com/tuantech/proxy-server/api/controllers"
	"github.com/tuantech/proxy-server/api/database"
	"github.com/tuantech/proxy-server/api/middleware"
)

func main() {
	// Khởi tạo kết nối database
	db, err := database.InitDB()
	if err != nil {
		log.Fatalf("Không thể kết nối đến cơ sở dữ liệu: %v", err)
	}
	defer db.Close()

	// Khởi tạo router
	r := gin.Default()

	// Đăng ký các route
	r.POST("/login", controllers.Login)

	// Các route yêu cầu xác thực
	auth := r.Group("/api")
	auth.Use(middleware.AuthMiddleware())
	{
		// User routes
		auth.GET("/users", controllers.GetUsers)
		auth.POST("/users", controllers.CreateUser)
		auth.PUT("/users/:username", controllers.UpdateUser)
		auth.DELETE("/users/:username", controllers.DeleteUser)
		auth.PUT("/users/:username/password", controllers.ResetPassword)
		auth.PUT("/change-password", controllers.ChangePassword)
	}

	// Khởi động server
	log.Printf("Server đang chạy tại cổng %s", config.ServerPort)
	r.Run(":" + config.ServerPort)
}