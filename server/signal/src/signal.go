package src

import (
	"goRTCServer/pkg/logger"
	"goRTCServer/pkg/utils"
	"goRTCServer/server/signal/ws"
	"net/http"

	"github.com/cloudwebrtc/go-protoo/transport"
)

// InitSignalServer 初始化ws
func InitSignalServer(host string, port string, cert, key string) {
	config := ws.DefaultConfig()
	config.Host = host
	config.Port = port
	config.CertFile = cert
	config.KeyFile = key
	wsServer := ws.NewWebSocketServer(handler)
	go wsServer.Bind(config)
}

func handler(transport *transport.WebSocketTransport, request *http.Request) {
	logger.Debugf("handler = %v", request.URL.Query())
	vars := request.URL.Query()
	peerID := vars["peer"]
	if peerID == nil || len(peerID) < 1 {
		return
	}
	id := peerID[0]
	peer := ws.NewPeer(id, transport)

	handleRequest := func(req map[string]interface{}, accept ws.AcceptFunc, reject ws.RejectFunc) {
		defer utils.Recover("signal handleRequest")

		method := utils.Val(req, "method")
		if method == "" {
			reject(-1, ws.ErrInvalidMethod)
			return
		}
		data := req["data"]
		if data == nil {
			reject(-1, ws.ErrInvalidData)
			return
		}
		msg := data.(map[string]interface{})
		handlerWebSocket(method, peer, msg, accept, reject)
	}

	handleNotification := func(notification map[string]interface{}) {
		defer utils.Recover("signal handleNotification")

		method := utils.Val(notification, "method")
		if method == "" {
			ws.DefaultReject(-1, ws.ErrInvalidMethod)
			return
		}
		data := notification["data"]
		if data == nil {
			ws.DefaultReject(-1, ws.ErrInvalidData)
			return
		}
		msg := data.(map[string]interface{})
		handlerWebSocket(method, peer, msg, ws.DefaultAccept, ws.DefaultReject)
	}
	handleClose := func(codoe int, err string) {
		peer.Close()
	}
	peer.On("req", handleRequest)
	peer.On("notification", handleNotification)
	peer.On("close", handleClose)
	peer.On("error", handleClose)
}
