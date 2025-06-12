# Luồng Dữ Liệu trong SOCKS5 Proxy Server

## Tổng quan

Tài liệu này mô tả chi tiết luồng dữ liệu từ client thông qua proxy server đến máy chủ đích (destination server). Proxy server này được triển khai theo giao thức SOCKS5 với xác thực username/password.

## Quy trình kết nối và truyền dữ liệu

### 1. Thiết lập kết nối từ Client đến Proxy Server

1. **Khởi tạo kết nối**:
   - Client mở một kết nối TCP đến proxy server (mặc định là cổng 1080).
   - Proxy server chấp nhận kết nối thông qua hàm `listener.Accept()` và tạo một goroutine mới để xử lý kết nối này bằng hàm `handleConnection()`.
   - Thông tin kết nối mới được ghi lại: `s.Logger.Info("New connection", "client", clientAddr)`.

2. **Bắt tay SOCKS5 (SOCKS5 Handshake)**:
   - Client gửi một gói tin chứa phiên bản SOCKS (0x05) và danh sách các phương thức xác thực được hỗ trợ.
   - Proxy server đọc thông tin này và kiểm tra xem phương thức xác thực username/password (0x02) có được hỗ trợ không.
   - Proxy server phản hồi với phiên bản SOCKS và phương thức xác thực đã chọn.

3. **Xác thực Username/Password**:
   - Client gửi thông tin xác thực bao gồm phiên bản xác thực (0x01), độ dài username, username, độ dài password và password.
   - Proxy server kiểm tra thông tin xác thực với dữ liệu trong `s.Credentials`.
   - Nếu xác thực thành công, proxy server lưu thông tin username đã xác thực vào map `s.connections` và gửi phản hồi thành công (0x00).
   - Nếu xác thực thất bại, proxy server gửi phản hồi thất bại (0x01) và đóng kết nối.

### 2. Xử lý yêu cầu kết nối từ Client

1. **Nhận yêu cầu kết nối**:
   - Client gửi yêu cầu kết nối chứa: phiên bản SOCKS, loại lệnh (CONNECT, BIND, UDP), byte dự trữ (0x00), và loại địa chỉ đích.
   - Proxy server chỉ hỗ trợ lệnh CONNECT (0x01), nếu nhận được lệnh khác sẽ trả về lỗi.

2. **Phân tích địa chỉ đích**:
   - Dựa vào loại địa chỉ (IPv4, IPv6, hoặc tên miền), proxy server đọc và phân tích địa chỉ đích.
   - Nếu là tên miền, proxy server sẽ phân giải tên miền thành địa chỉ IP bằng hàm `net.LookupIP()`.
   - Proxy server cũng đọc cổng đích từ 2 byte cuối của yêu cầu.

3. **Kết nối đến máy chủ đích**:
   - Proxy server tạo một chuỗi địa chỉ đích dạng `địa_chỉ:cổng` và ghi log: `s.Logger.Info("Connecting to destination", "address", dstAddrPort)`.
   - Proxy server thiết lập kết nối TCP đến máy chủ đích bằng `net.DialTimeout()` với thời gian chờ là 10 giây.
   - Nếu kết nối thất bại, proxy server gửi mã lỗi phù hợp về cho client (như CONNECTION_REFUSED, NETWORK_UNREACHABLE, HOST_UNREACHABLE, v.v.).

4. **Phản hồi thành công cho Client**:
   - Nếu kết nối thành công, proxy server gửi phản hồi thành công (0x00) kèm theo địa chỉ và cổng bind của kết nối local.
   - Proxy server ghi log: `s.Logger.Info("Connection established", "source", conn.RemoteAddr(), "destination", dstAddrPort)`.

### 3. Truyền dữ liệu hai chiều

1. **Thiết lập luồng dữ liệu**:
   - Proxy server tạo hai goroutine để xử lý dữ liệu hai chiều:
     - Từ Client đến máy chủ đích (client -> target)
     - Từ máy chủ đích đến Client (target -> client)

