package srvlib

import (
	"os"
	"strconv"
	"strings"
	"time"

	"code.google.com/p/goprotobuf/proto"
	"github.com/idealeak/goserver/core/logger"
	"github.com/idealeak/goserver/core/netlib"
	"github.com/idealeak/goserver/srvlib/protocol"
)

var (
	SessionAttributeServiceInfo = &serviceMgr{}
	SessionAttributeServiceFlag = &serviceMgr{}
	ServiceMgr                  = &serviceMgr{servicesPool: make(map[int32]map[int32]*protocol.ServiceInfo)}
)

type serviceMgr struct {
	servicesPool map[int32]map[int32]*protocol.ServiceInfo
}

func (this *serviceMgr) RegisteService(s *netlib.Session, services []*protocol.ServiceInfo) {
	if this == nil || services == nil || len(services) == 0 {
		return
	}

	s.SetAttribute(SessionAttributeServiceInfo, services)
	for _, service := range services {
		srvid := service.GetSrvId()
		srvtype := service.GetSrvType()
		if _, has := this.servicesPool[srvtype]; !has {
			this.servicesPool[srvtype] = make(map[int32]*protocol.ServiceInfo)
		}
		this.servicesPool[srvtype][srvid] = service

		pack := &protocol.SSServiceInfo{}
		pack.Service = service
		proto.SetDefaults(pack)
		sessiontypes := GetCareSessionsByService(service.GetSrvType())
		areaId := service.GetAreaId()
		for _, v1 := range sessiontypes {
			ServerSessionMgrSington.Broadcast(pack, int(areaId), int(v1))
		}
	}
}

func (this *serviceMgr) UnregisteService(service *protocol.ServiceInfo) {
	if this == nil || service == nil {
		return
	}

	srvid := service.GetSrvId()
	srvtype := service.GetSrvType()
	if v, has := this.servicesPool[srvtype]; has {
		delete(v, srvid)
	}

	pack := &protocol.SSServiceShut{}
	pack.Service = service
	proto.SetDefaults(pack)
	sessiontypes := GetCareSessionsByService(service.GetSrvType())
	areaId := service.GetAreaId()
	for _, v1 := range sessiontypes {
		ServerSessionMgrSington.Broadcast(pack, int(areaId), int(v1))
	}
}

func (this *serviceMgr) OnRegiste(s *netlib.Session) {
	if this == nil || s == nil {
		return
	}

	if s.GetAttribute(SessionAttributeServiceFlag) == nil {
		return
	}
	attr := s.GetAttribute(SessionAttributeServerInfo)
	if attr != nil {
		if srvInfo, ok := attr.(*protocol.SSSrvRegiste); ok && srvInfo != nil {
			services := GetCareServicesBySession(srvInfo.GetType())
			for _, v1 := range services {
				if v2, has := this.servicesPool[v1]; has {
					for _, v3 := range v2 {
						func(si *protocol.ServiceInfo) {
							pack := &protocol.SSServiceInfo{}
							proto.SetDefaults(pack)
							pack.Service = si
							logger.Trace("Server Type=", srvInfo.GetType(), " Id=", srvInfo.GetId(), " Name=", srvInfo.GetName(), " Service=", si)
							s.Send(pack)
						}(v3)
					}
				}
			}
		}
	}
}

func (this *serviceMgr) OnUnregiste(s *netlib.Session) {
}

func (this *serviceMgr) ClearServiceBySession(s *netlib.Session) {
	attr := s.GetAttribute(SessionAttributeServiceInfo)
	if attr != nil {
		if services, ok := attr.([]*protocol.ServiceInfo); ok {
			for _, service := range services {
				this.UnregisteService(service)
			}
		}
		s.RemoveAttribute(SessionAttributeServiceInfo)
	}
}

func (this *serviceMgr) ReportService(s *netlib.Session) {
	acceptors := netlib.GetAcceptors()
	cnt := len(acceptors)
	if cnt > 0 {
		pack := &protocol.SSServiceRegiste{
			Services: make([]*protocol.ServiceInfo, 0, cnt),
		}
		proto.SetDefaults(pack)
		for _, v := range acceptors {
			addr := v.Addr()
			if addr == nil {
				continue
			}
			network := addr.Network()
			s := addr.String()
			ipAndPort := strings.Split(s, ":")
			if len(ipAndPort) < 2 {
				continue
			}

			port, err := strconv.Atoi(ipAndPort[len(ipAndPort)-1])
			if err != nil {
				continue
			}

			sc := v.GetSessionConfig()
			si := &protocol.ServiceInfo{
				AreaId:          proto.Int32(int32(sc.AreaId)),
				SrvId:           proto.Int32(int32(sc.Id)),
				SrvType:         proto.Int32(int32(sc.Type)),
				SrvPID:          proto.Int32(int32(os.Getpid())),
				SrvName:         proto.String(sc.Name),
				NetworkType:     proto.String(network),
				Ip:              proto.String(sc.Ip),
				Port:            proto.Int32(int32(port)),
				WriteTimeOut:    proto.Int32(int32(sc.WriteTimeout / time.Second)),
				ReadTimeOut:     proto.Int32(int32(sc.ReadTimeout / time.Second)),
				IdleTimeOut:     proto.Int32(int32(sc.IdleTimeout / time.Second)),
				MaxDone:         proto.Int32(int32(sc.MaxDone)),
				MaxPend:         proto.Int32(int32(sc.MaxPend)),
				MaxPacket:       proto.Int32(int32(sc.MaxPacket)),
				RcvBuff:         proto.Int32(int32(sc.RcvBuff)),
				SndBuff:         proto.Int32(int32(sc.SndBuff)),
				SoLinger:        proto.Int32(int32(sc.SoLinger)),
				KeepAlive:       proto.Bool(sc.KeepAlive),
				NoDelay:         proto.Bool(sc.NoDelay),
				IsAutoReconn:    proto.Bool(sc.IsAutoReconn),
				IsInnerLink:     proto.Bool(sc.IsInnerLink),
				SupportFragment: proto.Bool(sc.SupportFragment),
				AllowMultiConn:  proto.Bool(sc.AllowMultiConn),
				AuthKey:         proto.String(sc.AuthKey),
				EncoderName:     proto.String(sc.EncoderName),
				DecoderName:     proto.String(sc.DecoderName),
				FilterChain:     sc.FilterChain,
				HandlerChain:    sc.HandlerChain,
			}
			pack.Services = append(pack.Services, si)
		}
		s.Send(pack)
	}
}

