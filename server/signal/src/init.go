package src

import (
	"goRTCServer/pkg/etcd"
	"goRTCServer/pkg/logger"
	"goRTCServer/pkg/proto"
	"goRTCServer/pkg/utils"
	"goRTCServer/server/signal/conf"
	"goRTCServer/server/signal/ws"
	"net/http"
	"time"

	nprotoo "github.com/cloudwebrtc/nats-protoo"
	"github.com/sirupsen/logrus"
)

const (
	statCycle = 10 * time.Second
)

var (
	rooms      *ws.Rooms
	signalNode *etcd.ServiceNode
	watch      *etcd.ServiceWatcher
	signalNats *nprotoo.NatsProtoo
	caster     *nprotoo.Broadcaster
	rpcs       = make(map[string]*nprotoo.Requestor)
)

func init() {
	logger.DoInit(conf.Kafka.URL, "rtc_signal")
	logger.LogKf.SetLevel(logrus.DebugLevel)
}

// 启动服务
func Start() {
	rooms = ws.NewRooms()
	// 服务注册
	signalNode = etcd.NewServiceNode(conf.Etcd.Adds, conf.Global.NodeDC, conf.Global.NodeID, conf.Global.Name)
	signalNode.RegisterNode()
	// 服务发现
	watch = etcd.NewServiceWatcher(conf.Etcd.Adds)
	go watch.WatchServiceNode("", watchServiceCallBack)
	// 消息注册
	signalNats = nprotoo.NewNatsProtoo(conf.Nats.URL)
	signalNats.OnRequest(signalNode.GetRPCChannel(), handleRPCMsg)
	// 消息广播
	caster = signalNats.NewBroadcaster(signalNode.GetEventChannel())
	// 启动websocket
	InitSignalServer(conf.Signal.Host, conf.Signal.Port, conf.Signal.Cert, conf.Signal.Key)
	// 启动房间资源回收
	go CheckRoom()
	// 启动调试
	if conf.Global.Pprof != "" {
		go debug()
	}

}

func Stop() {
	if signalNats != nil {
		signalNats.Close()
	}
	if signalNode != nil {
		signalNode.Close()
	}
	if watch != nil {
		watch.Close()
	}
}

func debug() {
	logger.Debugf("start signal server on %s", conf.Global.Pprof)
	http.ListenAndServe(conf.Global.Pprof, nil)
}

// watchServiceCallBack 查看所有的Node节点
func watchServiceCallBack(state int32, n etcd.Node) {
	if state == etcd.ServerUp {
		// 新增节点
		// 判断是否为广播节点
		if n.Name == "signal" {
			if n.NodeID != signalNode.NodeInfo().NodeID {
				eventID := etcd.GetEventChannel(n)
				signalNats.OnBroadcast(eventID, handleBroadcast)
			}
		}
		if n.Name == "sdu" {
			eventId := etcd.GetEventChannel(n)
			signalNats.OnBroadcast(eventId, handleBroadcast)
		}
		id := n.NodeID
		_, found := rpcs[id]
		if !found {
			rpcID := etcd.GetPRCChannel(n)
			rpcs[id] = signalNats.NewRequestor(rpcID)
		}
	} else if state == etcd.ServerDown {
		delete(rpcs, n.NodeID)
	} else {

	}
}

// GetRPCHandlerByServiceName 通过服务名获取RPC handler
func GetRPCHandlerByServiceName(name string) *nprotoo.Requestor {
	var node *etcd.Node
	services, find := watch.GetNodes(name)
	if find {
		for _, server := range services {
			node = &server
			break
		}
	}
	if node != nil {
		rpc, find := rpcs[node.NodeID]
		if find {
			return rpc
		}
	}
	return nil
}

// GetRPCHandlerByNodeID 获取指定id的RPC Handler
func GetRPCHandlerByNodeId(nid string) *nprotoo.Requestor {
	node, find := watch.GetNodeByID(nid)
	if !find {
		return nil
	}
	if node != nil {
		rpc, ok := rpcs[node.NodeID]
		if ok {
			return rpc
		}
	}
	return nil
}

// GetRPCHandlerByPayload 获取负载最低的RPC handler
func GetRPCHandlerByPayload(name string) (*nprotoo.Requestor, string) {
	node, ok := watch.GetNodeByPayload(signalNode.NodeInfo().NodeDC, name)
	if !ok {
		return nil, ""
	}
	rpc, find := rpcs[node.NodeID]
	if find {
		return rpc, node.NodeID
	}
	return nil, ""
}

// GetExistByUid 根据rid uid判断人是否在线,
func GetExistByUid(rid, uid string) bool {
	registerRPC := GetRPCHandlerByServiceName("register")
	if registerRPC == nil {
		logger.Errorf("GetExistByUid cannot  get available register node")
		return false
	}
	resp, err := registerRPC.SyncRequest(proto.SignalToRegisterGetSignalInfo, utils.Map("rid", rid, "uid", uid))
	if err != nil {
		logger.Errorf(err.Reason)
		return false
	}
	signalId := utils.Val(utils.Unmarshal(string(resp)), "signalid")
	if signalId != "" {
		if signalId == signalNode.NodeInfo().NodeID {
			return true
		} else {
			signal := GetRPCHandlerByNodeId(signalId)
			return (signal != nil)
		}
	}
	return false
}

