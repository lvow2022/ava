package ws

import (
	"context"
	"net/http"

	"github.com/gorilla/websocket"
)

// EventHandler 定义 WS 消息事件回调
type EventHandler interface {
	OnOpen(c *WSClient)
	OnMessage(c *WSClient, msgType int, msg []byte)
	OnError(c *WSClient, err error)
	OnClose(c *WSClient)
}

// WSClient 通用 WebSocket 客户端
type WSClient struct {
	conn    *websocket.Conn
	handler EventHandler
	ctx     context.Context
	cancel  context.CancelFunc
	writeCh chan wsMessage
}

type wsMessage struct {
	msgType int
	data    []byte
}

// NewWSClient 创建 WSClient 并初始化通道
func NewWSClient(url string, header http.Header, handler EventHandler) (*WSClient, error) {
	conn, _, err := websocket.DefaultDialer.Dial(url, header)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	client := &WSClient{
		conn:    conn,
		handler: handler,
		ctx:     ctx,
		cancel:  cancel,
		writeCh: make(chan wsMessage, 100), // 缓冲写队列
	}

	handler.OnOpen(client)

	go client.readLoop()
	go client.writeLoop()

	return client, nil
}

// readLoop 持续读取消息
func (c *WSClient) readLoop() {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			msgType, msg, err := c.conn.ReadMessage()
			if err != nil {
				c.handler.OnError(c, err)
				c.Close()
				return
			}
			c.handler.OnMessage(c, msgType, msg)
		}
	}
}

// writeLoop 持续写消息
func (c *WSClient) writeLoop() {
	for {
		select {
		case <-c.ctx.Done():
			return
		case msg := <-c.writeCh:
			err := c.conn.WriteMessage(msg.msgType, msg.data)
			if err != nil {
				c.handler.OnError(c, err)
				c.Close()
				return
			}
		}
	}
}

// 底层统一方法
func (c *WSClient) send(msgType int, data []byte) {
	select {
	case c.writeCh <- wsMessage{msgType: msgType, data: data}:
	case <-c.ctx.Done():
	}
}

func (c *WSClient) SendText(text string) {
	c.send(websocket.TextMessage, []byte(text))
}

func (c *WSClient) SendBinary(data []byte) {
	c.send(websocket.BinaryMessage, data)
}

// Close 优雅关闭连接
func (c *WSClient) Close() {
	c.cancel()
	c.conn.Close()
	c.handler.OnClose(c)
}