func (this *serviceMgr) GetServices(srvtype int32) map[int32]*protocol.ServiceInfo {
	if v, has := this.servicesPool[srvtype]; has {
		return v
	}
	return nil
}

func init() {

	// service registe
	netlib.RegisterFactory(int(protocol.SrvlibPacketID_PACKET_SS_SERVICE_REGISTE), netlib.PacketFactoryWrapper(func() interface{} {
		return &protocol.SSServiceRegiste{}
	}))
	netlib.RegisterHandler(int(protocol.SrvlibPacketID_PACKET_SS_SERVICE_REGISTE), netlib.HandlerWrapper(func(s *netlib.Session, pack interface{}) error {
		if sr, ok := pack.(*protocol.SSServiceRegiste); ok {
			ServiceMgr.RegisteService(s, sr.GetServices())
		}
		return nil
	}))

	// service info
	netlib.RegisterFactory(int(protocol.SrvlibPacketID_PACKET_SS_SERVICE_INFO), netlib.PacketFactoryWrapper(func() interface{} {
		return &protocol.SSServiceInfo{}
	}))
	netlib.RegisterHandler(int(protocol.SrvlibPacketID_PACKET_SS_SERVICE_INFO), netlib.HandlerWrapper(func(s *netlib.Session, pack interface{}) error {
		if sr, ok := pack.(*protocol.SSServiceInfo); ok {
			service := sr.GetService()
			if service != nil {
				sc := &netlib.SessionConfig{
					Id:              int(service.GetSrvId()),
					Type:            int(service.GetSrvType()),
					AreaId:          int(service.GetAreaId()),
					Name:            service.GetSrvName(),
					Ip:              service.GetIp(),
					Port:            int(service.GetPort()),
					WriteTimeout:    time.Duration(service.GetWriteTimeOut()),
					ReadTimeout:     time.Duration(service.GetReadTimeOut()),
					IdleTimeout:     time.Duration(service.GetIdleTimeOut()),
					MaxDone:         int(service.GetMaxDone()),
					MaxPend:         int(service.GetMaxPend()),
					MaxPacket:       int(service.GetMaxPacket()),
					RcvBuff:         int(service.GetRcvBuff()),
					SndBuff:         int(service.GetSndBuff()),
					IsClient:        true,
					IsAutoReconn:    true,
					AuthKey:         service.GetAuthKey(),
					SoLinger:        int(service.GetSoLinger()),
					KeepAlive:       service.GetKeepAlive(),
					NoDelay:         service.GetNoDelay(),
					IsInnerLink:     service.GetIsInnerLink(),
					SupportFragment: service.GetSupportFragment(),
					AllowMultiConn:  service.GetAllowMultiConn(),
					EncoderName:     service.GetEncoderName(),
					DecoderName:     service.GetDecoderName(),
					FilterChain:     service.GetFilterChain(),
					HandlerChain:    service.GetHandlerChain(),
				}
				sc.Init()
				err := netlib.Connect(sc)
				if err != nil {
					logger.Warn("connect server failed err:", err)
				}
			}
		}
		return nil
	}))

	// service shutdown
	netlib.RegisterFactory(int(protocol.SrvlibPacketID_PACKET_SS_SERVICE_SHUT), netlib.PacketFactoryWrapper(func() interface{} {
		return &protocol.SSServiceShut{}
	}))
	netlib.RegisterHandler(int(protocol.SrvlibPacketID_PACKET_SS_SERVICE_SHUT), netlib.HandlerWrapper(func(s *netlib.Session, pack interface{}) error {
		if sr, ok := pack.(*protocol.SSServiceShut); ok {
			service := sr.GetService()
			if service != nil {
				netlib.ShutConnector(service.GetIp(), int(service.GetPort()))
			}
		}
		return nil
	}))

	ServerSessionMgrSington.SetListener(ServiceMgr)
}
