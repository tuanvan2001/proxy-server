package main

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/time/rate"
)

const (
	// SOCKS5 protocol constants
	SOCKS_VERSION = 0x05

	// Authentication methods
	NO_AUTH                = 0x00
	USERNAME_PASSWORD_AUTH = 0x02
	NO_ACCEPTABLE_METHODS  = 0xFF

	// Command types
	CONNECT = 0x01
	BIND    = 0x02
	UDP     = 0x03

	// Address types
	IPV4_ADDRESS   = 0x01
	DOMAIN_ADDRESS = 0x03
	IPV6_ADDRESS   = 0x04

	// Reply codes
	SUCCEEDED                = 0x00
	GENERAL_FAILURE          = 0x01
	CONNECTION_NOT_ALLOWED   = 0x02
	NETWORK_UNREACHABLE      = 0x03
	HOST_UNREACHABLE         = 0x04
	CONNECTION_REFUSED       = 0x05
	TTL_EXPIRED              = 0x06
	COMMAND_NOT_SUPPORTED    = 0x07
	ADDRESS_TYPE_UNSUPPORTED = 0x08

	// Rate limiting (100 KB/s) - Tạm thời vô hiệu hóa giới hạn băng thông
	// RATE_LIMIT  = 100 * 1024 * 1024 // bytes per second
	// BURST_LIMIT = 1024 * 1024       // burst size
	// Đặt giá trị rất cao để vô hiệu hóa giới hạn băng thông
	RATE_LIMIT  = 1000 * 1024 * 1024 // 1 GB/s - thực tế là không giới hạn
	BURST_LIMIT = 100 * 1024 * 1024  // 100 MB burst - thực tế là không giới hạn
)

// User credentials for authentication
type User struct {
	Username      string
	Password      string
	MaxConnection int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// ProxyServer represents our SOCKS5 proxy server
type ProxyServer struct {
	Addr            string
	Logger          *slog.Logger
	DB              *sql.DB
	mutex           sync.RWMutex
	connections     map[string]string // Maps client address to authenticated username
	userConnections map[string]int    // Maps username to number of active connections
	connMutex       sync.RWMutex
}

// NewProxyServer creates a new SOCKS5 proxy server
func NewProxyServer(addr string) *ProxyServer {
	// Setup logger - chỉ log ra console
	logOpts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	logHandler := slog.NewTextHandler(os.Stdout, logOpts)
	logger := slog.New(logHandler)

	// Connect to MySQL database
	db, err := sql.Open("mysql", "root:Tuan123@tcp(127.0.0.1:3306)/proxy")
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		logger.Error("Failed to ping database", "error", err)
		os.Exit(1)
	}

	logger.Info("Connected to MySQL database")

	// Create server
	return &ProxyServer{
		Addr:            addr,
		Logger:          logger,
		DB:              db,
		connections:     make(map[string]string),
		userConnections: make(map[string]int),
	}
}

// Start starts the proxy server
func (s *ProxyServer) Start() error {
	listener, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}
	defer listener.Close()
	defer s.DB.Close()

	// s.Logger.Info("SOCKS5 proxy server started", "address", s.Addr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			s.Logger.Error("Failed to accept connection", "error", err)
			continue
		}

		go s.handleConnection(conn)
	}
}

// handleConnection processes a client connection
func (s *ProxyServer) handleConnection(conn net.Conn) {
	clientAddr := conn.RemoteAddr().String()

	// Cleanup connection data when done
	defer func() {
		conn.Close()
		// Lưu ý: Việc giảm số lượng kết nối đã được xử lý trong hàm proxyData
		// Chỉ xóa kết nối khỏi map nếu chưa được xử lý bởi proxyData
		s.connMutex.Lock()
		_, exists := s.connections[clientAddr]
		if exists {
			// Kết nối vẫn còn trong map, có thể do lỗi xảy ra trước khi proxyData được gọi
			// hoặc proxyData không được gọi (ví dụ: lỗi xác thực)
			username := s.connections[clientAddr]
			// Decrease connection count for this user
			s.userConnections[username]--
			// s.Logger.Info("Connection closed in handleConnection, decreasing count", "username", username, "connections", s.userConnections[username])
			if s.userConnections[username] <= 0 {
				delete(s.userConnections, username)
			}
			delete(s.connections, clientAddr)
		}
		s.connMutex.Unlock()
	}()

	// s.Logger.Info("New connection", "client", clientAddr)

	// Perform SOCKS5 handshake
	if err := s.handleHandshake(conn); err != nil {
		s.Logger.Error("Handshake failed", "client", clientAddr, "error", err)
		return
	}

	// Process client request
	if err := s.handleRequest(conn); err != nil {
		s.Logger.Error("Request failed", "client", clientAddr, "error", err)
		return
	}
}

