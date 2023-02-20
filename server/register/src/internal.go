package src

import (
	"fmt"
	"goRTCServer/pkg/logger"
	"goRTCServer/pkg/proto"
	"goRTCServer/pkg/utils"
	"strings"

	nprotoo "github.com/cloudwebrtc/nats-protoo"
)

// 处理RPC请求
func handleRPCMsg(request nprotoo.Request, accept nprotoo.RespondFunc, reject nprotoo.RejectFunc) {
	go handleRPCRequest(request, accept, reject)
}

// 接收signal消息处理
func handleRPCRequest(req nprotoo.Request, accept nprotoo.RespondFunc, reject nprotoo.RejectFunc) {
	defer utils.Recover("register.handleRPCRequest")
	method := req.Method
	data := utils.Unmarshal(string(req.Data))

	var res map[string]interface{}
	err := &nprotoo.Error{Code: 400, Reason: fmt.Sprintf("Unknown method [%s]", method)}

	// 根据method处理
	switch method {
	case proto.SignalToRegisterOnJoin:
		res, err = clientJoin(data)
	case proto.SignalToRegisterOnLeave:
		res, err = clientLeave(data)
	case proto.SignalToRegisterKeepAlive:
		res, err = keepalive(data)
	case proto.SignalToRegisterOnStreamAdd:
		res, err = streamAdd(data)
	case proto.SignalToRegisterOnStreamRemove:
		res, err = streamRemove(data)
	case proto.SignalToRegisterGetSignalInfo:
		res, err = getUserOnlineByUid(data)
	case proto.SignalToRegisterGetSfuInfo:
		res, err = getSfuByMid(data)
	case proto.SignalToRegisterGetRoomUsers:
		res, err = getRoomUsers(data)
	case proto.SignalToRegisterGetRoomPubs:
		res, err = getRoomPubs(data)
	}
	// 判断成功
	if err != nil {
		reject(err.Code, err.Reason)
	} else {
		accept(res)
	}
}

/*
	"method", proto.SignalToRegisterOnJoin "rid", rid, "uid" uid "signalId" signalId
*/
// 有人加入房间
func clientJoin(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	logger.LogKf.Debugf("register.join, data is %v", data)
	rid := utils.Val(data, "rid")
	uid := utils.Val(data, "uid")
	// 获取用户的signal服务器
	signalId := utils.Val(data, "signalId")

	uKey := proto.GetUserNodeKey(rid, uid)
	err := regRedis.Set(uKey, signalId, redisShort)
	if err != nil {
		logger.LogKf.Error("signal.clientJoin redis.set err, err is %v, data is %v", err, data)
		return nil, &nprotoo.Error{
			Code:   401,
			Reason: fmt.Sprintf("client join err is %v", err),
		}
	}
	return utils.Map("rid", rid, "uid", uid, "signalID", signalId), nil
}

/*
	"method", proto.SignalToRegisterOnLeave "rid" rid "uid" uid
*/
// 有人退出房间
func clientLeave(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	logger.LogKf.Debugf("register.leave, data is %v", data)
	rid := utils.Val(data, "rid")
	uid := utils.Val(data, "uid")
	// 获取用户的Signal服务器
	uKey := proto.GetUserNodeKey(rid, uid)
	uVaule := regRedis.Get(uKey)
	if len(uVaule) > 0 {
		// del this user cache
		err := regRedis.Del(uKey)
		if err != nil {
			logger.Debugf("register.clientLeave redis.Del err, err is %v, data is %v", err, data)
		}
	}
	return utils.Map("rid", rid, "uid", uid), nil
}

