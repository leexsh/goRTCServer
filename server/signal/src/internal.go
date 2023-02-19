package src

import (
	"fmt"
	"goRTCServer/pkg/logger"
	"goRTCServer/pkg/proto"
	"goRTCServer/pkg/utils"

	nprotoo "github.com/cloudwebrtc/nats-protoo"
)

// 处理RPC请求 ->reigster
func handleRPCMsg(request nprotoo.Request, accept nprotoo.RespondFunc, reject nprotoo.RejectFunc) {
	go handleRPCRequest(request, accept, reject)
}

func handleRPCRequest(req nprotoo.Request, accept nprotoo.RespondFunc, reject nprotoo.RejectFunc) {
	defer utils.Recover("signal.handleRPCRequest")
	method := req.Method
	data := utils.Unmarshal(string(req.Data))

	var res map[string]interface{}
	err := &nprotoo.Error{Code: 400, Reason: fmt.Sprintf("Unknown method [%s]", method)}

	switch method {
	// 处理和register服务器之间的通信
	case proto.SignalToSignalOnKick:
		res, err = peerKick(data)
	}
	if err != nil {
		reject(err.Code, err.Reason)
	}
	accept(res)
}

// handleBroadcastMsgs 处理广播消息
func handleBroadcast(msg nprotoo.Notification, subj string) {
	defer utils.Recover("biz.handleBroadcast")
	logger.Debugf("signal.handleBroadcast msg, msg = %v", msg)

	method := msg.Method
	mData := msg.Data
	data := utils.Unmarshal(string(mData))
	rid := utils.Val(data, "rid")
	uid := utils.Val(data, "uid")

	switch method {
	case proto.SignalToSignalOnJoin:
		NotifyPeersWithoutID(rid, uid, proto.SignalToClientOnJoin, data)
	case proto.SignalToSignalOnLeave:
		NotifyPeersWithoutID(rid, uid, proto.SignalToClientOnLeave, data)
	case proto.SignalToSignalOnStreamAdd:
		NotifyPeersWithoutID(rid, uid, proto.SignalToClientOnStreamAdd, data)
	case proto.SignalToSignalOnStreamRemove:
		NotifyPeersWithoutID(rid, uid, proto.SignalToClientOnStreamRemove, data)
	case proto.SignalToSignalBroadcast:
		NotifyPeersWithoutID(rid, uid, proto.SignalToClientBroadcast, data)
	case proto.SfuToSignalOnStreamRemove:
		mid := utils.Val(data, "mid")
		sfuRemoveStream(rid, uid, mid)
	}
}

/*
	“method” proto.SignalToSignalOnKick "rid" rid "uid" uid
*/
// 踢出房间
func peerKick(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rid := utils.Val(data, "rid")
	uid := utils.Val(data, "uid")

	// 获取register句柄
	registerRPC := GetRPCHandlerByServiceName("register")
	if registerRPC == nil {
		return nil, &nprotoo.Error{Code: -1, Reason: "can't get available register rpc node"}
	}

	// 删除数据库数据
	resp, err := registerRPC.SyncRequest(proto.SignalToRegisterOnStreamRemove, utils.Map("rid", rid, "uid", uid, "mid", ""))
	rmp := utils.Unmarshal(string(resp))
	if err != nil {
		rmPubs, ok := rmp["rmPubs"].([]interface{})
		if ok {
			SendNotifyByUids(rid, uid, proto.SignalToSignalOnStreamRemove, rmPubs)
		}
	} else {
		logger.Errorf("signal.peerKick request register streamRemove err, err is %v", err.Reason)
	}
	// 删除数据库的用户
	resp, err = registerRPC.SyncRequest(proto.SignalToRegisterOnLeave, utils.Map("rid", rid, "uid", uid))
	rmp = utils.Unmarshal(string(resp))
	if err != nil {
		logger.Errorf("signal.peerKick request register UserRemove err, err is %v", err.Reason)
	}
	// 通知所有人
	SendNotifyByUids(rid, uid, proto.SignalToSignalOnStreamRemove, rmp["rmPubs"].([]interface{}))
	// 通知客户端
	SendNotifyByUid(rid, uid, proto.SignalToRegisterOnLeave, rmp)

	// 删除本地对象
	room := rooms.GetRoom(rid)
	if room != nil {
		room.DelPeer(uid)
	}
	return utils.Map(), nil
}

// handleBroadCastMsgs 处理广播消息
func handleBroadCas(msg map[string]interface{}, subj string) {

}

// 处理sfu移除流
func sfuRemoveStream(rid, uid, mid string) {
	registerRPC := GetRPCHandlerByServiceName("register")
	if registerRPC == nil {
		return
	}
	resp, err := registerRPC.SyncRequest(proto.SignalToRegisterOnStreamRemove, utils.Map("rid", rid, "uid", uid, "mid", mid))
	if err != nil {
		return
	}
	rmp := utils.Unmarshal(string(resp))
	rmPubs, ok := rmp["rmPubs"].([]interface{})
	if ok {
		SendNotifyByUids(rid, uid, proto.SignalToClientOnStreamRemove, rmPubs)
	}
	return
}

// NotifyPeersWithoutID 通知房间内的其他
func NotifyPeersWithoutID(rid, uid, method string, msg map[string]interface{}) {
	rooms.NotifyWithoutUid(rid, uid, method, msg)
}

// NotifyPeerWithID 通知房间内的指定人
func NotifyPeerWithId(rid, uid, method string, msg map[string]interface{}) {
	rooms.NotifyWithUid(rid, uid, method, msg)
}
