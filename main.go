package main

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

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

	// Rate limiting (100 KB/s)
	RATE_LIMIT  = 100 * 1024 // bytes per second
	BURST_LIMIT = 50 * 1024  // burst size
)

// User credentials for authentication
type Credentials struct {
	Username string
	Password string
}

// ProxyServer represents our SOCKS5 proxy server
type ProxyServer struct {
	Addr        string
	Logger      *slog.Logger
	Credentials map[string]string
	mutex       sync.RWMutex
	connections map[string]string // Maps client address to authenticated username
	connMutex   sync.RWMutex
}

// NewProxyServer creates a new SOCKS5 proxy server
func NewProxyServer(addr string) *ProxyServer {
	// Setup logger
	logDir := "log"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		slog.Default().Error("Failed to create log directory", "error", err)
		os.Exit(1)
	}

	logFileName := fmt.Sprintf("access_%s.log", time.Now().Format("2006_01_02"))
	logFilePath := filepath.Join(logDir, logFileName)

	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		slog.Default().Error("Failed to open log file", "path", logFilePath, "error", err)
		os.Exit(1)
	}

	logOpts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	logHandler := slog.NewTextHandler(logFile, logOpts)
	logger := slog.New(logHandler)

	// Create server
	return &ProxyServer{
		Addr:   addr,
		Logger: logger,
		Credentials: map[string]string{
			"user1": "password1", // Default credentials
			"admin": "admin123",
			"tuan":  "tuan123",
		},
		connections: make(map[string]string),
	}
}

// Start starts the proxy server
func (s *ProxyServer) Start() error {
	listener, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}
	defer listener.Close()

	s.Logger.Info("SOCKS5 proxy server started", "address", s.Addr)

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
		// Remove connection from the map when done
		s.connMutex.Lock()
		delete(s.connections, clientAddr)
		s.connMutex.Unlock()
	}()

	s.Logger.Info("New connection", "client", clientAddr)

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

	s.mutex.RLock()
	expectedPassword, exists := s.Credentials[usernameStr]
	s.mutex.RUnlock()

	var authStatus byte = 0x01 // Failure by default
	if exists && expectedPassword == passwordStr {
		authStatus = 0x00 // Success
		s.Logger.Info("Authentication successful", "username", usernameStr)

		// Store the authenticated username for this connection
		clientAddr := conn.RemoteAddr().String()
		s.connMutex.Lock()
		s.connections[clientAddr] = usernameStr
		s.connMutex.Unlock()
	} else {
		s.Logger.Warn("Authentication failed", "username", usernameStr)
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
		username, exists := s.connections[clientAddr]
		s.connMutex.RUnlock()

		if exists {
			s.Logger.Info("Domain resolved", "domain", dstAddr, "ip", dstIP.String(), "user", username)
		} else {
			s.Logger.Info("Domain resolved", "domain", dstAddr, "ip", dstIP.String(), "user", "unknown")
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
	s.Logger.Info("Connecting to destination", "address", dstAddrPort)

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
	s.Logger.Info("Connection established", "source", conn.RemoteAddr(), "destination", dstAddrPort)
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

// proxyData handles bidirectional data transfer with rate limiting
func (s *ProxyServer) proxyData(client, target net.Conn) {
	// Create rate limiters for both directions
	clientLimiter := rate.NewLimiter(rate.Limit(RATE_LIMIT), BURST_LIMIT)
	targetLimiter := rate.NewLimiter(rate.Limit(RATE_LIMIT), BURST_LIMIT)

	wg := &sync.WaitGroup{}
	wg.Add(2)

	// Client -> Target
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		transferred := int64(0)

		for {
			n, err := client.Read(buf)
			if n > 0 {
				// Apply rate limiting
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

		s.Logger.Info("Connection closed", "direction", "client->target", "bytes", transferred)
	}()

	// Target -> Client
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		transferred := int64(0)

		for {
			n, err := target.Read(buf)
			if n > 0 {
				// Apply rate limiting
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

		s.Logger.Info("Connection closed", "direction", "target->client", "bytes", transferred)
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
