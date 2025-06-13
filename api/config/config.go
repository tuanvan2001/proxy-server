package config

// ServerPort là cổng mà API server sẽ lắng nghe
const ServerPort = "8080"

// DatabaseConfig chứa thông tin kết nối đến cơ sở dữ liệu
const (
	DBDriver   = "mysql"
	DBUser     = "root"
	DBPassword = "Tuan123"
	DBName     = "proxy"
	DBHost     = "127.0.0.1"
	DBPort     = "3306"
)

// JWTConfig chứa các cấu hình liên quan đến JWT
const (
	JWTSecret     = "Oegjsc1029384756" // Khóa bí mật để ký JWT
	JWTExpiration = 24                        // Thời gian hết hạn của token (giờ)
)

// AdminUsers là danh sách các tài khoản được phép sử dụng API
var AdminUsers = []string{
	"admin", // Tài khoản admin mặc định
	// Thêm các tài khoản admin khác nếu cần
}