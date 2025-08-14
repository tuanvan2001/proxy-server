package config

import (
	"os"
	"strconv"
)

// getEnv returns the value of the environment variable named by key or the fallback value.
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// getEnvInt returns the integer value of the environment variable or the fallback value.
func getEnvInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if v, err := strconv.Atoi(value); err == nil {
			return v
		}
	}
	return fallback
}

// ServerPort là cổng mà API server sẽ lắng nghe
var ServerPort = getEnv("SERVER_PORT", "8080")

// DatabaseConfig chứa thông tin kết nối đến cơ sở dữ liệu
var (
	DBDriver   = getEnv("DB_DRIVER", "mysql")
	DBUser     = getEnv("DB_USER", "root")
	DBPassword = getEnv("DB_PASSWORD", "Tuan123")
	DBName     = getEnv("DB_NAME", "proxy")
	DBHost     = getEnv("DB_HOST", "127.0.0.1")
	DBPort     = getEnv("DB_PORT", "3306")
)

// JWTConfig chứa các cấu hình liên quan đến JWT
var (
	JWTSecret     = getEnv("JWT_SECRET", "Oegjsc1029384756") // Khóa bí mật để ký JWT
	JWTExpiration = getEnvInt("JWT_EXPIRATION", 24)          // Thời gian hết hạn của token (giờ)
)

// AdminUsers là danh sách các tài khoản được phép sử dụng API
var AdminUsers = []string{
	"admin", // Tài khoản admin mặc định
	// Thêm các tài khoản admin khác nếu cần
}