/*
	"method", proto.SignalToRegisterKeepAlive, "rid" rid "uid" uid
*/
// 保活处理
func keepalive(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	logger.LogKf.Debugf("register.keepalive, data is %v", data)
	rid := utils.Val(data, "rid")
	uid := utils.Val(data, "uid")
	// 获取用户的Signal服务器
	uKey := proto.GetUserNodeKey(rid, uid)

	err := regRedis.Expire(uKey, redisShort)
	if err != nil {
		logger.Errorf("register.keepalive redis.Expire err, err is %v, data ois %v", err, data)
		return nil, &nprotoo.Error{
			Code:   402,
			Reason: fmt.Sprintf("keep alive err is %v", err),
		}
	}
	return utils.Map("rid", rid, "uid", uid), nil
}

/*
	"method", proto.SignalToRegisterOnStreamAdd
*/
// 有人发布流
func streamAdd(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	logger.Debugf("register.streamAdd, data is %v", data)
	rid := utils.Val(data, "rid")
	uid := utils.Val(data, "uid")
	mid := utils.Val(data, "mid")
	sfuId := utils.Val(data, "sfuid")
	minfo := utils.Val(data, "minfo")
	// 获取用户的流信息
	uKey := proto.GetMediaInfoKey(rid, uid, mid)
	err := regRedis.Set(uKey, minfo, redisKeyTTL)
	if err != nil {
		logger.Errorf("register.steamAdd media redis.Set err, err is %v, data is %v", err, data)
		return nil, &nprotoo.Error{
			Code:   405,
			Reason: fmt.Sprintf("streamAdd err, err is %v", err),
		}
	}
	// 获取用户的sfu服务器
	uKey = proto.GetMediaPubKey(rid, uid, mid)
	err = regRedis.Set(uKey, sfuId, redisKeyTTL)
	if err != nil {
		logger.Errorf("register.streamAdd pub redis.Set err, err is %v, data is %v", err, data)
		return nil, &nprotoo.Error{
			Code:   406,
			Reason: fmt.Sprintf("streamAdd err, err is %v", err),
		}
	}
	// 生成resp对象
	return utils.Map("rid", rid, "uid", uid, "mid", mid, "sfuid", sfuId, "minfo", data["minfo"]), nil
}

/*
	"method", proto.SignalToRegisterOnStreamRemove
*/
// 有人取消发布流
func streamRemove(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	logger.Debugf("register.streamRemove, data is %v", data)
	rid := utils.Val(data, "rid")
	uid := utils.Val(data, "uid")
	mid := utils.Val(data, "mid")
	// 判断mid是否为空
	rmPubs := make([]map[string]interface{}, 0)
	var ukey string
	if mid == "" {
		ukey = "/media/rid/" + rid + "/uid/" + uid + "/mid/*"
		ukeys := regRedis.Keys(ukey)
		for _, key := range ukeys {
			ukey = key
			// 删除key值
			err := regRedis.Del(ukey)
			if err != nil {
				logger.Errorf("register.streamRemove media redis.Del err, err is %v, data is %v", err, data)
			}
		}
		ukey = "/pub/rid/" + rid + "/uid/" + uid + "/mid/*"
		ukeys = regRedis.Keys(ukey)
		for _, key := range ukeys {
			ukey = key
			// 删除key值
			arr := strings.Split(key, "/")
			mid := arr[7]
			err := regRedis.Del(ukey)
			if err != nil {
				logger.Errorf("register.streamRemove media redis.Del err, err is %v, data is %v", err, data)
			}
			rmPubs = append(rmPubs, utils.Map("rid", rid, "uid", uid, "mid", mid))
		}
	} else {
		// 获取用户流信息
		ukey = proto.GetMediaPubKey(rid, uid, mid)
		uKeys := regRedis.Keys(ukey)
		for _, key := range uKeys {
			ukey = key
			// 删除key值
			err := regRedis.Del(ukey)
			if err != nil {
				logger.Errorf("register.streamRemove pub redis.Del err, err is %v, data is %v", err, data)
			}
		}
		// 获取用户流的sfu服务器
		ukey = proto.GetMediaPubKey(rid, uid, mid)
		uKeys = regRedis.Keys(ukey)
		for _, key := range uKeys {
			ukey = key
			arr := strings.Split(key, "/")
			mid := arr[7]
			// 删除key值
			err := regRedis.Del(ukey)
			if err != nil {
				logger.Errorf("register.streamRemove pub redis.Del err, err is %v, data is %v", err, data)
			}
			rmPubs = append(rmPubs, utils.Map("rid", rid, "uid", uid, "mid", mid))
		}
	}
	return utils.Map("rmPubs", rmPubs), nil
}

