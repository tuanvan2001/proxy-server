# Các lệnh CURL để gọi API trong dự án

## Đăng nhập

```bash
curl -X POST "http://localhost:8080/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin", "password":"Tuandev2001"}'
```

## Lấy danh sách người dùng (có phân trang và tìm kiếm)

```bash
curl -X GET "http://localhost:8080/api/users?page=1&pageSize=10&search=admin" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

## Tạo người dùng mới

```bash
curl -X POST "http://localhost:8080/api/users" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -d '{"username":"newuser", "password":"password123", "maxConnection":5}'
```

## Cập nhật thông tin người dùng

```bash
curl -X PUT "http://localhost:8080/api/users/newuser" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -d '{"maxConnection":10}'
```

## Đặt lại mật khẩu người dùng

```bash
curl -X PUT "http://localhost:8080/api/users/newuser/password" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -d '{"newPassword":"newpassword123"}'
```

## Đổi mật khẩu của người dùng đang đăng nhập

```bash
curl -X PUT "http://localhost:8080/api/change-password" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -d '{"oldPassword":"Tuandev2001", "newPassword":"newadminpassword"}'
```

## Xóa người dùng

```bash
curl -X DELETE "http://localhost:8080/api/users/newuser" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -d '{"password":"password123"}'
```

## Lưu ý

- Thay thế `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...` bằng token JWT thực tế nhận được sau khi đăng nhập
- Các lệnh trên sử dụng host `localhost` và port `8080`, thay đổi nếu cần
- Các lệnh có thể được sao chép và chạy trực tiếp trong terminal