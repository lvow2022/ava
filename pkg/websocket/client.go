package websocket

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// WSConfig WebSocket 客户端配置
type WSConfig struct {
	// URL WebSocket 服务器地址（ws:// 或 wss://）
	URL string

	// Headers 连接时发送的 HTTP 头
	Headers http.Header

	// TLSConfig TLS 配置（用于 wss:// 连接）
	// 如果为 nil，使用默认的 TLS 配置
	TLSConfig *tls.Config

	// DialTimeout 连接超时时间（默认 5 秒）
	DialTimeout time.Duration

	// HandshakeTimeout 握手超时时间（默认 10 秒）
	HandshakeTimeout time.Duration
}

// WsClient WebSocket 客户端接口
type WsClient interface {
	Recv(ctx context.Context) ([]byte, error)
	Send(ctx context.Context, data []byte) error
	Close() error
	Done() <-chan struct{}
}

type wsClient struct {
	conn *websocket.Conn

	recvCh chan []byte
	sendCh chan []byte

	done chan struct{}

	closeOnce sync.Once
	err       atomic.Value // 保存 error
}

// NewWsClient 创建新的 WebSocket 客户端并建立连接
func NewWsClient(ctx context.Context, config WSConfig) (WsClient, error) {
	if config.DialTimeout == 0 {
		config.DialTimeout = 5 * time.Second
	}
	if config.HandshakeTimeout == 0 {
		config.HandshakeTimeout = 10 * time.Second
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: config.HandshakeTimeout,
	}

	if config.TLSConfig != nil {
		dialer.TLSClientConfig = config.TLSConfig
	}

	dialCtx, cancel := context.WithTimeout(ctx, config.DialTimeout)
	defer cancel()

	conn, resp, err := dialer.DialContext(dialCtx, config.URL, config.Headers)
	if err != nil {
		if conn != nil {
			_ = conn.Close()
		}
		return nil, fmt.Errorf("dial websocket: %w, resp: %v", err, resp)
	}

	// 创建客户端
	c := &wsClient{
		conn:   conn,
		recvCh: make(chan []byte, 128),
		sendCh: make(chan []byte, 128),
		done:   make(chan struct{}),
	}

	go c.readLoop()
	go c.writeLoop()

	return c, nil
}

func (c *wsClient) readLoop() {
	defer c.Close()

	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			c.setErr(err)
			return
		}

		select {
		case c.recvCh <- msg:
		case <-c.done:
			return
		}
	}
}

func (c *wsClient) writeLoop() {
	defer c.Close()

	for {
		select {
		case msg := <-c.sendCh:
			if err := c.conn.WriteMessage(websocket.BinaryMessage, msg); err != nil {
				c.setErr(err)
				return
			}
		case <-c.done:
			return
		}
	}
}

func (c *wsClient) Recv(ctx context.Context) ([]byte, error) {
	select {
	case msg := <-c.recvCh:
		return msg, nil

	case <-c.done:
		if err := c.getErr(); err != nil {
			return nil, err
		}
		return nil, io.EOF

	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *wsClient) Send(ctx context.Context, msg []byte) error {
	select {
	case c.sendCh <- msg:
		return nil

	case <-c.done:
		return io.EOF

	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *wsClient) Close() error {
	c.closeOnce.Do(func() {
		close(c.done)
		_ = c.conn.Close()
	})
	return nil
}

func (c *wsClient) Done() <-chan struct{} {
	return c.done
}

func (c *wsClient) setErr(err error) {
	if err != nil {
		c.err.Store(err)
	}
}

func (c *wsClient) getErr() error {
	if v := c.err.Load(); v != nil {
		return v.(error)
	}
	return nil
}
