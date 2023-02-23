package src

import (
	"goRTCServer/pkg/etcd"
	"goRTCServer/pkg/logger"
	"goRTCServer/pkg/proto"
	"goRTCServer/pkg/utils"
	"goRTCServer/server/sfu/conf"
	"goRTCServer/server/sfu/rtc"
	"net/http"
	"strings"
	"time"

	nprotoo "github.com/cloudwebrtc/nats-protoo"
	"github.com/sirupsen/logrus"
)

const statCycle = 10 * time.Second

var (
	sfuNode *etcd.ServiceNode
	sfuNats *nprotoo.NatsProtoo
	caster  *nprotoo.Broadcaster
)

func init() {
	logger.DoInit(conf.Kafka.URL, "dev_rtc_sfu")
	logger.SetLevel(logrus.DebugLevel)
}

// start 启动服务
func Start() {
	// 服务注册
	sfuNode = etcd.NewServiceNode(conf.Etcd.Adds, conf.Global.NodeDC, conf.Global.NodeID, conf.Global.Name)
	sfuNode.RegisterNode()
	// 消息注册
	sfuNats = nprotoo.NewNatsProtoo(conf.Nats.URL)
	sfuNats.OnRequest(sfuNode.GetRPCChannel(), handleRPCRequest)
	// 消息广播
	caster = sfuNats.NewBroadcaster(sfuNode.GetEventChannel())
	// 启动RTC
	rtc.InitRTC()
	// 启动调试
	if conf.Global.Pprof != "" {
		go debug()
	}
	// 启动其他
	go CheckRTC()
	go UpdatePaylaod()
}

// Stop 关闭连接
func Stop() {
	rtc.FreeRTC()
	if sfuNats != nil {
		sfuNats.Close()
	}
	if sfuNode != nil {
		sfuNats.Close()
	}
}

// CheckRTC 通知信令 流被移除
func CheckRTC() {
	for i := range rtc.CleanRouter {
		str := strings.Split(i, "/")
		rid := str[3]
		uid := str[5]
		mid := str[7]
		caster.Say(proto.SfuToSignalOnStreamRemove, utils.Map("rid", rid, "uid", uid, "mid", mid))
	}
}

// 更新sfu服务器的负债
func UpdatePaylaod() {
	t := time.NewTicker(statCycle)
	defer t.Stop()
	for range t.C {
		sfuNode.UpdateNodePayload(rtc.GetRouters())
	}
}

func debug() {
	logger.Debugf("start sfu pprof on %s", conf.Global.Pprof)
	http.ListenAndServe(conf.Global.Pprof, nil)
}
