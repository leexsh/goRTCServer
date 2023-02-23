package rtc

import (
	"errors"
	"fmt"
	"goRTCServer/pkg/logger"
	"goRTCServer/server/sfu/conf"
	"strings"
	"sync"
	"time"

	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v2"
	"github.com/pion/webrtc/v2/pkg/media/oggwriter"
)

const liveCycle = 6 * time.Second

// Router 对象
type Router struct {
	Id   string
	stop bool
	pub  *Pub
	subs map[string]*Sub
	sync.Mutex
	audioAlive time.Time
	videoAlive time.Time
	pktBuffer  map[uint16]*rtp.Packet
	oggWriter  *oggwriter.OggWriter
}

// NewRouter 创建新的Router对象
func NewRouter(id string) *Router {
	open := conf.Ogg.OPEN
	var writer *oggwriter.OggWriter
	if open {
		spilt := strings.Split(id, "/")
		writer, _ = oggwriter.New(fmt.Sprintf("%s.gg", spilt[7]), 48000, 2)
	}
	return &Router{
		Id:         id,
		stop:       false,
		pub:        nil,
		subs:       make(map[string]*Sub),
		Mutex:      sync.Mutex{},
		audioAlive: time.Now().Add(liveCycle),
		videoAlive: time.Now().Add(liveCycle),
		pktBuffer:  make(map[uint16]*rtp.Packet),
		oggWriter:  writer,
	}
}

// AddPub 增加Pub对象
func (r *Router) AddPub(mid, sdp string) (string, error) {
	pub, err := NewPub(mid)
	if err != nil {
		logger.Errorf("router add pub err, err is %v, id is %s, mid is %s", err, r.Id, mid)
		return "", err
	}
	offer := webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: sdp}
	answer, err := pub.Answer(offer)
	if err != nil {
		logger.Errorf("router pub answer err=%v, id=%s, mid=%s", err, r.Id, mid)
		pub.Close()
		return "", err
	}

	logger.Debugf("router add pub, pub is %s", r.Id)
	r.pub = pub
	go r.DoAudioWork()
	go r.DoVideoWork()
	return answer.SDP, nil
}

// AddSub
func (r *Router) AddSub(sid, sdp string) (string, error) {
	sub, err := NewSub(sid)
	if err != nil {
		logger.Errorf("router add sub err, err is %v, id is %s, sid is %s", err, r.Id, sid)
		return "", err
	}
	if r.pub != nil {
		if r.pub.TrackAudio != nil {
			if r.pub.TrackAudio.Track() != nil {
				err := sub.AddTrack(r.pub.TrackAudio.Track())
				if err != nil {
					logger.Errorf("router sub add audio track err, err is %v, id is %s, sid is %s", err, r.Id, sid)
					sub.Close()
					return "", err
				}
			}
		}
	}

	if r.pub != nil {
		if r.pub.TrackVideo != nil {
			if r.pub.TrackVideo.Track() != nil {
				err := sub.AddTrack(r.pub.TrackVideo.Track())
				if err != nil {
					logger.Errorf("router sub add Video track err, err is %v, id is %s, sid is %s", err, r.Id, sid)
					sub.Close()
					return "", err
				}
			}
		}
	}
	if sub.TrackVideo == nil && sub.TrackVideo == nil {
		sub.Close()
		return "", errors.New("router sub no audio and video track")
	}
	offer := webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: sdp}
	answer, err := sub.Answer(offer)
	if err != nil {
		logger.Errorf("router sub offer err, err is %v, id is %s, sid is %s", err, r.Id, sid)
		sub.Close()
	}
	logger.Debugf("router add sub, sub is %s", sub.Id)

	r.Lock()
	r.subs[sid] = sub
	r.Unlock()

	go r.DoRTCPWork(sub)
	return answer.SDP, nil
}

// GetPub 获取Pub对象
func (r *Router) GetPub() *Pub {
	return r.pub
}

// GetSub 获取Sub对象
func (r *Router) GetSub(sid string) *Sub {
	r.Lock()
	defer r.Unlock()
	return r.subs[sid]
}

