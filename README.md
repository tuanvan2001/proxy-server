# SOCKS5 Proxy Server

Đây là một proxy server SOCKS5 được viết bằng Go với các tính năng sau:

## Tính năng

- **Hỗ trợ IPv4 và IPv6**: Xử lý địa chỉ đích IPv4 (ATYP=1) và IPv6 (ATYP=4)
- **Xác thực username/password**: Hỗ trợ phương thức xác thực 0x02 theo RFC 1929
- **Phân giải tên miền**: Xử lý tên miền (ATYP=3) thông qua resolver DNS của Go
- **Giới hạn tốc độ mạng**: Sử dụng gói golang.org/x/time/rate để giới hạn băng thông (100 KB/s)
- **Logging chi tiết**: Sử dụng gói log/slog để ghi log các sự kiện xác thực và kết nối

## Cài đặt

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

Proxy server mặc định chạy trên cổng 1080. Bạn có thể thay đổi cổng này bằng cách chỉnh sửa dòng sau trong file `main.go`:

```go
server := NewProxyServer(":1080")
```

## Tài khoản mặc định

Proxy server được cấu hình với các tài khoản mặc định sau:

- Username: `user1`, Password: `password1`
- Username: `admin`, Password: `admin123`

Bạn có thể thêm hoặc sửa đổi các tài khoản này trong hàm `NewProxyServer` trong file `main.go`.

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
  - Xác thực username/password
  - Phân giải tên miền
  - Giới hạn tốc độ
  - Logging

## Giao thức SOCKS5

Proxy server tuân thủ giao thức SOCKS5 theo RFC 1928 và RFC 1929:

1. **Handshake**: Client kết nối và thương lượng phương thức xác thực
2. **Xác thực**: Sử dụng phương thức username/password theo RFC 1929
3. **Yêu cầu**: Client gửi yêu cầu kết nối đến địa chỉ đích
4. **Phản hồi**: Server thiết lập kết nối và gửi phản hồi
5. **Truyền dữ liệu**: Server chuyển tiếp dữ liệu giữa client và đích

## Giấy phép

MIT