package src

import (
	"goRTCServer/pkg/etcd"
	"goRTCServer/pkg/logger"
	myRedis "goRTCServer/pkg/redis"
	"goRTCServer/server/register/conf"
	"net/http"
	"time"

	nprotoo "github.com/cloudwebrtc/nats-protoo"
	"github.com/sirupsen/logrus"
)

const (
	redisShort  = 60 * time.Second
	redisKeyTTL = 24 * time.Hour
)

var (
	regRedis *myRedis.Redis
	regNde   *etcd.ServiceNode
	regNats  *nprotoo.NatsProtoo
)

func init() {
	logger.DoInit(conf.Kafka.URL, "rtc_register")
	logger.SetLevel(logrus.DebugLevel)
}

// 启动服务
func Start() {
	// 服务注册
	node := etcd.NewServiceNode(conf.Etcd.Addrs, conf.Global.NodeDC, conf.Global.NodeID, conf.Global.Name)
	node.RegisterNode()

	// 消息注册
	regNats = nprotoo.NewNatsProtoo(conf.Nats.URL)
	regNats.OnRequest(node.GetRPCChannel(), handleRPCMsg)

	// 数据库
	regRedis = myRedis.NewRedis(myRedis.Config(*conf.Redis))
	// 启动调试
	if conf.Global.Pprof != "" {
		go debug()
	}
}

func Stop() {
	if regNats != nil {
		regNats.Close()
	}
	if regNde != nil {
		regNde.Close()
	}
}

func debug() {
	logger.LogKf.Debugf("start register on %s", conf.Global.Pprof)
	http.ListenAndServe(conf.Global.Pprof, nil)
}