// DelSub 删除Sub对象
func (r *Router) DelSub(sid string) {
	r.Lock()
	defer r.Unlock()
	sub := r.subs[sid]
	if sub != nil {
		sub.Close()
		delete(r.subs, sid)
	}
}

// GetSubCount 获取sub数量
func (r *Router) GetSubs() int {
	r.Lock()
	defer r.Unlock()
	res := 0
	for _, sub := range r.subs {
		res++
		logger.Debugf("router sub count++, sid is %s", sub.Id)
	}
	return res
}

// Alive 判断Router状态
func (r *Router) Alive() bool {
	if r.stop {
		return false
	}
	if r.pub != nil {
		if r.pub.stop || !r.pub.alive {
			return false
		}
		bAudio := !r.audioAlive.Before(time.Now())
		bVideo := !r.videoAlive.Before(time.Now())
		return (!bAudio || !bVideo)
	}
	return true
}

// Close 关闭Router
func (r *Router) Close() {
	r.stop = true
	if r.pub != nil {
		r.pub.Close()
		r.pub = nil
	}
	r.Lock()
	for sid, sub := range r.subs {
		sub.Close()
		delete(r.subs, sid)
	}
	r.Unlock()

	r.pktBuffer = make(map[uint16]*rtp.Packet)
	if r.oggWriter != nil {
		r.oggWriter.Close()
	}
}

// DoAudioWork 处理音频
func (r *Router) DoAudioWork() {
	for true {
		if r.stop || r.pub == nil || r.pub.stop || !r.pub.alive {
			return
		}

		if r.pub != nil && r.pub.TrackAudio != nil {
			pkt, err := r.pub.ReadAudioRTP()
			if err == nil {
				r.audioAlive = time.Now().Add(liveCycle)
				r.Lock()
				for sid, sub := range r.subs {
					if sub.stop || !sub.alive {
						sub.Close()
						delete(r.subs, sid)
					} else {
						sub.WriteAudioRTP(pkt)
					}
				}
				if r.oggWriter != nil {
					r.oggWriter.WriteRTP(pkt)
				}
				r.Unlock()
			}
		} else {
			time.Sleep(time.Second)
		}
	}
}

// DoVideoWork 处理视频
func (r *Router) DoVideoWork() {
	for {
		if r.stop || r.pub == nil || r.pub.stop || !r.pub.alive {
			return
		}
		if r.pub != nil && r.pub.TrackVideo != nil {
			pkt, err := r.pub.ReadVideoRTP()
			if err == nil {
				// 缓存包
				r.pktBuffer[pkt.SequenceNumber] = pkt
				// 转发包
				r.videoAlive = time.Now().Add(liveCycle)
				r.Lock()
				for sid, sub := range r.subs {
					if sub.stop || !sub.alive {
						sub.Close()
						delete(r.subs, sid)
					} else {
						sub.WriteVideoRTP(pkt)
					}
				}
				r.Unlock()
			}
		} else {
			time.Sleep(time.Second)
		}
	}
}

// DoRTCPWork 处理RTCP包， 目前只处理视频
func (r *Router) DoRTCPWork(sub *Sub) {
	for true {
		if r.stop || sub.TrackVideo == nil || sub.stop || !sub.alive {
			return
		}

		if sub.TrackVideo != nil {
			pkt, err := sub.ReadVideoRTCP()
			if err == nil {
				switch (pkt).(type) {
				case *rtcp.PictureLossIndication:
					if r.pub != nil {
						r.pub.WriteVideoRTCP(pkt)
					}
				case *rtcp.TransportLayerNack:
					nack := (pkt.(*rtcp.TransportLayerNack))
					for _, pair := range nack.Nacks {
						nackpkt := &rtcp.TransportLayerNack{
							SenderSSRC: nack.SenderSSRC,
							MediaSSRC:  nack.MediaSSRC,
							Nacks:      []rtcp.NackPair{{PacketID: pair.PacketID}},
						}
						if r.pub != nil {
							nackpktTmp := r.pktBuffer[pair.PacketID]
							if nackpktTmp != nil {
								sub.WriteVideoRTP(nackpktTmp)
							} else {
								r.pub.WriteVideoRTCP(nackpkt)
							}
						}
					}
				default:

				}

			}
		}
	}
}
