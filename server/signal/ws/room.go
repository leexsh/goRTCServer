package ws

import (
	"goRTCServer/pkg/logger"
	"sync"
)

// Room 房间对象
type Room struct {
	id         string
	peers      map[string]*Peer
	peersMutex sync.RWMutex
}

// NewRoom 新建Room对象
func NewRoom(rid string) *Room {
	return &Room{
		id:    rid,
		peers: make(map[string]*Peer),
	}
}

// ID 返回iid
func (r *Room) ID() string {
	return r.id
}

// AddPeer 新增peer
func (r *Room) AddPeer(peer *Peer) {
	uid := peer.ID()
	r.DelPeer(uid)
	r.peersMutex.Lock()
	defer r.peersMutex.Unlock()
	r.peers[uid] = peer
	//
}

// DelPeer 删除peer
func (r *Room) DelPeer(uid string) {
	r.peersMutex.Lock()
	defer r.peersMutex.Unlock()
	if r.peers[uid] != nil {
		r.peers[uid].Close()
		delete(r.peers, uid)
	}
}

// GetPeer 获取Peer
func (r *Room) GetPeer(uid string) *Peer {
	r.peersMutex.Lock()
	defer r.peersMutex.Unlock()
	if peer, ok := r.peers[uid]; ok {
		return peer
	}
	return nil
}

// GetPeers 获取peers
func (r *Room) GetPeers() map[string]*Peer {
	return r.peers
}

// MapPeers 遍历所有的peer
func (r *Room) MapPeers(fn func(string, *Peer)) {
	r.peersMutex.Lock()
	defer r.peersMutex.Unlock()
	for uid, peer := range r.peers {
		fn(uid, peer)
	}
}

// NotifyWithUid 通知房间内的指定人
func (r *Room) NotifyWithUid(uid, method string, data map[string]interface{}) {
	r.peersMutex.Lock()
	defer r.peersMutex.Unlock()
	for id, peer := range r.peers {
		if uid == id {
			peer.Notify(method, data)
			break
		}
	}
}

// NotifyWithoutUid 通知房间里面其他人
func (r *Room) NotifyWithoutUid(fuid, method string, data map[string]interface{}) {
	r.peersMutex.Lock()
	defer r.peersMutex.Unlock()
	for uid, peer := range r.peers {
		if uid != fuid {
			peer.Notify(method, data)
		}
	}
}

// NotifyAll 通知房间里面所有人
func (r *Room) NotifyAll(method string, data map[string]interface{}) {
	r.peersMutex.Lock()
	defer r.peersMutex.Unlock()
	for _, peer := range r.peers {
		peer.Notify(method, data)
	}
}

// Close 关闭room
func (r *Room) Close() {
	r.peersMutex.Lock()
	defer r.peersMutex.Unlock()
	logger.LogKf.Debugf("close Room, room id is %s", r.id)
	for _, peer := range r.peers {
		peer.Close()
	}
}
