package proxy

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"api-gateway/internal/config"
	"api-gateway/pkg/logging"
	"crypto/tls"

	"github.com/fasthttp/websocket"
	fiberws "github.com/gofiber/websocket/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// Tracer for WebSocket proxy
var wsTracer = otel.Tracer("websocket-proxy")

// WebSocketProxy handles WebSocket connections and proxying
type WebSocketProxy struct {
	config *config.Config
	logger *logging.Logger
	dialer *websocket.Dialer
}

// NewWebSocketProxy creates a new WebSocket proxy instance
func NewWebSocketProxy(cfg *config.Config, logger *logging.Logger) (*WebSocketProxy, error) {
	dialer := websocket.Dialer{
		HandshakeTimeout: time.Second * 10,
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		// Enable compression
		EnableCompression: true,
		// Don't verify TLS for internal communication
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		// Enable all subprotocols
		Subprotocols: []string{"websocket"},
		// Custom header generator
		Jar: nil, // Don't use cookies
	}

	return &WebSocketProxy{
		config: cfg,
		logger: logger,
		dialer: &dialer,
	}, nil
}

// ProxyWebSocket handles WebSocket connection proxying
func (p *WebSocketProxy) ProxyWebSocket(c *fiberws.Conn, target string, path string, headers map[string]string, ctx context.Context) error {
	// Debug: Trace context bilgilerini logla
	p.logger.Debug("WebSocket proxy starting with context",
		zap.Bool("context_is_nil", ctx == nil),
		zap.String("target", target),
		zap.String("path", path))

	// Trace context için orijinal context'i saklayalım
	spanCtx := ctx

	// Start a new span for the WebSocket connection
	var span trace.Span
	if spanCtx != nil {
		spanCtx, span = wsTracer.Start(spanCtx, "proxy-websocket-connection")
		defer span.End()

		p.logger.Debug("Created span for WebSocket connection using provided context",
			zap.String("span_name", "proxy-websocket-connection"))
	} else {
		// Eğer context nil ise yeni bir context oluştur
		spanCtx = context.Background()
		spanCtx, span = wsTracer.Start(spanCtx, "proxy-websocket-connection")
		defer span.End()

		p.logger.Debug("Created span for WebSocket connection using new background context",
			zap.String("span_name", "proxy-websocket-connection"))
	}

	// Parse target URL
	targetURL, err := url.Parse(target)
	if err != nil {
		return fmt.Errorf("failed to parse target URL: %w", err)
	}

	// Create WebSocket URL for target
	wsScheme := "ws"
	httpScheme := "http"
	if targetURL.Scheme == "https" {
		wsScheme = "wss"
		httpScheme = "https"
	}

	// Ensure path has leading slash
	if path != "" && !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// Get query parameters if any
	queryParams := ""
	if rawQuery := headers["X-Original-Query"]; rawQuery != "" {
		queryParams = rawQuery
		// Remove from headers to prevent duplication
		delete(headers, "X-Original-Query")
	}

	// Parse the path to handle query parameters correctly
	parsedPath, err := url.Parse(path)
	if err != nil {
		// p.logger.Error("Failed to parse path", "path", path, "error", err)
		parsedPath = &url.URL{Path: path}
	}

	// Combine query parameters if they exist in both path and headers
	if parsedPath.RawQuery != "" && queryParams != "" {
		queryParams = parsedPath.RawQuery
	} else if parsedPath.RawQuery != "" {
		queryParams = parsedPath.RawQuery
	}

	// Create both HTTP and WebSocket URLs
	httpURL := fmt.Sprintf("%s://%s%s", httpScheme, targetURL.Host, parsedPath.Path)
	wsURL := fmt.Sprintf("%s://%s%s", wsScheme, targetURL.Host, parsedPath.Path)
	if queryParams != "" {
		httpURL = fmt.Sprintf("%s?%s", httpURL, queryParams)
		wsURL = fmt.Sprintf("%s?%s", wsURL, queryParams)
	}

	p.logger.Info("Proxying WebSocket connection",
		zap.String("target_http", httpURL),
		zap.String("target_ws", wsURL),
		zap.String("protocol", headers["Sec-WebSocket-Protocol"]))

	// Prepare headers for the target connection
	header := http.Header{}

	// Copy non-WebSocket headers
	for k, v := range headers {
		// Skip empty values and WebSocket specific headers
		if v == "" || strings.HasPrefix(strings.ToLower(k), "sec-websocket-") ||
		   strings.EqualFold(k, "Upgrade") || strings.EqualFold(k, "Connection") {
			continue
		}

		// Handle array-like header values
		if strings.HasPrefix(v, "[") && strings.HasSuffix(v, "]") {
			// Extract values without brackets and split by comma
			values := strings.Split(strings.Trim(v, "[]"), ",")
			for _, val := range values {
				val = strings.TrimSpace(val)
				if val != "" {
					header.Add(k, val)
				}
			}
		} else {
			header.Add(k, v)
		}
	}

	// Set source and host headers
	header.Set("X-Source", "api-gateway")
	header.Set("Host", targetURL.Host)

	// Propagate trace context to outgoing request
	if span != nil {
		otel.GetTextMapPropagator().Inject(spanCtx, propagation.HeaderCarrier(header))
	}

	// Configure dialer for this specific connection
	dialer := *p.dialer
	dialer.HandshakeTimeout = time.Second * 10
	dialer.EnableCompression = true

	// For Socket.IO connections, configure specific settings
	isSocketIO := strings.Contains(wsURL, "/socket.io/")
	if isSocketIO {
		// Socket.IO specific settings
		dialer.Subprotocols = nil // Clear any subprotocols
		dialer.EnableCompression = false // Disable compression for Socket.IO
		p.logger.Debug("Socket.IO connection detected, cleared subprotocols and disabled compression")
	} else if proto := headers["Sec-WebSocket-Protocol"]; proto != "" {
		// Handle array-like protocol values for non-Socket.IO connections
		if strings.HasPrefix(proto, "[") && strings.HasSuffix(proto, "]") {
			protocols := strings.Split(strings.Trim(proto, "[]"), ",")
			cleanProtocols := make([]string, 0)
			for _, p := range protocols {
				if p = strings.TrimSpace(p); p != "" {
					cleanProtocols = append(cleanProtocols, p)
				}
			}
			if len(cleanProtocols) > 0 {
				dialer.Subprotocols = cleanProtocols
			}
		} else {
			dialer.Subprotocols = []string{proto}
		}
	}

	// Log connection attempt details
	p.logger.Debug("WebSocket connection details",
		zap.String("target_http", httpURL),
		zap.String("target_ws", wsURL),
		zap.Any("headers", header),
		zap.Any("protocols", dialer.Subprotocols),
		zap.Bool("is_socket_io", isSocketIO))

	// Connect to target WebSocket server with context timeout
	// Burada her zaman yeni bir background context kullan, trace context'den bağımsız olarak
	dialCtx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	p.logger.Info("Attempting WebSocket connection",
		zap.String("target_url", wsURL),
		zap.Any("headers", header))

	targetConn, resp, err := dialer.DialContext(dialCtx, wsURL, header)
	if err != nil {
		if resp != nil {
			body := make([]byte, 1024)
			n, _ := resp.Body.Read(body)
			p.logger.Error("WebSocket connection failed",
				zap.Int("status", resp.StatusCode),
				zap.Error(err),
				zap.String("target_url", wsURL),
				zap.String("response_body", string(body[:n])),
				zap.Any("response_headers", resp.Header),
				zap.Any("request_headers", header))
		} else {
			p.logger.Error("WebSocket connection failed with no response",
				zap.Error(err),
				zap.String("target_url", wsURL),
				zap.Any("request_headers", header))
		}
		return fmt.Errorf("failed to connect to target WebSocket: %w", err)
	}
	defer targetConn.Close()

	p.logger.Info("WebSocket connection established",
		zap.String("target_url", wsURL))

	// For Socket.IO, wait for the initial handshake message
	if isSocketIO {
		targetConn.SetReadDeadline(time.Now().Add(5 * time.Second))
		messageType, message, err := targetConn.ReadMessage()
		if err != nil {
			p.logger.Error("Socket.IO initial handshake failed",
				zap.Error(err),
				zap.String("target_url", wsURL))
			return fmt.Errorf("Socket.IO handshake failed: %w", err)
		}
		p.logger.Info("Socket.IO initial message received from target",
			zap.Int("messageType", messageType),
			zap.String("message", string(message)),
			zap.String("target_url", wsURL))

		if err := c.WriteMessage(messageType, message); err != nil {
			p.logger.Error("Failed to forward Socket.IO initial message to client",
				zap.Error(err),
				zap.String("target_url", wsURL))
			return fmt.Errorf("failed to forward Socket.IO initial message: %w", err)
		}
		p.logger.Info("Socket.IO initial message forwarded to client",
			zap.Int("messageType", messageType),
			zap.String("message", string(message)),
			zap.String("target_url", wsURL))

		messageType, message, err = c.ReadMessage()
		if err != nil {
			p.logger.Error("Failed to receive client's Socket.IO handshake response",
				zap.Error(err),
				zap.String("target_url", wsURL))
			return fmt.Errorf("failed to receive client's Socket.IO handshake response: %w", err)
		}
		p.logger.Info("Socket.IO client handshake response received",
			zap.Int("messageType", messageType),
			zap.String("message", string(message)),
			zap.String("target_url", wsURL))

		if err := targetConn.WriteMessage(messageType, message); err != nil {
			p.logger.Error("Failed to forward client's Socket.IO handshake response to target",
				zap.Error(err),
				zap.String("target_url", wsURL))
			return fmt.Errorf("failed to forward client's Socket.IO handshake response: %w", err)
		}
		p.logger.Info("Socket.IO client handshake response forwarded to target",
			zap.Int("messageType", messageType),
			zap.String("message", string(message)),
			zap.String("target_url", wsURL))

		// Handshake sonrası normal mesajlaşma için deadline'i kaldır
		targetConn.SetReadDeadline(time.Time{})
		c.SetReadDeadline(time.Time{})
	}

	// Create channels for message passing
	errChan := make(chan error, 2)
	done := make(chan struct{})
	defer close(done)

	// Forward messages from client to target
	go func() {
		defer func() {
			p.logger.Debug("Client to target forwarder stopped")
		}()
		for {
			select {
			case <-done:
				return
			default:
				messageType, message, err := c.ReadMessage()
				if err != nil {
					p.logger.Error("Client WebSocket read error",
						zap.Error(err),
						zap.String("error_type", fmt.Sprintf("%T", err)),
						zap.Bool("is_socket_io", strings.Contains(wsURL, "/socket.io/")))
					errChan <- err
					return
				}
				err = targetConn.WriteMessage(messageType, message)
				if err != nil {
					p.logger.Error("Target WebSocket write error",
						zap.Error(err),
						zap.String("error_type", fmt.Sprintf("%T", err)),
						zap.Bool("is_socket_io", strings.Contains(wsURL, "/socket.io/")))
					errChan <- err
					return
				}
			}
		}
	}()

	// Forward messages from target to client
	go func() {
		defer func() {
			p.logger.Debug("Target to client forwarder stopped")
		}()
		for {
			select {
			case <-done:
				return
			default:
				messageType, message, err := targetConn.ReadMessage()
				if err != nil {
					p.logger.Error("Target WebSocket read error",
						zap.Error(err),
						zap.String("error_type", fmt.Sprintf("%T", err)),
						zap.Bool("is_socket_io", strings.Contains(wsURL, "/socket.io/")))
					errChan <- err
					return
				}
				err = c.WriteMessage(messageType, message)
				if err != nil {
					p.logger.Error("Client WebSocket write error",
						zap.Error(err),
						zap.String("error_type", fmt.Sprintf("%T", err)),
						zap.Bool("is_socket_io", strings.Contains(wsURL, "/socket.io/")))
					errChan <- err
					return
				}
			}
		}
	}()

	// Wait for an error from either goroutine
	err = <-errChan
	if err != nil {
		if websocket.IsCloseError(err,
			websocket.CloseGoingAway,
			websocket.CloseNormalClosure,
			1005, // CloseNoStatus
			websocket.CloseAbnormalClosure) { // 1006
			p.logger.Debug("WebSocket connection closed",
				zap.Error(err),
				zap.String("error_type", fmt.Sprintf("%T", err)),
				zap.Bool("is_socket_io", strings.Contains(wsURL, "/socket.io/")))
		} else {
			p.logger.Error("WebSocket proxy error",
				zap.Error(err),
				zap.String("error_type", fmt.Sprintf("%T", err)),
				zap.String("target_url", wsURL),
				zap.Bool("is_socket_io", strings.Contains(wsURL, "/socket.io/")))
			return fmt.Errorf("WebSocket proxy error: %w", err)
		}
	}

	return nil
}