// GetSFURPCHandlerByMID 根据rid mid获取sfu节点的rpc句柄
func GetSFURPCHandlerByMID(rid, mid string) *nprotoo.Requestor {
	registerRPC := GetRPCHandlerByServiceName("register")
	if registerRPC == nil {
		logger.Errorf("GetSFURPCHandlerByMid cannot get available register node")
		return nil
	}
	resp, err := registerRPC.SyncRequest(proto.SignalToRegisterGetSfuInfo, utils.Map("rid", rid, "mid", mid))
	if err != nil {
		logger.Errorf(err.Reason)
		return nil
	}
	logger.Infof("GetSFURPCHandlerByMID success, resp is %v", resp)
	var sfu *nprotoo.Requestor
	sfuid := utils.Val(utils.Unmarshal(string(resp)), "sfuid")
	if sfuid != "" {
		sfu = GetRPCHandlerByNodeId(sfuid)
	}
	return sfu
}

/*
	"method" proto. "rid" rid "uid" uid
*/
// 获取房间内其他用户的数据
func FindRoomUsers(rid, uid string) (bool, interface{}) {
	registerRPC := GetRPCHandlerByServiceName("register")
	if registerRPC == nil {
		logger.Errorf("FindRoomUsers cannot get available register node")
		return false, nil
	}
	resp, err := registerRPC.SyncRequest(proto.SignalToRegisterGetRoomUsers, utils.Map("rid", rid, "uid", uid))
	if err != nil {
		logger.Errorf(err.Reason)
		return false, nil
	}
	logger.Infof("FindRoomUsers success, resp is %v", resp)
	mp := utils.Unmarshal(string(resp))
	if mp["users"] == nil {
		logger.Errorf("findRoomUsers users is nil")
		return false, nil
	}
	users := mp["users"].([]interface{})
	return true, users
}

// FindRoomPubs 获取房间内其他用户流信息
func FindRoomPubs(rid, uid string) (bool, []interface{}) {
	registerRPC := GetRPCHandlerByServiceName("register")
	if registerRPC == nil {
		logger.Errorf("FindRoomPubs cannot get available register node")
		return false, nil
	}
	resp, err := registerRPC.SyncRequest(proto.SignalToRegisterGetRoomPubs, utils.Map("rid", rid, "uid", uid))
	if err != nil {
		logger.Errorf(err.Reason)
		return false, nil
	}
	logger.Infof("FindRoomPubs success, resp is %v", resp)
	mp := utils.Unmarshal(string(resp))
	if mp["pubs"] == nil {
		logger.Errorf("FindRoomPubs pubs is nil")
		return false, nil
	}
	pubs := mp["pubs"].([]interface{})
	return true, pubs
}

// CheckRoom 检查所有的房间
func CheckRoom() {
	t := time.NewTicker(statCycle)
	defer t.Stop()
	for range t.C {
		for rid, room := range rooms.GetRooms() {
			for uid := range room.GetPeers() {
				exist := GetExistByUid(rid, uid)
				if !exist {
					// 获取register RPC句柄
					regigstRPC := GetRPCHandlerByServiceName("register")
					if regigstRPC == nil {
						continue
					}
					// 删除数据库流
					resp, err := regigstRPC.SyncRequest(proto.SignalToRegisterOnStreamRemove, utils.Map("rid", rid, "uid", uid, "mid", ""))
					rmp := utils.Unmarshal(string(resp))
					if err == nil {
						rmPubs, ok := rmp["rmPubs"].(map[string]interface{})
						if ok {
							SendNotifyByUid(rid, uid, proto.SignalToClientOnStreamRemove, rmPubs)
						}
					} else {
						logger.Errorf("signal.checkRoom request register streamRemove err, err is %v", err.Reason)
					}
					// 删除数据库中用户
					resp, err = regigstRPC.SyncRequest(proto.SignalToRegisterOnLeave, utils.Map("rid", rid, "uid", uid))
					rmp = utils.Unmarshal(string(resp))
					if err != nil {

						SendNotifyByUid(rid, uid, proto.SignalToClientOnLeave, rmp)
					} else {
						logger.Errorf("signal.checkRoom request register UserRemove err, err is %v", err.Reason)
					}
					// 删除本地对象
					room.DelPeer(uid)
					logger.Debugf("room = %s, del peer uid = %s", rid, uid)
				}
			}
			if len(room.GetPeers()) == 0 {
				logger.Debugf("no peer in room, room=%s now", rid)
				rooms.DelRoom(rid)
			}
		}
	}
}

// SendNotifyByUid 单发广播给其他人
func SendNotifyByUid(rid, skipUid, method string, msg map[string]interface{}) {
	NotifyPeersWithoutID(rid, skipUid, method, msg)
	caster.Say(method, msg)
}

// SendNotifyByUids 群发广播给其他人
func SendNotifyByUids(rid, skipUid, method string, msgs []interface{}) {
	for _, msg := range msgs {
		data, ok := msg.(map[string]interface{})
		if ok {
			SendNotifyByUid(rid, skipUid, method, data)
		}
	}
}
