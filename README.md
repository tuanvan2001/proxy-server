# SOCKS5 Proxy Server với MySQL Authentication

Đây là một proxy server SOCKS5 được viết bằng Go với các tính năng sau:

## Tính năng

- **Hỗ trợ IPv4 và IPv6**: Xử lý địa chỉ đích IPv4 (ATYP=1) và IPv6 (ATYP=4)
- **Xác thực username/password qua MySQL**: Hỗ trợ phương thức xác thực 0x02 theo RFC 1929 với dữ liệu người dùng từ MySQL
- **Giới hạn số lượng kết nối đồng thời**: Mỗi người dùng có giới hạn số kết nối tối đa riêng
- **Phân giải tên miền**: Xử lý tên miền (ATYP=3) thông qua resolver DNS của Go
- **Giới hạn tốc độ mạng**: Sử dụng gói golang.org/x/time/rate để giới hạn băng thông (100 KB/s)
- **Logging chi tiết**: Sử dụng gói log/slog để ghi log các sự kiện xác thực và kết nối

## Cài đặt

### Yêu cầu

- Go 1.18 trở lên
- MySQL Server

### Thiết lập cơ sở dữ liệu

1. Tạo cơ sở dữ liệu MySQL:

```sql
CREATE DATABASE proxy_server;
USE proxy_server;
```

2. Chạy script SQL để tạo bảng và dữ liệu mẫu:

```bash
mysql -u root -p proxy_server < table.sql
```

### Cài đặt và chạy proxy server

```bash
# Clone repository
git clone <repository-url>
cd proxy-server

# Tải các dependency
go mod tidy

# Biên dịch
go build

# Chạy server
./proxy-server
```

Hoặc chạy trực tiếp không cần biên dịch:

```bash
go run main.go
```

## Cấu hình

### Cổng lắng nghe

Proxy server mặc định chạy trên cổng 1080. Bạn có thể thay đổi cổng này bằng cách chỉnh sửa dòng sau trong file `main.go`:

```go
server := NewProxyServer(":1080")
```

### Kết nối MySQL

Chỉnh sửa thông tin kết nối MySQL trong file `main.go`:

```go
db, err := sql.Open("mysql", "root:password@tcp(127.0.0.1:3306)/proxy_server")
```

## Quản lý người dùng

Proxy server sử dụng MySQL để lưu trữ và xác thực người dùng. Bảng `user` trong cơ sở dữ liệu `proxy_server` chứa thông tin người dùng với các trường sau:

- `username`: Tên đăng nhập (khóa chính)
- `password`: Mật khẩu đã được mã hóa MD5
- `maxConnection`: Số lượng kết nối đồng thời tối đa cho phép
- `createdAt`: Thời gian tạo tài khoản
- `updatedAt`: Thời gian cập nhật tài khoản gần nhất

Các tài khoản mặc định được tạo trong file `table.sql`:

- Username: `user1`, Password: `password1`, Max Connections: 5
- Username: `admin`, Password: `admin123`, Max Connections: 10
- Username: `tuan`, Password: `tuan123`, Max Connections: 8

### Thêm hoặc sửa đổi người dùng

Bạn có thể thêm hoặc sửa đổi người dùng bằng các câu lệnh SQL:

```sql
-- Thêm người dùng mới
INSERT INTO user (username, password, maxConnection) VALUES ('newuser', MD5('password'), 5);

-- Cập nhật mật khẩu
UPDATE user SET password = MD5('newpassword') WHERE username = 'username';

-- Cập nhật số lượng kết nối tối đa
UPDATE user SET maxConnection = 10 WHERE username = 'username';
```

## Sử dụng

Bạn có thể cấu hình các ứng dụng hoặc trình duyệt để sử dụng proxy SOCKS5 này:

- **Địa chỉ**: localhost (hoặc 127.0.0.1)
- **Cổng**: 1080
- **Loại proxy**: SOCKS5
- **Xác thực**: Bật xác thực và sử dụng một trong các tài khoản được cấu hình

## Giới hạn tốc độ

Proxy server giới hạn tốc độ truyền dữ liệu ở mức 100 KB/s cho mỗi kết nối. Bạn có thể điều chỉnh giới hạn này bằng cách thay đổi các hằng số sau trong file `main.go`:

```go
RATE_LIMIT = 100 * 1024 // bytes per second
BURST_LIMIT = 50 * 1024 // burst size
```

## Cấu trúc mã nguồn

- **main.go**: Chứa toàn bộ mã nguồn của proxy server
  - Xử lý giao thức SOCKS5
  - Xác thực username/password qua MySQL
  - Giới hạn số lượng kết nối đồng thời
  - Phân giải tên miền
  - Giới hạn tốc độ
  - Logging
- **table.sql**: Script SQL để tạo bảng user và dữ liệu mẫu

## Giao thức SOCKS5

Proxy server tuân thủ giao thức SOCKS5 theo RFC 1928 và RFC 1929:

1. **Handshake**: Client kết nối và thương lượng phương thức xác thực
2. **Xác thực**: Sử dụng phương thức username/password theo RFC 1929
3. **Yêu cầu**: Client gửi yêu cầu kết nối đến địa chỉ đích
4. **Phản hồi**: Server thiết lập kết nối và gửi phản hồi
5. **Truyền dữ liệu**: Server chuyển tiếp dữ liệu giữa client và đích

## Giấy phép

MIT