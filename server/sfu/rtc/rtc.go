package rtc

import (
	"goRTCServer/pkg/logger"
	"goRTCServer/server/sfu/conf"
	"sync"
	"time"

	"github.com/pion/webrtc/v2"
)

const statCycle = 5 * time.Second
const maxCleanSize = 100

var (
	stop         bool
	icePortStart uint16
	icePortEnd   uint16
	iceServers   []webrtc.ICEServer
	routers      map[string]*Router
	routerLock   sync.Mutex
	CleanRouter  chan string
)

// 初始化RTC
func InitRTC() {
	stop = false
	if len(conf.WebRTC.ICEPortRange) == 2 {
		icePortStart = conf.WebRTC.ICEPortRange[0]
		icePortEnd = conf.WebRTC.ICEPortRange[1]
	}

	iceServers = make([]webrtc.ICEServer, 0)
	for _, iceServer := range conf.WebRTC.ICEServers {
		server := webrtc.ICEServer{
			URLs:       iceServer.URLS,
			Username:   iceServer.Username,
			Credential: iceServer.Credentiail,
		}
		iceServers = append(iceServers, server)
	}

	routers = make(map[string]*Router)
	CleanRouter = make(chan string, maxCleanSize)

	go CheckRouter()

}

// 销毁RTC
func FreeRTC() {
	stop = true
	routerLock.Lock()
	defer routerLock.Unlock()
	for id, router := range routers {
		if router != nil {
			router.Close()
			delete(routers, id)
		}
	}
}

// GetNewRoutuer 获取router
func GetNewRouter(id string) *Router {
	r := GetRouter(id)
	if r == nil {
		return AddRouter(id)
	}
	return r
}

// GetRouters 获取Routers所有负载
func GetRouters() int {
	routerLock.Lock()
	defer routerLock.Unlock()
	res := 0
	for _, router := range routers {
		pub := router.GetPub()
		if pub != nil {
			res++
		}
		res += router.GetSubs()
	}
	return res
}

// 获取Router
func GetRouter(id string) *Router {
	routerLock.Lock()
	defer routerLock.Unlock()
	return routers[id]
}

// AddRouter 增加Router
func AddRouter(id string) *Router {
	logger.Debugf("add router, router is %s", id)
	r := NewRouter(id)
	routerLock.Lock()
	defer routerLock.Unlock()
	routers[id] = r
	return r
}

// DelRouter 删除Router
func DelRouter(id string) {
	r := GetRouter(id)
	if r != nil {
		logger.Debugf("Del router, r is %s", r.Id)
		r.Close()
		routerLock.Lock()
		defer routerLock.Unlock()
		delete(routers, id)
	}
}

// CheckRouter 查询所有的Router状态
func CheckRouter() {
	t := time.NewTicker(statCycle)
	defer t.Stop()
	for true {
		if stop {
			return
		}
		<-t.C
		routerLock.Lock()
		for id, router := range routers {
			if !router.Alive() {
				logger.Debugf("router is dead, id is %s", id)
				router.Close()
				delete(routers, id)
				CleanRouter <- id
			}
		}
		routerLock.Unlock()
	}
}
