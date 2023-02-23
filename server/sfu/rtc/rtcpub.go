package rtc

import (
	"errors"
	"goRTCServer/pkg/logger"
	"io"

	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v2"
)

const maxRTCChanSize = 100

type Pub struct {
	Id    string
	stop  bool
	alive bool
	pc    *webrtc.PeerConnection

	TrackAudio *webrtc.RTPReceiver
	TrackVideo *webrtc.RTPReceiver
	RtpAudioCh chan *rtp.Packet
	RtpVideoCh chan *rtp.Packet
}

func NewPub(pid string) (*Pub, error) {
	cfg := webrtc.Configuration{
		ICEServers:         iceServers,
		ICETransportPolicy: webrtc.ICETransportPolicyAll,
		SDPSemantics:       webrtc.SDPSemanticsUnifiedPlanWithFallback,
	}
	engine := webrtc.MediaEngine{}
	opus := webrtc.NewRTPCodec(webrtc.RTPCodecTypeAudio, webrtc.Opus, 48000, 2, "minptime=10;useinbandfec=1;stereo=1",
		webrtc.DefaultPayloadTypeOpus, &codecs.OpusPayloader{})
	engine.RegisterCodec(opus)
	engine.RegisterCodec(webrtc.NewRTPVP8Codec(webrtc.DefaultPayloadTypeVP8, 90000))

	setting := webrtc.SettingEngine{}
	if icePortStart != 0 && icePortEnd != 0 {
		setting.SetEphemeralUDPPortRange(icePortStart, icePortEnd)
	}

	api := webrtc.NewAPI(webrtc.WithMediaEngine(engine), webrtc.WithSettingEngine(setting))
	pcnew, err := api.NewPeerConnection(cfg)
	if err != nil {
		logger.Errorf("pub new peer err, err is %v, pubid is %s", err, pid)
	}
	_, err = pcnew.AddTransceiverFromKind(webrtc.RTPCodecTypeAudio, webrtc.RtpTransceiverInit{
		Direction: webrtc.RTPTransceiverDirectionRecvonly,
	})
	if err != nil {
		logger.Errorf("pub add audio recv err=%v, pubid=%s", err, pid)
		pcnew.Close()
		return nil, err
	}

	_, err = pcnew.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo, webrtc.RtpTransceiverInit{
		Direction: webrtc.RTPTransceiverDirectionRecvonly,
	})
	if err != nil {
		logger.Errorf("pub add video recv err=%v, pubid=%s", err, pid)
		pcnew.Close()
		return nil, err
	}

	pub := &Pub{
		Id:         pid,
		pc:         pcnew,
		stop:       false,
		alive:      true,
		TrackAudio: nil,
		TrackVideo: nil,
		RtpAudioCh: make(chan *rtp.Packet, maxRTCChanSize),
		RtpVideoCh: make(chan *rtp.Packet, maxRTCChanSize),
	}
	pcnew.OnConnectionStateChange(pub.OnPeerConnect)
	pcnew.OnTrack(pub.OnTrackRemote)
	return pub, nil
}

// OnPeerConnect Pub连接状态的回到
func (p *Pub) OnPeerConnect(state webrtc.PeerConnectionState) {
	if state == webrtc.PeerConnectionStateConnected {
		logger.Debugf("pub peer connected, pid is %s", p.Id)
		p.alive = true
	} else if state == webrtc.PeerConnectionStateDisconnected {
		logger.Debugf("pub peer disconnected, pid is %s", p.Id)
		p.alive = false
	} else if state == webrtc.PeerConnectionStateFailed {
		logger.Debugf("pub peer failed, pid is %s", p.Id)
		p.alive = false
	}
}

// OnTrackRemote 接受到track的回调
func (p *Pub) OnTrackRemote(track *webrtc.Track, receiver *webrtc.RTPReceiver) {
	if track.Kind() == webrtc.RTPCodecTypeAudio {
		p.TrackAudio = receiver
		logger.Debugf("OnTrackRemote pub audio. pid is %s", p.Id)
		go p.DoAudioRTP()
	}

	if track.Kind() == webrtc.RTPCodecTypeVideo {
		p.TrackVideo = receiver
		logger.Debugf("OnTrackRemote pub video. pid is %s", p.Id)
		go p.DoVideoRTP()
	}
}

// Close 关闭连接
func (p *Pub) Close() {
	logger.Debugf("pub close, pid is %s", p.Id)
	p.stop = true
	p.pc.Close()
	close(p.RtpAudioCh)
	close(p.RtpVideoCh)
}

// Answer SDP交换
func (p *Pub) Answer(offer webrtc.SessionDescription) (webrtc.SessionDescription, error) {
	err := p.pc.SetRemoteDescription(offer)
	if err != nil {
		logger.Errorf("pub set offer err, err is %v, pid is %s", err, p.Id)
		return webrtc.SessionDescription{}, err
	}

	answer, err := p.pc.CreateAnswer(nil)
	if err != nil {
		logger.Errorf("pub create answer err, err is %v, pid is %s", err, p.Id)
		return webrtc.SessionDescription{}, err
	}
	err = p.pc.SetLocalDescription(answer)
	if err != nil {
		logger.Errorf("pub set answer err, err is %v, pid is %s", err, p.Id)
		return webrtc.SessionDescription{}, err
	}
	return answer, err
}

// DoAudioRTP 处理音频包
func (p *Pub) DoAudioRTP() {
	if p.TrackAudio != nil && p.TrackAudio.Track() != nil {
		for true {
			if p.stop || !p.alive {
				return
			}
			rtp, err := p.TrackAudio.Track().ReadRTP()
			if err != nil {
				if err == io.EOF {
					p.alive = false
					logger.Errorf("pub.TrackAudio Read RTP error, err is io.EOF")
				}
			} else {
				if p.stop || p.alive == false {
					return
				}
				p.RtpAudioCh <- rtp
			}
		}
	}
}

// DoVideoRTP 处理视频包
func (p *Pub) DoVideoRTP() {
	if p.TrackVideo != nil && p.TrackVideo.Track() != nil {
		for true {
			if p.stop || !p.alive {
				return
			}
			rtp, err := p.TrackVideo.Track().ReadRTP()
			if err != nil {
				if err == io.EOF {
					p.alive = false
					logger.Errorf("pub.TrackVideo Read RTP error, err is io.EOF")
				}
			} else {
				if p.stop || p.alive == false {
					return
				}
				p.RtpVideoCh <- rtp
			}
		}
	}
}

// ReadAudioRTP 读取音频RTP包
func (p *Pub) ReadAudioRTP() (*rtp.Packet, error) {
	rtp, ok := <-p.RtpAudioCh
	if !ok {
		return nil, errors.New("pub audio rtp chan close")
	}
	return rtp, nil
}

// ReadVideoRTP 读取视频RTP包
func (p *Pub) ReadVideoRTP() (*rtp.Packet, error) {
	rtp, ok := <-p.RtpVideoCh
	if !ok {
		return nil, errors.New("pub video rtp chan close")
	}
	return rtp, nil
}

// WriteVideoRtcp 发送RTCP包
func (p *Pub) WriteVideoRTCP(pkg rtcp.Packet) error {
	if p.pc == nil {
		return errors.New("pub pc is nil")
	}
	return p.pc.WriteRTCP([]rtcp.Packet{pkg})
}
