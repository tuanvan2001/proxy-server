package database

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"github.com/tuantech/proxy-server/api/config"
)

// InitDB khởi tạo kết nối đến cơ sở dữ liệu MySQL
func InitDB() (*sql.DB, error) {
	// Tạo chuỗi kết nối DSN (Data Source Name)
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		config.DBUser,
		config.DBPassword,
		config.DBHost,
		config.DBPort,
		config.DBName)

	// Mở kết nối đến cơ sở dữ liệu
	db, err := sql.Open(config.DBDriver, dsn)
	if err != nil {
		return nil, err
	}

	// Kiểm tra kết nối
	if err := db.Ping(); err != nil {
		return nil, err
	}

	// Cấu hình pool kết nối
	db.SetMaxOpenConns(25) // Số lượng kết nối mở tối đa
	db.SetMaxIdleConns(5)  // Số lượng kết nối rảnh tối đa

	return db, nil
}