2. **Giới hạn tốc độ truyền**:
   - Mỗi hướng truyền dữ liệu được áp dụng giới hạn tốc độ (rate limiting) bằng thư viện `rate`:
     - Tốc độ giới hạn: 100 KB/s (RATE_LIMIT)
     - Kích thước burst: 50 KB (BURST_LIMIT)

3. **Luồng dữ liệu từ Client đến máy chủ đích**:
   - Proxy server đọc dữ liệu từ client với buffer 4096 byte.
   - Áp dụng giới hạn tốc độ bằng `clientLimiter.WaitN()`.
   - Ghi dữ liệu đến máy chủ đích.
   - Theo dõi số byte đã truyền.
   - Khi kết nối đóng hoặc gặp lỗi, ghi log: `s.Logger.Info("Connection closed", "direction", "client->target", "bytes", transferred)`.

4. **Luồng dữ liệu từ máy chủ đích đến Client**:
   - Proxy server đọc dữ liệu từ máy chủ đích với buffer 4096 byte.
   - Áp dụng giới hạn tốc độ bằng `targetLimiter.WaitN()`.
   - Ghi dữ liệu đến client.
   - Theo dõi số byte đã truyền.
   - Khi kết nối đóng hoặc gặp lỗi, ghi log: `s.Logger.Info("Connection closed", "direction", "target->client", "bytes", transferred)`.

### 4. Kết thúc kết nối

1. **Đóng kết nối**:
   - Khi một trong hai bên (client hoặc máy chủ đích) đóng kết nối, các goroutine xử lý dữ liệu sẽ kết thúc.
   - Proxy server đợi cả hai goroutine kết thúc bằng `wg.Wait()`.
   - Kết nối đến máy chủ đích được đóng bằng `defer dstConn.Close()`.
   - Kết nối từ client được đóng bằng `defer conn.Close()`.

2. **Dọn dẹp dữ liệu**:
   - Thông tin kết nối được xóa khỏi map `s.connections` bằng `delete(s.connections, clientAddr)`.

## Sơ đồ luồng dữ liệu

```
+--------+                  +---------------+                  +------------------+
|        |  1. Kết nối TCP  |               |  3. Kết nối TCP  |                  |
| Client | ----------------> | Proxy Server | ----------------> | Máy chủ đích     |
|        | <---------------- |               | <---------------- | (Destination)    |
+--------+  4. Dữ liệu      +---------------+  4. Dữ liệu      +------------------+
              hai chiều                          hai chiều
              
              2. Xác thực SOCKS5
              & Yêu cầu kết nối
```

## Chi tiết kỹ thuật

### Các hằng số giao thức SOCKS5

- **Phiên bản SOCKS**: 0x05
- **Phương thức xác thực**: 0x00 (Không xác thực), 0x02 (Username/Password)
- **Loại lệnh**: 0x01 (CONNECT), 0x02 (BIND), 0x03 (UDP)
- **Loại địa chỉ**: 0x01 (IPv4), 0x03 (Tên miền), 0x04 (IPv6)
- **Mã phản hồi**: 0x00 (Thành công), 0x01-0x08 (Các mã lỗi khác nhau)

### Giới hạn tốc độ

- **Tốc độ tối đa**: 100 KB/s cho mỗi kết nối
- **Kích thước burst**: 50 KB

### Xử lý lỗi

- Proxy server xử lý nhiều loại lỗi khác nhau như lỗi kết nối, lỗi đọc/ghi, lỗi phân giải tên miền.
- Mỗi lỗi được ghi log với mức độ phù hợp (Info, Error, Warn).
- Các mã lỗi SOCKS5 được gửi về client để thông báo tình trạng lỗi.

## Kết luận

Proxy server SOCKS5 này cung cấp một cầu nối an toàn giữa client và máy chủ đích, với các tính năng:
- Xác thực người dùng bằng username/password
- Hỗ trợ IPv4, IPv6 và tên miền
- Giới hạn tốc độ truyền dữ liệu
- Ghi log chi tiết cho mục đích giám sát và gỡ lỗi