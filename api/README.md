# API Quản lý Người dùng cho SOCKS5 Proxy Server

API này cung cấp các endpoint để quản lý người dùng cho SOCKS5 Proxy Server, bao gồm đăng nhập, tạo người dùng mới, cập nhật thông tin người dùng, đổi mật khẩu và đặt lại mật khẩu.

## Cài đặt

### Yêu cầu

- Go 1.18 trở lên
- MySQL Server

### Cài đặt và chạy API server

```bash
# Di chuyển vào thư mục api
cd /path/to/proxy-server/api

# Tải các dependency
go mod tidy

# Biên dịch
go build

# Chạy server
./api
```

Hoặc chạy trực tiếp không cần biên dịch:

```bash
go run main.go
```

## Cấu hình

Cấu hình của API được lấy từ các biến môi trường sau (giá trị mặc định trong ngoặc):

| Biến | Mặc định | Mô tả |
|------|----------|------|
| `SERVER_PORT` | `8080` | Cổng lắng nghe của API |
| `DB_DRIVER` | `mysql` | Loại cơ sở dữ liệu |
| `DB_USER` | `root` | Tên người dùng cơ sở dữ liệu |
| `DB_PASSWORD` | `Tuan123` | Mật khẩu cơ sở dữ liệu |
| `DB_NAME` | `proxy` | Tên cơ sở dữ liệu |
| `DB_HOST` | `127.0.0.1` | Địa chỉ máy chủ cơ sở dữ liệu |
| `DB_PORT` | `3306` | Cổng cơ sở dữ liệu |
| `JWT_SECRET` | `Oegjsc1029384756` | Khóa bí mật ký JWT |
| `JWT_EXPIRATION` | `24` | Thời hạn token (giờ) |

Ví dụ thiết lập trước khi chạy API server:

```bash
export DB_USER=myuser
export DB_PASSWORD=mypass
export JWT_SECRET=mysecret
```

`AdminUsers` vẫn được định nghĩa trong `config/config.go`.

## API Endpoints

### Đăng nhập

```
POST /login
```

**Request Body:**

```json
{
  "username": "admin",
  "password": "Tuandev2001"
}
```

**Response:**

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires": "2023-06-01T15:04:05Z"
}
```

### Lấy danh sách người dùng (có phân trang và tìm kiếm)

```
GET /api/users?page=1&pageSize=10&search=admin
```

**Headers:**

```
Authorization: Bearer {token}
```

**Query Parameters:**

- `page`: Số trang (mặc định: 1)
- `pageSize`: Số lượng người dùng mỗi trang (mặc định: 10, tối đa: 100)
- `search`: Từ khóa tìm kiếm theo username (tùy chọn)

**Response:**

```json
{
  "users": [
    {
      "username": "admin",
      "maxConnection": 10,
      "createdAt": "2023-05-01T10:00:00Z",
      "updatedAt": "2023-05-01T10:00:00Z"
    },
    {
      "username": "user1",
      "maxConnection": 5,
      "createdAt": "2023-05-02T11:00:00Z",
      "updatedAt": "2023-05-02T11:00:00Z"
    }
  ],
  "total": 2,
  "page": 1,
  "pageSize": 10,
  "totalPages": 1
}
```

### Tạo người dùng mới

```
POST /api/users
```

**Headers:**

```
Authorization: Bearer {token}
```

**Request Body:**

```json
{
  "username": "newuser",
  "password": "password123",
  "maxConnection": 5
}
```

**Response:**

```json
{
  "message": "User created successfully",
  "username": "newuser"
}
```

### Cập nhật thông tin người dùng

```
PUT /api/users/{username}
```

**Headers:**

```
Authorization: Bearer {token}
```

**Request Body:**

```json
{
  "maxConnection": 10
}
```

**Response:**

```json
{
  "message": "User updated successfully",
  "username": "username"
}
```

### Xóa người dùng

```
DELETE /api/users/{username}
```

**Headers:**

```
Authorization: Bearer {token}
Content-Type: application/json
```

**Request Body:**

```json
{
  "password": "password123"
}
```

**Response:**

```json
{
  "message": "User deleted successfully",
  "username": "username"
}
```

### Đặt lại mật khẩu người dùng

```
PUT /api/users/{username}/password
```

**Headers:**

```
Authorization: Bearer {token}
```

**Request Body:**

```json
{
  "newPassword": "newpassword123"
}
```

**Response:**

```json
{
  "message": "Password reset successfully",
  "username": "username"
}
```

### Đổi mật khẩu của người dùng đang đăng nhập

```
PUT /api/change-password
```

**Headers:**

```
Authorization: Bearer {token}
```

**Request Body:**

```json
{
  "oldPassword": "oldpassword",
  "newPassword": "newpassword123"
}
```

**Response:**

```json
{
  "message": "Password changed successfully",
  "username": "username"
}
```

## Bảo mật

- API sử dụng JWT (JSON Web Token) để xác thực người dùng
- Chỉ những tài khoản được liệt kê trong `AdminUsers` mới có thể đăng nhập và sử dụng API
- Mật khẩu được mã hóa bằng MD5 trước khi lưu vào cơ sở dữ liệu
- Tất cả các endpoint quản lý người dùng đều yêu cầu xác thực JWT