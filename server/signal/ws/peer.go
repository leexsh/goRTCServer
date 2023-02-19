package ws

import (
	"goRTCServer/pkg/logger"

	peer "github.com/cloudwebrtc/go-protoo/peer"
	"github.com/cloudwebrtc/go-protoo/transport"
)

// peer 对象
type Peer struct {
	peer.Peer
}

// 新建peer对象
func NewPeer(uid string, t *transport.WebSocketTransport) *Peer {
	return &Peer{
		*peer.NewPeer(uid, t),
	}
}

// On 事件处理
func (p *Peer) On(event, listener interface{}) {
	p.On(event, listener)
}

// Close peer关闭
func (p *Peer) Close() {
	logger.LogKf.Debugf("Close Room Peer is %s", p.ID())
	p.Peer.Close()
}
