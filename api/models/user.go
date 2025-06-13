package models

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"time"
)

// User đại diện cho một người dùng trong hệ thống
type User struct {
	Username      string    `json:"username"`
	Password      string    `json:"password,omitempty"`
	MaxConnection int       `json:"maxConnection"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// PaginatedUsers đại diện cho kết quả phân trang danh sách người dùng
type PaginatedUsers struct {
	Users      []*User `json:"users"`
	Total      int     `json:"total"`
	Page       int     `json:"page"`
	PageSize   int     `json:"pageSize"`
	TotalPages int     `json:"totalPages"`
}

// HashPassword mã hóa mật khẩu bằng MD5
func HashPassword(password string) string {
	hash := md5.Sum([]byte(password))
	return hex.EncodeToString(hash[:])
}

// GetUserByUsername lấy thông tin người dùng theo username
func GetUserByUsername(db *sql.DB, username string) (*User, error) {
	user := &User{}
	query := "SELECT username, password, maxConnection, createdAt, updatedAt FROM user WHERE username = ?"
	err := db.QueryRow(query, username).Scan(
		&user.Username,
		&user.Password,
		&user.MaxConnection,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	return user, nil
}

// CreateUser tạo một người dùng mới
func CreateUser(db *sql.DB, user *User) error {
	// Kiểm tra xem người dùng đã tồn tại chưa
	_, err := GetUserByUsername(db, user.Username)
	if err == nil {
		return errors.New("username already exists")
	} else if err.Error() != "user not found" {
		return err
	}

	// Mã hóa mật khẩu
	hashedPassword := HashPassword(user.Password)

	// Thêm người dùng mới
	query := "INSERT INTO user (username, password, maxConnection) VALUES (?, ?, ?)"
	_, err = db.Exec(query, user.Username, hashedPassword, user.MaxConnection)
	return err
}

// UpdateUser cập nhật thông tin người dùng
func UpdateUser(db *sql.DB, user *User) error {
	// Kiểm tra xem người dùng có tồn tại không
	_, err := GetUserByUsername(db, user.Username)
	if err != nil {
		return err
	}

	// Cập nhật thông tin người dùng
	query := "UPDATE user SET maxConnection = ? WHERE username = ?"
	_, err = db.Exec(query, user.MaxConnection, user.Username)
	return err
}

// DeleteUser xóa người dùng
func DeleteUser(db *sql.DB, username string) error {
	// Kiểm tra xem người dùng có tồn tại không
	user, err := GetUserByUsername(db, username)
	if err != nil {
		return err
	}

	// Xóa người dùng
	query := "DELETE FROM user WHERE username = ?"
	_, err = db.Exec(query, user.Username)
	return err
}

// ChangePassword thay đổi mật khẩu của người dùng
func ChangePassword(db *sql.DB, username, oldPassword, newPassword string) error {
	// Lấy thông tin người dùng
	user, err := GetUserByUsername(db, username)
	if err != nil {
		return err
	}

	// Kiểm tra mật khẩu cũ
	if HashPassword(oldPassword) != user.Password {
		return errors.New("incorrect password")
	}

	// Cập nhật mật khẩu mới
	query := "UPDATE user SET password = ? WHERE username = ?"
	_, err = db.Exec(query, HashPassword(newPassword), username)
	return err
}

// ResetPassword đặt lại mật khẩu của người dùng
func ResetPassword(db *sql.DB, username, newPassword string) error {
	// Kiểm tra xem người dùng có tồn tại không
	_, err := GetUserByUsername(db, username)
	if err != nil {
		return err
	}

	// Cập nhật mật khẩu mới
	query := "UPDATE user SET password = ? WHERE username = ?"
	_, err = db.Exec(query, HashPassword(newPassword), username)
	return err
}

// GetAllUsers lấy danh sách tất cả người dùng
func GetAllUsers(db *sql.DB) ([]User, error) {
	rows, err := db.Query("SELECT username, password, maxConnection, createdAt, updatedAt FROM user")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		err := rows.Scan(&user.Username, &user.Password, &user.MaxConnection, &user.CreatedAt, &user.UpdatedAt)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

// GetUsersPaginated lấy danh sách người dùng có phân trang và tìm kiếm theo username
func GetUsersPaginated(db *sql.DB, page, pageSize int, search string) (*PaginatedUsers, error) {
	// Đảm bảo page và pageSize hợp lệ
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	// Tính offset cho phân trang
	offset := (page - 1) * pageSize

	// Xây dựng câu truy vấn với điều kiện tìm kiếm nếu có
	countQuery := "SELECT COUNT(1) FROM user"
	query := "SELECT username, maxConnection, createdAt, updatedAt FROM user"

	var args []interface{}
	if search != "" {
		searchPattern := "%" + search + "%"
		countQuery += " WHERE username LIKE ?"
		query += " WHERE username LIKE ?"
		args = append(args, searchPattern)
	}

	// Thêm phân trang
	query += " ORDER BY username LIMIT ? OFFSET ?"
	args = append(args, pageSize, offset)

	// Đếm tổng số người dùng
	var total int
	row := db.QueryRow(countQuery, args[:len(args)-2]...)
	if err := row.Scan(&total); err != nil {
		return nil, fmt.Errorf("error counting users: %v", err)
	}

	// Tính tổng số trang
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	// Lấy danh sách người dùng theo phân trang
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("error querying users: %v", err)
	}
	defer rows.Close()

	users := []*User{}
	for rows.Next() {
		user := &User{}
		err := rows.Scan(
			&user.Username,
			&user.MaxConnection,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning user: %v", err)
		}
		users = append(users, user)
	}

	return &PaginatedUsers{
		Users:      users,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}
