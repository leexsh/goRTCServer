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

const maxRTCPChanSize = 100

// sub 订阅对象
type Sub struct {
	Id    string
	stop  bool
	alive bool
	pc    *webrtc.PeerConnection

	writeErrcnt int
	TrackAudio  *webrtc.RTPSender
	TrackVideo  *webrtc.RTPSender
	RtcpAudioCh chan rtcp.Packet
	RtcpVideoCh chan rtcp.Packet
}

func NewSub(sid string) (*Sub, error) {
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
		logger.Errorf("sub new peer err=%v, sid=%s", err, sid)
		return nil, err
	}

	sub := &Sub{
		Id:          sid,
		pc:          pcnew,
		stop:        false,
		alive:       true,
		writeErrcnt: 0,
		TrackAudio:  nil,
		TrackVideo:  nil,
		RtcpAudioCh: make(chan rtcp.Packet, maxRTCPChanSize),
		RtcpVideoCh: make(chan rtcp.Packet, maxRTCPChanSize),
	}
	pcnew.OnConnectionStateChange(sub.OnPeerConnect)
	return sub, nil
}

// OnPeerConnect Sub连接状态回调
func (s *Sub) OnPeerConnect(state webrtc.PeerConnectionState) {
	if state == webrtc.PeerConnectionStateConnected {
		logger.Debugf("sub peer connected = %s", s.Id)
		s.alive = true
		go s.DoVideoRtcp()
	}
	if state == webrtc.PeerConnectionStateDisconnected {
		logger.Debugf("sub peer disconnected = %s", s.Id)
		s.alive = false
	}
	if state == webrtc.PeerConnectionStateFailed {
		logger.Debugf("sub peer failed = %s", s.Id)
		s.alive = false
	}
}

// Close 关闭Sub
func (s *Sub) Close() {
	logger.Debugf("sub close = %s", s.Id)
	s.stop = true
	s.pc.Close()
	close(s.RtcpAudioCh)
	close(s.RtcpVideoCh)
}

// AddTrack 增加Track
func (s *Sub) AddTrack(remoteTrack *webrtc.Track) error {
	track, err := s.pc.NewTrack(remoteTrack.PayloadType(), remoteTrack.SSRC(), remoteTrack.ID(), remoteTrack.Label())
	if err != nil {
		logger.Errorf("sub new track err, err is %v, sid is %s", err, s.Id)
		return err
	}
	sender, err := s.pc.AddTrack(track)
	if err != nil {
		logger.Errorf("sub add track err, err is %v, sid is %s", err, s.Id)
		return err
	}
	if remoteTrack.Kind() == webrtc.RTPCodecTypeAudio {
		s.TrackAudio = sender
	}
	if remoteTrack.Kind() == webrtc.RTPCodecTypeVideo {
		s.TrackVideo = sender
	}
	return nil
}

// Answer 交换SDP
func (s *Sub) Answer(offer webrtc.SessionDescription) (webrtc.SessionDescription, error) {
	err := s.pc.SetRemoteDescription(offer)
	if err != nil {
		logger.Errorf("sub set answer err, err is %v, sid is %s", err, s.Id)
		return webrtc.SessionDescription{}, err
	}

	sdp, err := s.pc.CreateAnswer(nil)
	if err != nil {
		logger.Errorf("sub create sdp err, err is %v, sid is %s", err, s.Id)
		return webrtc.SessionDescription{}, err
	}
	err = s.pc.SetLocalDescription(sdp)
	if err != nil {
		logger.Errorf("sub set sdp err, err is %v, sid is %s", err, s.Id)
		return webrtc.SessionDescription{}, err
	}
	return sdp, nil
}

// DoAudioRtcp 接收音频RTCP包
func (s *Sub) DoAudioRtcp() {
	if s.TrackAudio != nil {
		for {
			if s.stop || !s.alive {
				return
			}

			rtcps, err := s.TrackAudio.ReadRTCP()
			if err != nil {
				if err == io.EOF {
					s.alive = false
				}
			} else {
				for _, rtcp := range rtcps {
					if s.stop || !s.alive {
						return
					}
					s.RtcpAudioCh <- rtcp
				}
			}
		}
	}
}

// DoVideoRtcp 接收视频RTCP包
func (s *Sub) DoVideoRtcp() {
	if s.TrackVideo != nil {
		for {
			if s.stop || !s.alive {
				return
			}

			rtcps, err := s.TrackVideo.ReadRTCP()
			if err != nil {
				if err == io.EOF {
					s.alive = false
				}
			} else {
				for _, rtcp := range rtcps {
					if s.stop || !s.alive {
						return
					}
					s.RtcpVideoCh <- rtcp
				}
			}
		}
	}
}

// ReadAudioRTCP 读音频RTCP包
func (s *Sub) ReadAudioRTCP() (rtcp.Packet, error) {
	pkt, ok := <-s.RtcpAudioCh
	if !ok {
		return nil, errors.New("audio rtcp chan close")
	}
	return pkt, nil
}

// ReadVideoRTCP 读视频RTCP包
func (s *Sub) ReadVideoRTCP() (rtcp.Packet, error) {
	pkt, ok := <-s.RtcpVideoCh
	if !ok {
		return nil, errors.New("video rtcp chan close")
	}
	return pkt, nil
}

// WriteAudioRTP 写音频包
func (s *Sub) WriteAudioRTP(pkt *rtp.Packet) error {
	if s.TrackAudio != nil && s.TrackAudio.Track() != nil && !s.stop && !s.alive {
		return s.TrackAudio.Track().WriteRTP(pkt)
	}
	return errors.New("sub audio track is nil or peer not connect")
}

// WriteVideoRTP 写视频包
func (s *Sub) WriteVideoRTP(pkt *rtp.Packet) error {
	if s.TrackVideo != nil && s.TrackVideo.Track() != nil && !s.stop && !s.alive {
		return s.TrackVideo.Track().WriteRTP(pkt)
	}
	return errors.New("sub video track is nil or peer not connect")
}

// return write error
func (s *Sub) WriteErrTotal() int {
	return s.writeErrcnt
}

// reset write error
func (s *Sub) WriteErrReset() {
	s.writeErrcnt = 0
}

// WriteErr add
func (s *Sub) WriteErrAdd() {
	s.writeErrcnt++
}
