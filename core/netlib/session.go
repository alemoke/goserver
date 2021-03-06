// session
package netlib

import (
	"net"
	"strconv"
	"time"

	"github.com/idealeak/goserver/core/container"
	"github.com/idealeak/goserver/core/logger"
	"github.com/idealeak/goserver/core/utils"
)

var (
	SendRoutinePoison interface{} = nil
)

type SessionCloseListener interface {
	onClose(s *Session)
}

type Session struct {
	Id          int
	sendBuffer  chan interface{}
	recvBuffer  chan *action
	conn        net.Conn
	sc          *SessionConfig
	attributes  *container.SynchronizedMap
	scl         SessionCloseListener
	createTime  time.Time
	lastSndTime time.Time
	lastRcvTime time.Time
	waitor      *utils.Waitor
	quit        bool
	shutSend    bool
	shutRecv    bool
	IsConned    bool
}

func newSession(id int, conn net.Conn, sc *SessionConfig, scl SessionCloseListener) *Session {
	s := &Session{
		Id:         id,
		sc:         sc,
		scl:        scl,
		conn:       conn,
		createTime: time.Now(),
		waitor:     utils.NewWaitor(),
	}

	s.init()

	return s
}

func (s *Session) init() {
	s.sendBuffer = make(chan interface{}, s.sc.MaxPend)
	s.recvBuffer = make(chan *action, s.sc.MaxDone)
	s.attributes = container.NewSynchronizedMap()
}

func (s *Session) SetAttribute(key, value interface{}) bool {
	return s.attributes.Set(key, value)
}

func (s *Session) RemoveAttribute(key interface{}) {
	s.attributes.Delete(key)
}

func (s *Session) GetAttribute(key interface{}) interface{} {
	return s.attributes.Get(key)
}

func (s *Session) GetSessionConfig() *SessionConfig {
	return s.sc
}

func (s *Session) RemoteAddr() string {
	return s.conn.RemoteAddr().String()
}

func (s *Session) start() {
	s.lastRcvTime = time.Now()
	go s.recvRoutine()
	go s.sendRoutine()
}

func (s *Session) sendRoutine() {

	defer func() {
		if err := recover(); err != nil {
			logger.Trace(s.Id, " ->close: Session.procSend err: ", err)
		}
		s.sc.encoder.FinishEncode(s)
		s.shutWrite()
		s.Close()
	}()

	s.waitor.Add(1)
	defer s.waitor.Done()

	var (
		err  error
		data []byte
	)

	for !s.quit || len(s.sendBuffer) != 0 {
		select {
		case msg, ok := <-s.sendBuffer:
			if !ok {
				panic("[comm expt]sendBuffer chan closed")
			}

			if msg == nil {
				panic("[comm expt]normal close send")
			}

			if s.sc.WriteTimeout != 0 {
				s.conn.SetWriteDeadline(time.Now().Add(s.sc.WriteTimeout))
			}

			data, err = s.sc.encoder.Encode(s, msg, s.conn)
			if err != nil {
				logger.Trace("s.sc.encoder.Encode err", err)
				if s.sc.IsInnerLink == false {
					panic(err)
				}
			}
			s.FirePacketSent(data)
			s.lastSndTime = time.Now()
		}
	}
}

func (s *Session) recvRoutine() {

	defer func() {
		if err := recover(); err != nil {
			logger.Trace(s.Id, " ->close: Session.procRecv err: ", err)
		}
		s.sc.decoder.FinishDecode(s)
		s.shutRead()
		s.Close()
	}()

	s.waitor.Add(1)
	defer s.waitor.Done()

	var (
		err      error
		pck      interface{}
		packetid int
		raw      []byte
	)

	for !s.quit {
		if s.sc.ReadTimeout != 0 {
			s.conn.SetReadDeadline(time.Now().Add(s.sc.ReadTimeout))
		}

		packetid, pck, err, raw = s.sc.decoder.Decode(s, s.conn)
		if err != nil {
			bUnproc := true
			bPackErr := false
			if _, ok := err.(*UnparsePacketTypeErr); ok {
				bPackErr = true
				if s.sc.eph != nil && s.sc.eph.OnErrorPacket(s, packetid, raw) {
					bUnproc = false
				}
			}
			if bUnproc {
				logger.Trace("s.sc.decoder.Decode err ", err)
				if s.sc.IsInnerLink == false {
					panic(err)
				} else if !bPackErr {
					panic(err)
				}
			}
		}
		if pck != nil {
			if s.FirePacketReceived(packetid, pck) {
				act := AllocAction()
				act.s = s
				act.p = pck
				act.packid = packetid
				act.n = "packet:" + strconv.Itoa(packetid)
				s.recvBuffer <- act
			}
		}
		s.lastRcvTime = time.Now()
	}
}