// handleHandshake performs the SOCKS5 handshake
func (s *ProxyServer) handleHandshake(conn net.Conn) error {
	// Read the SOCKS version and number of authentication methods
	buf := make([]byte, 2)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return err
	}

	version := buf[0]
	nMethods := buf[1]

	if version != SOCKS_VERSION {
		return fmt.Errorf("unsupported SOCKS version: %d", version)
	}

	// Read authentication methods
	methods := make([]byte, nMethods)
	if _, err := io.ReadFull(conn, methods); err != nil {
		return err
	}

	// Check if username/password authentication is supported
	var methodSelected byte = NO_ACCEPTABLE_METHODS
	for _, method := range methods {
		if method == USERNAME_PASSWORD_AUTH {
			methodSelected = USERNAME_PASSWORD_AUTH
			break
		}
	}

	// Send selected method
	response := []byte{SOCKS_VERSION, methodSelected}
	if _, err := conn.Write(response); err != nil {
		return err
	}

	// If no acceptable method found
	if methodSelected == NO_ACCEPTABLE_METHODS {
		return errors.New("no acceptable authentication methods")
	}

	// Perform username/password authentication
	return s.performAuth(conn)
}

// MD5Hash returns the MD5 hash of a string
func MD5Hash(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}

// performAuth handles username/password authentication
func (s *ProxyServer) performAuth(conn net.Conn) error {
	// Read auth version
	buf := make([]byte, 1)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return err
	}

	if buf[0] != 0x01 { // Username/password auth version
		return fmt.Errorf("unsupported auth version: %d", buf[0])
	}

	// Read username length
	if _, err := io.ReadFull(conn, buf); err != nil {
		return err
	}
	usernameLen := int(buf[0])

	// Read username
	username := make([]byte, usernameLen)
	if _, err := io.ReadFull(conn, username); err != nil {
		return err
	}

	// Read password length
	if _, err := io.ReadFull(conn, buf); err != nil {
		return err
	}
	passwordLen := int(buf[0])

	// Read password
	password := make([]byte, passwordLen)
	if _, err := io.ReadFull(conn, password); err != nil {
		return err
	}

	// Verify credentials
	usernameStr := string(username)
	passwordStr := string(password)
	hashedPassword := MD5Hash(passwordStr)

	// Query the database for user credentials
	var user User
	query := "SELECT username, password, maxConnection FROM user WHERE username = ?"
	err := s.DB.QueryRow(query, usernameStr).Scan(&user.Username, &user.Password, &user.MaxConnection)

	var authStatus byte = 0x01 // Failure by default
	if err == nil && user.Password == hashedPassword {
		// Check if user has reached max connections
		clientAddr := conn.RemoteAddr().String()
		s.connMutex.Lock()
		currentConnections := s.userConnections[usernameStr]

		if currentConnections >= user.MaxConnection {
			s.Logger.Warn("Max connections reached", "username", usernameStr,
				"current", currentConnections, "max", user.MaxConnection)
			s.connMutex.Unlock()

			// Send auth failure response
			response := []byte{0x01, 0x01} // Auth failure
			conn.Write(response)
			return errors.New("max connections reached")
		}

		// Authentication successful
		authStatus = 0x00 // Success
		// s.Logger.Info("Authentication successful", "username", usernameStr)

		// Store the authenticated username for this connection and increment counter
		s.connections[clientAddr] = usernameStr
		s.userConnections[usernameStr] = currentConnections + 1
		// s.Logger.Info("Connection established", "username", usernameStr,
		// 	"connections", s.userConnections[usernameStr], "max", user.MaxConnection)
		s.connMutex.Unlock()
	} else {
		// s.Logger.Warn("Authentication failed", "username", usernameStr, "error", err)
	}

	// Send auth response
	response := []byte{0x01, authStatus}
	if _, err := conn.Write(response); err != nil {
		return err
	}

	if authStatus != 0x00 {
		return errors.New("authentication failed")
	}

	return nil
}

