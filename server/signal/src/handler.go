package src

import (
	"goRTCServer/pkg/logger"
	"goRTCServer/pkg/proto"
	"goRTCServer/pkg/utils"
	"goRTCServer/server/signal/ws"
)

// handlerWebSocket 信令处理
func handlerWebSocket(method string, peer *ws.Peer, msg map[string]interface{}, accept ws.AcceptFunc, reject ws.RejectFunc) {
	switch method {
	case proto.ClientToSignalJoin:
		join(peer, msg, accept, reject)
	case proto.ClientToSignalLeave:
		leave(peer, msg, accept, reject)
	case proto.ClientToSignalKeepAlive:
		keepalive(peer, msg, accept, reject)
	case proto.ClientToSignalPublish:
		publish(peer, msg, accept, reject)
	case proto.ClientToSignalUnPublish:
		unpublish(peer, msg, accept, reject)
	case proto.ClientToSignalSubscribe:
		subscribe(peer, msg, accept, reject)
	case proto.ClientToSignalUnSubscribe:
		unsubscribe(peer, msg, accept, reject)
	case proto.ClientToSignalBroadcast:
		broadcast(peer, msg, accept, reject)
	case proto.ClientToSignalGetRoomUsers:
		getusers(peer, msg, accept, reject)
	case proto.ClientToSignalGetRoomPubs:
		getpubs(peer, msg, accept, reject)
	default:
		ws.DefaultReject(codeUnknownErr, codeStr(codeUnknownErr))
	}
}

/*
  "request":true
  "id":3764139
  "method":"join"
  "data":{
    "rid":"room"
  }
*/
// 用户加入房间
func join(peer *ws.Peer, msg map[string]interface{}, accept ws.AcceptFunc, reject ws.RejectFunc) {
	if invalid(msg, "rid", reject) {
		return
	}
	uid := peer.ID()
	rid := utils.Val(msg, "rid")

	// 获取register服务器的RPC句柄
	registerRPC := GetRPCHandlerByServiceName("register")
	if registerRPC == nil {
		reject(codeRegisterRPCErr, codeStr(codeRegisterRPCErr))
		return
	}
	// 1. 查询uid是否在房间内
	resp, err := registerRPC.SyncRequest(proto.SignalToRegisterGetSignalInfo, utils.Map("rid", rid, "uid", uid))
	rmp := utils.Unmarshal(string(resp))
	if err == nil {
		signalId := rmp["signalid"].(string)
		if signalId != signalNode.NodeInfo().NodeID {
			// 1.1 不在当前节点 通知其他节点关闭
			rpcSignal := rpcs[signalId]
			if rpcSignal != nil {
				rpcSignal.SyncRequest(proto.SignalToSignalOnKick, utils.Map("rid", rid, "uid", uid))
			}
		} else {
			// 1.2 user 在当前节点
			// 删除数据库流信息
			resp, err := registerRPC.SyncRequest(proto.SignalToRegisterOnStreamRemove, utils.Map("rid", rid, "uid", uid, "mid", ""))
			rmp = utils.Unmarshal(string(resp))
			if err != nil {
				rmPubs, ok := rmp["rmPubs"].([]interface{})
				if ok {
					SendNotifyByUids(rid, uid, proto.SignalToSignalOnStreamRemove, rmPubs)
				}
			} else {
				logger.Errorf("signal.join request register streamRemove err, err is %v", err.Reason)
			}
			// 删除数据库用户信息
			_, err = registerRPC.SyncRequest(proto.SignalToRegisterOnLeave, utils.Map("rid", rid, "uid", uid))
			if err != nil {
				logger.Errorf("signal.join request register userLeave err:%s", err.Reason)
			}

			// 发送广播给所有人
			SendNotifyByUids(rid, uid, proto.SignalToClientOnLeave, rmp["rmPubs"].([]interface{}))
			// 通知本地对象
			SendNotifyByUid(rid, uid, proto.SignalToClientOnLeave, map[string]interface{}{"rid": rid, "uid": uid})
			// 删除本地对象
			room := rooms.GetRoom(rid)
			if room != nil {
				room.DelPeer(uid)
			}
		}

	}
	// 2.重新进房
	room := rooms.GetRoom(rid)
	room.AddPeer(peer)
	// 3.写数据库
	resp, err = registerRPC.SyncRequest(proto.SignalToRegisterOnJoin, utils.Map("rid", rid, "uid", uid, "signalId", signalNode.NodeInfo().NodeID))

	if err != nil {
		reject(err.Code, err.Reason)
		return
	}
	// 4.广播通知房间内其他人
	SendNotifyByUids(rid, uid, proto.SignalToSignalOnJoin, []interface{}{utils.Unmarshal(string(resp))})

	_, users := FindRoomUsers(rid, uid)
	_, pubs := FindRoomPubs(rid, uid)
	res := utils.Map("users", users, "pubs", pubs)
	accept([]byte(utils.Marshal(res)))
}

