package src

import (
	"fmt"
	"goRTCServer/pkg/proto"
	"goRTCServer/pkg/utils"
	"goRTCServer/server/sfu/rtc"

	nprotoo "github.com/cloudwebrtc/nats-protoo"
)

// 处理sfu RPC请求
func handleRPCMsg(request nprotoo.Request, accept nprotoo.RespondFunc, reject nprotoo.RejectFunc) {
	go handleRPCRequest(request, accept, reject)
}

func handleRPCRequest(request nprotoo.Request, accept nprotoo.RespondFunc, reject nprotoo.RejectFunc) {
	defer utils.Recover("sfu.handleRPCRequest")
	method := request.Method
	data := utils.Unmarshal(string(request.Data))

	var res map[string]interface{}
	err := &nprotoo.Error{Code: 400, Reason: fmt.Sprintf("Unknown method [%s]", method)}

	if method != "" {
		switch method {
		case proto.SignalToSfuPublish:
			res, err = Publish(data)
		case proto.SignalToSfuUnPublish:
			res, err = UnPublish(data)
		case proto.SignalToSfuSubscribe:
			res, err = SubScribe(data)
		case proto.SignalToSfuUnSubscribe:
			res, err = UnSubscribe(data)
		}
	}
	if err != nil {
		reject(err.Code, err.Reason)
		return
	}
	accept(res)
}

/*
	"method", proto.BizToSfuPublish, "rid", rid, "uid", uid, "jsep", jsep
*/
// publish 处理推流
func Publish(msg map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	// 1. 获取参数
	if msg["jsep"] == nil {
		return nil, &nprotoo.Error{Code: 401, Reason: "cann't find jsep"}
	}
	jsep, ok := msg["jsep"].(map[string]interface{})
	if !ok {
		return nil, &nprotoo.Error{402, "jsep cannot transform to map"}
	}
	sdp := utils.Val(jsep, "sdp")
	rid := utils.Val(msg, "rid")
	uid := utils.Val(msg, "uid")
	mid := fmt.Sprintf("%s#%s", uid, utils.RandStr(6))

	// 2.获取Router
	key := proto.GetMediaPubKey(rid, uid, mid)
	router := rtc.GetNewRouter(key)
	if router == nil {
		return nil, &nprotoo.Error{403, fmt.Sprintf("cannot get router: %s", key)}
	}
	// 3.增加推流
	resp, err := router.AddPub(mid, sdp)
	if err != nil {
		return nil, &nprotoo.Error{403, fmt.Sprintf("add pub err, err is :%v", err)}
	}
	return utils.Map("mid", mid, "jsep", utils.Map("type", "answer", "sdp", resp)), nil
}

/*
	"method", proto.BizToSfuUnPublish, "rid", rid, "mid", mid
*/
// unpublish 处理取消发布流
func UnPublish(msg map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	// 1.获取参数
	rid := utils.Val(msg, "rid")
	mid := utils.Val(msg, "mid")
	uid := proto.GetUIDFromMID(mid)

	// 删除Router
	key := proto.GetMediaPubKey(rid, uid, mid)
	rtc.DelRouter(key)
	return utils.Map(), nil
}

/*
	"method", proto.BizToSfuSubscribe, "rid", rid, "suid", suid, "mid", mid, "jsep", jsep
*/
// subscribe 处理订阅流
func SubScribe(msg map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	// 1. 获取参数
	if msg["jsep"] == nil {
		return nil, &nprotoo.Error{Code: 401, Reason: "cann't find jsep"}
	}
	jsep, ok := msg["jsep"].(map[string]interface{})
	if !ok {
		return nil, &nprotoo.Error{402, "jsep cannot transform to map"}
	}
	sdp := utils.Val(jsep, "sdp")
	rid := utils.Val(msg, "rid")
	mid := utils.Val(msg, "mid")
	uid := proto.GetUIDFromMID(mid)

	suid := utils.Val(msg, "suid")
	sid := fmt.Sprintf("%s#%s", suid, utils.RandStr(6))

	// 2.获取Router
	key := proto.GetMediaPubKey(rid, uid, mid)
	router := rtc.GetRouter(key)
	if router == nil {
		return nil, &nprotoo.Error{403, fmt.Sprintf("cannot get router: %s", key)}
	}
	// 3.增加拉流
	resp, err := router.AddSub(sid, sdp)
	if err != nil {
		return nil, &nprotoo.Error{403, fmt.Sprintf("add sub error: %v", err)}
	}
	return utils.Map("sid", sid, "jsep", utils.Map("type", "answer", "sdp", resp)), nil
}

/*
	"method", proto.BizToSfuUnSubscribe, "rid", rid, "mid", mid, "sid", sid
*/
// unsubscribe 处理取消订阅流
func UnSubscribe(msg map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	// 获取参数
	rid := utils.Val(msg, "rid")
	mid := utils.Val(msg, "mid")
	sid := utils.Val(msg, "sid")
	uid := proto.GetUIDFromMID(mid)

	// 获取router
	key := proto.GetMediaPubKey(rid, uid, mid)
	router := rtc.GetRouter(key)
	if router == nil {
		return nil, &nprotoo.Error{Code: 410, Reason: fmt.Sprintf("can't get router:%s", key)}
	}

	// 删除拉流
	router.DelSub(sid)
	return utils.Map(), nil
}