// handleRequest processes the client's connection request
func (s *ProxyServer) handleRequest(conn net.Conn) error {
	// Read request header
	buf := make([]byte, 4)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return err
	}

	version := buf[0]
	command := buf[1]
	// buf[2] is reserved
	addrType := buf[3]

	if version != SOCKS_VERSION {
		return fmt.Errorf("unsupported SOCKS version: %d", version)
	}

	// Only support CONNECT command
	if command != CONNECT {
		s.sendReply(conn, COMMAND_NOT_SUPPORTED, nil)
		return fmt.Errorf("unsupported command: %d", command)
	}

	// Parse the destination address based on address type
	var dstAddr string
	var dstIP net.IP

	switch addrType {
	case IPV4_ADDRESS:
		addrBytes := make([]byte, 4)
		if _, err := io.ReadFull(conn, addrBytes); err != nil {
			return err
		}
		dstIP = net.IPv4(addrBytes[0], addrBytes[1], addrBytes[2], addrBytes[3])
		dstAddr = dstIP.String()

	case IPV6_ADDRESS:
		addrBytes := make([]byte, 16)
		if _, err := io.ReadFull(conn, addrBytes); err != nil {
			return err
		}
		dstIP = addrBytes
		dstAddr = dstIP.String()

	case DOMAIN_ADDRESS:
		addrLenBuf := make([]byte, 1)
		if _, err := io.ReadFull(conn, addrLenBuf); err != nil {
			return err
		}
		addrLen := int(addrLenBuf[0])

		domain := make([]byte, addrLen)
		if _, err := io.ReadFull(conn, domain); err != nil {
			return err
		}
		dstAddr = string(domain)

		// Resolve domain name to IP
		ips, err := net.LookupIP(dstAddr)
		if err != nil || len(ips) == 0 {
			s.sendReply(conn, HOST_UNREACHABLE, nil)
			return fmt.Errorf("failed to resolve domain %s: %v", dstAddr, err)
		}

		// Use the first resolved IP
		dstIP = ips[0]

		// Get the authenticated username for this connection
		clientAddr := conn.RemoteAddr().String()
		s.connMutex.RLock()
		_, exists := s.connections[clientAddr]
		s.connMutex.RUnlock()

		if exists {
			// s.Logger.Info("Domain resolved", "domain", dstAddr, "ip", dstIP.String(), "user", username)
		} else {
			// s.Logger.Info("Domain resolved", "domain", dstAddr, "ip", dstIP.String(), "user", "unknown")
		}

	default:
		s.sendReply(conn, ADDRESS_TYPE_UNSUPPORTED, nil)
		return fmt.Errorf("unsupported address type: %d", addrType)
	}

	// Read destination port
	portBuf := make([]byte, 2)
	if _, err := io.ReadFull(conn, portBuf); err != nil {
		return err
	}
	dstPort := binary.BigEndian.Uint16(portBuf)

	// Connect to the destination
	dstAddrPort := fmt.Sprintf("%s:%d", dstAddr, dstPort)
	// s.Logger.Info("Connecting to destination", "address", dstAddrPort)

	dstConn, err := net.DialTimeout("tcp", dstAddrPort, 10*time.Second)
	if err != nil {
		s.Logger.Error("Failed to connect to destination", "address", dstAddrPort, "error", err)

		// Send appropriate error response
		var replyCode byte
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			replyCode = TTL_EXPIRED
		} else if opErr, ok := err.(*net.OpError); ok {
			if opErr.Err.Error() == "connection refused" {
				replyCode = CONNECTION_REFUSED
			} else if opErr.Err.Error() == "network is unreachable" {
				replyCode = NETWORK_UNREACHABLE
			} else {
				replyCode = HOST_UNREACHABLE
			}
		} else {
			replyCode = GENERAL_FAILURE
		}

		s.sendReply(conn, replyCode, nil)
		return err
	}
	defer dstConn.Close()

	// Send success reply
	localAddr := dstConn.LocalAddr().(*net.TCPAddr)
	s.sendReply(conn, SUCCEEDED, localAddr)

	// Start proxying data
	// s.Logger.Info("Connection established", "source", conn.RemoteAddr(), "destination", dstAddrPort)
	s.proxyData(conn, dstConn)

	return nil
}

// sendReply sends a reply to the client
func (s *ProxyServer) sendReply(conn net.Conn, replyCode byte, bindAddr *net.TCPAddr) error {
	// Default bind address and port (used for errors)
	bindIP := net.IPv4(0, 0, 0, 0)
	bindPort := uint16(0)
	var addrType byte = IPV4_ADDRESS

	// If we have a bind address, use it
	if bindAddr != nil {
		if ip4 := bindAddr.IP.To4(); ip4 != nil {
			bindIP = ip4
			addrType = IPV4_ADDRESS
		} else {
			bindIP = bindAddr.IP
			addrType = IPV6_ADDRESS
		}
		bindPort = uint16(bindAddr.Port)
	}

	// Build response
	response := make([]byte, 0, 22) // Max size for IPv6
	response = append(response, SOCKS_VERSION, replyCode, 0x00, addrType)

	// Add bind address based on type
	if addrType == IPV4_ADDRESS {
		response = append(response, bindIP.To4()...)
	} else {
		response = append(response, bindIP.To16()...)
	}

	// Add bind port
	portBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(portBytes, bindPort)
	response = append(response, portBytes...)

	// Send response
	_, err := conn.Write(response)
	return err
}

