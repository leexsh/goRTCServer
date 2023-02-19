package ws

import (
	"encoding/json"
	"goRTCServer/pkg/logger"
	"net/http"

	"github.com/cloudwebrtc/go-protoo/peer"
	"github.com/cloudwebrtc/go-protoo/transport"
	"github.com/gorilla/websocket"
)

// 接受函数定义
type AcceptFunc peer.AcceptFunc

// 拒绝函数定义
type RejectFunc peer.RejectFunc

// error code
const (
	ErrInvalidMethod = "method not found"
	ErrInvalidData   = "data not found"
)

// 默认接受处理
func DefaultAccept(data json.RawMessage) {

	logger.LogKf.Debugf("Websocket accept data, data is %v", string(data))
}

// 默认拒绝处理
func DefaultReject(erroCode int, errorReason string) {
	logger.LogKf.Debugf("websocket reject errcode is %v, errorReason is %v", erroCode, errorReason)
}

// websocket 配置信息
type WebSocketServerConfig struct {
	Host          string
	Port          string
	CertFile      string
	KeyFile       string
	HTMLRoot      string
	WebsocketPath string
}

func DefaultConfig() WebSocketServerConfig {
	return WebSocketServerConfig{
		Host:          "0.0.0.0",
		Port:          "8081",
		HTMLRoot:      ".",
		WebsocketPath: "/ws",
	}
}

// websocket 对象
type WebSocketServer struct {
	handleWebSocket func(ws *transport.WebSocketTransport, req *http.Request)
	upgrader        websocket.Upgrader
}

// 新建一个websocket对象
func NewWebSocketServer(handler func(ws *transport.WebSocketTransport, req *http.Request)) *WebSocketServer {
	var server = &WebSocketServer{
		handleWebSocket: handler,
	}
	server.upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	return server
}

func (w *WebSocketServer) handleWebSocketRequest(writer http.ResponseWriter, req *http.Request) {
	respHeader := http.Header{}
	socket, err := w.upgrader.Upgrade(writer, req, respHeader)
	if err != nil {
		panic(err)
	}
	wsTransPort := transport.NewWebSocketTransport(socket)
	w.handleWebSocket(wsTransPort, req)
	wsTransPort.ReadLoop()
}

func (w *WebSocketServer) Bind(cfg WebSocketServerConfig) {
	http.HandleFunc(cfg.WebsocketPath, w.handleWebSocketRequest)
	logger.LogKf.Debugf("non-TLS websocketserver listening on : %d, %d", cfg.Host, cfg.Port)
	panic(http.ListenAndServe(cfg.Host+":"+cfg.Port, nil))
}
