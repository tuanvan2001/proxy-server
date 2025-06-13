package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/tuantech/proxy-server/api/config"
)

// AuthMiddleware là middleware xác thực JWT
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Lấy token từ header Authorization
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			c.Abort()
			return
		}

		// Kiểm tra định dạng Bearer token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header format must be Bearer {token}"})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// Parse và xác thực token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Kiểm tra phương thức ký
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(config.JWTSecret), nil
		})

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Kiểm tra token có hợp lệ không
		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			// Lấy username từ claims
			username, ok := claims["username"].(string)
			if !ok {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
				c.Abort()
				return
			}

			// Kiểm tra xem username có trong danh sách AdminUsers không
			isAdmin := false
			for _, adminUser := range config.AdminUsers {
				if username == adminUser {
					isAdmin = true
					break
				}
			}

			if !isAdmin {
				c.JSON(http.StatusForbidden, gin.H{"error": "User not authorized to access this API"})
				c.Abort()
				return
			}

			// Lưu username vào context để sử dụng trong các handler
			c.Set("username", username)
			c.Next()
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}
	}
}