func (s *Session) destroy() {
	s.FireDisconnectEvent()
}

func (s *Session) IsIdle() bool {
	return s.lastRcvTime.Add(s.sc.IdleTimeout).Before(time.Now())
}

func (s *Session) Close() {
	//logger.Trace(utils.GetCallStack())
	if s.quit {
		return
	}

	s.quit = true

	go s.reapRoutine()
}

func (s *Session) reapRoutine() {
	if !s.shutSend {
		//close send goroutiue(throw a poison)
		s.sendBuffer <- SendRoutinePoison
	}

	if !s.shutRecv {
		//close recv goroutiue
		s.shutRead()
	}

	s.waitor.Wait()
	s.scl.onClose(s)
}

func (s *Session) Send(msg interface{}, sync ...bool) bool {
	if s.quit || s.shutSend {
		return false
	}

	if len(sync) > 0 {
		select {
		case s.sendBuffer <- msg:
		case <-time.After(time.Duration(s.sc.WriteTimeout)):
			logger.Info(s.Id, " send buffer full,data be droped")
			if s.sc.IsInnerLink == false {
				s.Close()
			}
			return false
		}
	} else {
		select {
		case s.sendBuffer <- msg:
		default:
			logger.Info(s.Id, " send buffer full,data be droped")
			if s.sc.IsInnerLink == false {
				s.Close()
			}
			return false
		}
	}

	return true
}

func (s *Session) shutRead() {
	if s.shutRecv {
		return
	}

	s.shutRecv = true
	if tcpconn, ok := s.conn.(*net.TCPConn); ok {
		tcpconn.CloseRead()
	}
}

func (s *Session) shutWrite() {
	if s.shutSend {
		return
	}

	rest := len(s.sendBuffer)
	for rest > 0 {
		<-s.sendBuffer
		rest--
	}

	s.shutSend = true
	if tcpconn, ok := s.conn.(*net.TCPConn); ok {
		tcpconn.CloseWrite()
	}
}

func (s *Session) canShutdown() bool {
	return s.shutRecv && s.shutSend
}

func (s *Session) FireConnectEvent() bool {
	s.IsConned = true
	if s.sc.sfc != nil {
		if !s.sc.sfc.OnSessionOpened(s) {
			return false
		}
	}
	if s.sc.shc != nil {
		s.sc.shc.OnSessionOpened(s)
	}
	return true
}

func (s *Session) FireDisconnectEvent() bool {
	s.IsConned = false
	if s.sc.sfc != nil {
		if !s.sc.sfc.OnSessionClosed(s) {
			return false
		}
	}
	if s.sc.shc != nil {
		s.sc.shc.OnSessionClosed(s)
	}
	return true
}

func (s *Session) FirePacketReceived(packetid int, packet interface{}) bool {
	if s.sc.sfc != nil {
		if !s.sc.sfc.OnPacketReceived(s, packetid, packet) {
			return false
		}
	}
	if s.sc.shc != nil {
		s.sc.shc.OnPacketReceived(s, packetid, packet)
	}
	return true
}

func (s *Session) FirePacketSent(data []byte) bool {
	if s.sc.sfc != nil {
		if !s.sc.sfc.OnPacketSent(s, data) {
			return false
		}
	}
	if s.sc.shc != nil {
		s.sc.shc.OnPacketSent(s, data)
	}
	return true
}

func (s *Session) FireSessionIdle() bool {
	if s.sc.sfc != nil {
		if !s.sc.sfc.OnSessionIdle(s) {
			return false
		}
	}
	if s.sc.shc != nil {
		s.sc.shc.OnSessionIdle(s)
	}
	return true
}
