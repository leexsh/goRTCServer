package proto

import "strings"

const (
	/*
		client->singal服务器之间通信
	*/
	ClientToSignalJoin         = "join"        // 加入房间
	ClientToSignalLeave        = "leave"       // 离开房间
	ClientToSignalKeepAlive    = "keepalive"   // 保活
	ClientToSignalPublish      = "publish"     // 发布流
	ClientToSignalUnPublish    = "unpublish"   // 取消发布流
	ClientToSignalSubscribe    = "subscribe"   // 订阅流
	ClientToSignalUnSubscribe  = "unsubscribe" // 取消订阅流
	ClientToSignalBroadcast    = "broadcast"   // 广播
	ClientToSignalGetRoomUsers = "getusers"    // 获取房间内用户数据
	ClientToSignalGetRoomPubs  = "getpubs"     // 获取房间内用户流信息

	/*
		signal->client通信
	*/
	SignalToClientOnJoin         = "peer_join"     // 有用户加入房间
	SignalToClientOnLeave        = "peer_leave"    // 有用户离开房间
	SignalToClientOnStreamAdd    = "stream_add"    // 有人发布流
	SignalToClientOnStreamRemove = "stream_remove" // 有人取消发布
	SignalToClientBroadcast      = "broadcast"     // 有人发送广播
	SignalToClientOnKick         = "peer_kick"     // 被服务器踢下线

	/*
		signal->signal通信
	*/
	SignalToSignalOnJoin         = SignalToClientOnJoin         // 有用户加入房间
	SignalToSignalOnLeave        = SignalToClientOnLeave        // 有用户离开房间
	SignalToSignalOnStreamAdd    = SignalToClientOnStreamAdd    // 有用户发布流
	SignalToSignalOnStreamRemove = SignalToClientOnStreamRemove // 有用户取消发布
	SignalToSignalBroadcast      = SignalToClientBroadcast      // 有用户发广播
	SignalToSignalOnKick         = SignalToClientOnKick         // 被服务端踢下线

	/*
		signal <-> sfu通信
	*/
	SignalToSfuPublish        = ClientToSignalPublish     // signal->sfu 发布流
	SignalToSfuUnPublish      = ClientToSignalUnPublish   // signal->sfu 取消发布流
	SignalToSfuSubscribe      = ClientToSignalSubscribe   // signal->sfu 订阅流
	SignalToSfuUnSubscribe    = ClientToSignalUnSubscribe // signal->sfu 取消订阅
	SfuToSignalOnStreamRemove = "sfu_stream_remove"       // sfu->signal 通知流被移除

	/*
		signal -> register通信
	*/
	SignalToRegisterOnJoin         = "peer_join"     // signal->register 有人加入房间
	SignalToRegisterOnLeave        = "peer_leave"    // signal->register 有人离开房间
	SignalToRegisterKeepAlive      = "keepalive"     // signal->register 有人保活
	SignalToRegisterOnStreamAdd    = "stream_add"    // signal->register 有人开始推流
	SignalToRegisterOnStreamRemove = "stream_remove" // signal->register 有人停止推流
	SignalToRegisterGetSignalInfo  = "getSignalInfo" // signal->register 根据uid查询对应的user是否在线
	SignalToRegisterGetSfuInfo     = "getSfuInfo"    // signal->register 获取对应的sfu
	SignalToRegisterGetRoomUsers   = "getRoomUsers"  // signal->register 获取房间其他用户数据
	SignalToRegisterGetRoomPubs    = "getRoomPubs"   // signal->register 获取房间其他用户推流数据
)

// GetUIDFromMID 从mid中获取uid
func GetUIDFromMID(mid string) string {
	return strings.Split(mid, "#")[0]
}

// GetUserNodeKey 获取用户的signal服务器
func GetUserNodeKey(rid, uid string) string {
	return "/node/rid/" + rid + "/uid/" + uid
}

// GetMediaInfoKey  获取用户流信息
func GetMediaInfoKey(rid, uid, mid string) string {
	return "/media/rid" + rid + "/uid/" + uid + "/mid/" + mid
}

// GetMediaPubKey 获取用户流的sfu服务器
func GetMediaPubKey(rid, uid, mid string) string {
	return "/pub/rid" + rid + "/uid/" + uid + "/mid/" + mid
}