/*
	"method" proto.SignalToRegisterGetUserInfo
*/
// 获取rid, uid指定的用户是否在线
func getUserOnlineByUid(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rid := utils.Val(data, "rid")
	uid := utils.Val(data, "uid")
	// 获取用户的signal服务器
	ukey := proto.GetUserNodeKey(rid, uid)
	uKeys := regRedis.Keys(ukey)
	if len(uKeys) > 0 {
		signalId := regRedis.Get(ukey)
		return utils.Map("rid", rid, "uid", uid, "signalid", signalId), nil
	} else {
		return nil, &nprotoo.Error{Code: 410, Reason: fmt.Sprintf("cann't find signal node by key: %v", ukey)}
	}
}

/*
	"method" proto.SignalToRegisterGetSfuInfo
*/
// 获取mid指定对应的sfu节点
func getSfuByMid(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rid := utils.Val(data, "rid")
	mid := utils.Val(data, "mid")
	uid := proto.GetUIDFromMID(mid)
	// 获取用户流的sfu服务器
	ukey := proto.GetMediaPubKey(rid, uid, mid)
	uKeys := regRedis.Keys(ukey)
	if len(uKeys) > 0 {
		sfuId := regRedis.Get(ukey)
		return utils.Map("rid", rid, "sfuid", sfuId), nil
	} else {
		return nil, &nprotoo.Error{Code: 410, Reason: fmt.Sprintf("cann't find sfu node by key: %v", ukey)}
	}
}

/*
	“method” proto.SignalToRegisterGetRoomUsers
*/
// 获取房间内其他用户的数据
func getRoomUsers(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	logger.Debugf("register.getRoomUsers, data is %v", data)
	rid := utils.Val(data, "rid")
	uid := utils.Val(data, "uid")
	// 查询数据库
	users := make([]map[string]interface{}, 0)
	ukey := "/node/rid/" + rid + "/uid/*"
	ukeys := regRedis.Keys(ukey)
	for _, ke := range ukeys {
		// 去掉指定的uid
		arr := strings.Split(ke, "/")
		id := arr[5]
		if id == uid {
			continue
		}
		signalId := regRedis.Get(ke)
		usuer := utils.Map("rid", rid, "uid", id, "signalid", signalId)
		users = append(users, usuer)
	}
	// return
	resp := utils.Map("users", users)
	return resp, nil
}

/*
	"method", proto.SignalToRegisterGetRoomPubs
*/
// 获取房间内其他用户的推流数据
func getRoomPubs(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	logger.Debugf("register.getRoomPubs, data is %v", data)
	rid := utils.Val(data, "rid")
	uid := utils.Val(data, "uid")
	// 查询数据库
	pubs := make([]map[string]interface{}, 0)
	uKey := "/pub/rid/" + rid + "/uid/*"
	uKeys := regRedis.Keys(uKey)
	for _, key := range uKeys {
		// 去掉指定的uid
		arr := strings.Split(key, "/")
		id := arr[5]
		mid := arr[7]
		if id == uid {
			continue
		}
		sfuid := regRedis.Get(key)
		mKey := proto.GetMediaInfoKey(rid, uid, mid)
		minfo := regRedis.Get(mKey)

		pub := utils.Map("rid", rid, "uid", uid, "mid", mid, "sfuid", sfuid, "minfo", utils.Unmarshal(minfo))
		pubs = append(pubs, pub)
	}

	resp := utils.Map("pubs", pubs)
	return resp, nil
}