// proxyData handles bidirectional data transfer with rate limiting (hiện đã vô hiệu hóa giới hạn băng thông)
func (s *ProxyServer) proxyData(client, target net.Conn) {
	// Create rate limiters for both directions (hiện đã đặt giá trị rất cao để vô hiệu hóa giới hạn)
	clientLimiter := rate.NewLimiter(rate.Limit(RATE_LIMIT), BURST_LIMIT)
	targetLimiter := rate.NewLimiter(rate.Limit(RATE_LIMIT), BURST_LIMIT)

	// Lấy thông tin kết nối của client để cập nhật số lượng kết nối khi đóng
	clientAddr := client.RemoteAddr().String()
	s.connMutex.RLock()
	username, userExists := s.connections[clientAddr]
	s.connMutex.RUnlock()

	// Hàm giảm số lượng kết nối khi kết nối bị đóng
	decreaseConnectionCount := func() {
		if userExists {
			s.connMutex.Lock()
			// Kiểm tra lại vì có thể đã bị xóa bởi goroutine khác
			if _, exists := s.connections[clientAddr]; exists {
				// Giảm số lượng kết nối của user
				s.userConnections[username]--
				// s.Logger.Info("Connection closed in proxyData, decreasing count", "username", username, "connections", s.userConnections[username])
				if s.userConnections[username] <= 0 {
					delete(s.userConnections, username)
				}
				delete(s.connections, clientAddr)
			}
			s.connMutex.Unlock()
		}
	}

	wg := &sync.WaitGroup{}
	wg.Add(2)

	// Client -> Target
	go func() {
		defer wg.Done()
		defer decreaseConnectionCount() // Đảm bảo giảm số lượng kết nối khi goroutine kết thúc
		buf := make([]byte, 4096)
		transferred := int64(0)

		for {
			n, err := client.Read(buf)
			if n > 0 {
				// Apply rate limiting (hiện đã vô hiệu hóa)
				// Giữ lại code rate limiting nhưng đã đặt giá trị RATE_LIMIT và BURST_LIMIT rất cao
				if err := clientLimiter.WaitN(context.Background(), n); err != nil {
					s.Logger.Error("Rate limit error", "direction", "client->target", "error", err)
					break
				}

				// Write to target
				if _, err := target.Write(buf[:n]); err != nil {
					s.Logger.Error("Write error", "direction", "client->target", "error", err)
					break
				}

				transferred += int64(n)
			}

			if err != nil {
				if err != io.EOF {
					s.Logger.Error("Read error", "direction", "client->target", "error", err)
				}
				break
			}
		}

		// s.Logger.Info("Connection closed", "direction", "client->target", "bytes", transferred)
	}()

	// Target -> Client
	go func() {
		defer wg.Done()
		defer decreaseConnectionCount() // Đảm bảo giảm số lượng kết nối khi goroutine kết thúc
		buf := make([]byte, 4096)
		transferred := int64(0)

		for {
			n, err := target.Read(buf)
			if n > 0 {
				// Apply rate limiting (hiện đã vô hiệu hóa)
				// Giữ lại code rate limiting nhưng đã đặt giá trị RATE_LIMIT và BURST_LIMIT rất cao
				if err := targetLimiter.WaitN(context.Background(), n); err != nil {
					s.Logger.Error("Rate limit error", "direction", "target->client", "error", err)
					break
				}

				// Write to client
				if _, err := client.Write(buf[:n]); err != nil {
					s.Logger.Error("Write error", "direction", "target->client", "error", err)
					break
				}

				transferred += int64(n)
			}

			if err != nil {
				if err != io.EOF {
					s.Logger.Error("Read error", "direction", "target->client", "error", err)
				}
				break
			}
		}

		// s.Logger.Info("Connection closed", "direction", "target->client", "bytes", transferred)
	}()

	wg.Wait()
}

func main() {
	// Create and start the proxy server
	server := NewProxyServer(":1080")
	err := server.Start()
	if err != nil {
		fmt.Printf("Error starting server: %v\n", err)
		os.Exit(1)
	}
}