/*
  "request":true
  "id":3764139
  "method":"leave"
  "data":{
	"rid":"room"
  }
*/
// leave 离开房间
func leave(peer *ws.Peer, msg map[string]interface{}, accept ws.AcceptFunc, reject ws.RejectFunc) {

}

/*
  "request":true
  "id":3764139
  "method":"keepalive"
  "data":{
    "rid":"room",
  }
*/
// keepalive 保活
func keepalive(peer *ws.Peer, msg map[string]interface{}, accept ws.AcceptFunc, reject ws.RejectFunc) {

}

/*
  "request":true
  "id":3764139
  "method":"publish"
  "data":{
	"rid":"room",
	"jsep": {
		"type": "offer",
		"sdp": "..."},
		"minfo": {
	  		"audio": true,
	  		"video": true,
			"audiotype": 0,
			"videotype": 0,
	  }
  }
*/
// publish 发布流
func publish(peer *ws.Peer, msg map[string]interface{}, accept ws.AcceptFunc, reject ws.RejectFunc) {

}

/*
  "request":true
  "id":3764139
  "method":"unpublish"
  "data":{
	"rid": "room",
	"mid": "64236c21-21e8-4a3d-9f80-c767d1e1d67f#ABCDEF",
	"sfuid":"shenzhen-sfu-1", (可选)
  }
*/
// unpublish 取消发布流
func unpublish(peer *ws.Peer, msg map[string]interface{}, accept ws.AcceptFunc, reject ws.RejectFunc) {

}

/*
  "request":true
  "id":3764139
  "method":"subscribe"
  "data":{
  	"rid":"room",
    "mid": "64236c21-21e8-4a3d-9f80-c767d1e1d67f#ABCDEF",
	"jsep": {
		"type": "offer",
		"sdp":"..."
	},
	"sfuid":"shenzhen-sfu-1", (可选)
  }
*/
// subscribe 订阅流
func subscribe(peer *ws.Peer, msg map[string]interface{}, accept ws.AcceptFunc, reject ws.RejectFunc) {

}

/*
  "request":true
  "id":3764139
  "method":"unsubscribe"
  "data":{
	"rid": "room",
    "mid": "64236c21-21e8-4a3d-9f80-c767d1e1d67f#ABCDEF"
	"sid":"64236c21-21e8-4a3d-9f80-c767d1e1d67f#ABCDEF"
	"sfuid":"shenzhen-sfu-1", (可选)
  }
*/
// unsubscribe 取消订阅流
func unsubscribe(peer *ws.Peer, msg map[string]interface{}, accept ws.AcceptFunc, reject ws.RejectFunc) {

}

/*
	"request":true
	"id":3764139
	"method":"broadcast"
	"data":{
		"rid": "room",
		"data": "$date"
	}
*/
// broadcast 客户端发送广播给对方
func broadcast(peer *ws.Peer, msg map[string]interface{}, accept ws.AcceptFunc, reject ws.RejectFunc) {

}

/*
	"request":true
	"id":3764139
	"method":"getusers"
	"data":{
		"rid": "room",
	}
*/
// 获取房间其他用户数据
func getusers(peer *ws.Peer, msg map[string]interface{}, accept ws.AcceptFunc, reject ws.RejectFunc) {

}

/*
	"request":true
	"id":3764139
	"method":"getpubs"
	"data":{
		"rid": "room",
	}
*/
// 获取房间其他用户流数据
func getpubs(peer *ws.Peer, msg map[string]interface{}, accept ws.AcceptFunc, reject ws.RejectFunc) {

}
