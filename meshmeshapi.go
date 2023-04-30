package main

import (
	"container/list"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"go.bug.st/serial"
)

type SerialSession struct {
	Request   *ApiFrame
	Reply     *ApiFrame
	WaitReply uint8
	Wait      sync.WaitGroup
	SentTime  time.Time
}

func NewSimpleSerialSession(request *ApiFrame) *SerialSession {
	s := SerialSession{Request: request, WaitReply: 0}
	return &s
}

func NewSerialSession(request *ApiFrame) *SerialSession {
	s := SerialSession{Request: request, WaitReply: request.AwaitedReply()}
	s.SentTime = time.Now()
	return &s
}

type SerialConnection struct {
	connected  bool
	port       serial.Port
	debug      bool
	incoming   chan []byte
	session    *SerialSession
	Sessions   *list.List
	NextHandle uint16
	LocalNode  uint32
	ConnPathFn func(*ConnectedPathApiReply)
}

const (
	waitStartByte  = iota
	escapeNextByte = iota
	waitEndByte    = iota
)

func (serialConn *SerialConnection) GetNextHandle() uint16 {
	nh := serialConn.NextHandle
	serialConn.NextHandle += 1
	// Never use handle 0
	if serialConn.NextHandle == 0 {
		serialConn.NextHandle += 1
	}
	return nh
}

func (serialConn *SerialConnection) ReadFrame(buffer []byte, position int) {
	frame := NewApiFrame(buffer[0:position], true)
	if log.GetLevel() >= logrus.TraceLevel && buffer[0] != 0x39 {
		log.WithFields(logrus.Fields{"data": hex.EncodeToString(frame.data)}).Trace("From serial")
	}
	// Handle LOG packets first
	if buffer[0] == logEventApiReply {
		v, err := frame.Decode()
		if err != nil {
			log.Error("Can't decode incoming log packet 1/2")
		} else {
			l, ok := v.(LogEventApiReply)
			if !ok {
				log.Error("Can't decode incoming log packet 2/2")
			}
			log.WithFields(logrus.Fields{"from": l.From}).Debug(l.Line)
		}
		// Handle ConnectedPath packets next
	} else if buffer[0] == connectedPathApiReply {
		v, err := frame.Decode()
		if err != nil {
			log.Error("Can't decode incoming connectedpath packet 1/2")
		} else {
			c, ok := v.(ConnectedPathApiReply)
			if !ok {
				log.Error("Can't decode incoming connectedpath packet 2/2")
			}
			if serialConn.ConnPathFn != nil {
				serialConn.ConnPathFn(&c)
			}
		}
	} else {
		// Handle session pacekts next
		if serialConn.session != nil {
			if serialConn.session.WaitReply > 0 {
				if frame.AssertType(serialConn.session.WaitReply) {
					serialConn.session.Reply = frame
					serialConn.session.Wait.Done()
					serialConn.session = nil
				}
			}
		} else {
			log.WithField("type", fmt.Sprintf("%02X", buffer[0])).Error("Unused packet received")
		}
	}
}

func (conn *SerialConnection) checkSessionTimeout() {
	if conn.session != nil {
		if time.Since(conn.session.SentTime).Milliseconds() > 500 {
			conn.session.Reply = nil
			if conn.session.WaitReply > 0 {
				conn.session.Wait.Done()
			}
			conn.session = nil
		}
	}
}

func (serialConn *SerialConnection) Read() {
	var inputBufferPos int
	inputBuffer := make([]byte, 1500)
	var decodeState int = waitStartByte
	serialConn.port.ResetInputBuffer()

	for {
		var buffer = make([]byte, 1)
		// Read a byte from serial with a timout of a time slot
		serialConn.port.SetReadTimeout(50 * time.Millisecond)
		n, err := serialConn.port.Read(buffer)
		if err != nil {
			break
		}

		if n == 0 {
			// We don't receive any data check if we want a reply
			serialConn.checkSessionTimeout()
		} else if n > 0 {
			b := buffer[0]
			if decodeState == waitStartByte {
				if b == startApiFrame {
					inputBufferPos = 0
					decodeState = waitEndByte
				} else {
					fmt.Println("unknow char", b)
				}
			} else if decodeState == escapeNextByte {
				decodeState = waitEndByte
				inputBuffer[inputBufferPos] = b
				inputBufferPos += 1
			} else {
				if b == stopApiFrame {
					decodeState = waitStartByte
					serialConn.ReadFrame(inputBuffer, inputBufferPos)
					inputBufferPos = 0
				} else if b == escapeApiFrame {
					decodeState = escapeNextByte
				} else {
					inputBuffer[inputBufferPos] = b
					inputBufferPos += 1
				}
			}
		}
	}

	serialConn.connected = false
	serialConn.port.Close()
}

func (serialConn *SerialConnection) Write() {
	for {
		// If we are idle
		if serialConn.session == nil {
			// And there is not more work to do
			if serialConn.Sessions.Len() == 0 {
				// Sleep a time slot
				time.Sleep(50 * time.Millisecond)
			} else {
				// We are idle but we have work to do...
				element := serialConn.Sessions.Front().Value
				// Get next session and remove from list
				session, ok := element.(*SerialSession)
				serialConn.Sessions.Remove(serialConn.Sessions.Front())

				if ok {
					b := session.Request.Output()
					if log.GetLevel() >= logrus.TraceLevel {
						log.WithFields(logrus.Fields{"data": hex.EncodeToString(b)}).Trace("To serial")
					}

					// Write session on serial port
					n, err := serialConn.port.Write(b)

					if err != nil {
						log.WithField("err", err).Error("Write to serial port error")
						break
					}

					if n < len(b) {
						log.WithFields(logrus.Fields{"sent": n, "want": len(b)}).Error("Write to serial port incomplete")
						break
					}

					if session.WaitReply > 0 {
						// If we need a reply mark we as busy
						serialConn.session = session
					} else {
						// Sleep a time slot beofre send next session
						// Is a guard time for wifi retransmissions
						time.Sleep(50 * time.Millisecond)
					}
				} else {
					// Ok we don't really need this
					log.WithFields(logrus.Fields{"queue": serialConn.Sessions.Len(), "val": element}).
						Error("interface conversion invalid")
				}

			}
		} else {
			// We are busy Sleep a time slot before check again
			time.Sleep(50 * time.Millisecond)
		}
	}

	serialConn.connected = false
	serialConn.port.Close()
}

func (serialConn *SerialConnection) QueueApiSession(session *SerialSession) {
	serialConn.Sessions.PushBack(session)
}

func (serialConn *SerialConnection) SendApi(cmd interface{}) error {
	frame, err := NewApiFrameFromStruct(cmd, directProtocol, 0)
	if err != nil {
		return err
	}

	session := NewSimpleSerialSession(frame)
	serialConn.QueueApiSession(session)
	return nil
}

func (serialConn *SerialConnection) SendReceiveApiProt(cmd interface{}, protocol MeshProtocol, target MeshNodeId) (interface{}, error) {
	frame, err := NewApiFrameFromStruct(cmd, protocol, target)
	if err != nil {
		return nil, err
	}

	session := NewSerialSession(frame)
	session.Wait.Add(1)
	serialConn.QueueApiSession(session)
	session.Wait.Wait()

	if session.Reply == nil {
		return nil, errors.New("reply timeout")
	} else {
		return session.Reply.Decode()
	}
}

func (serialConn *SerialConnection) SendReceiveApi(cmd interface{}) (interface{}, error) {
	return serialConn.SendReceiveApiProt(cmd, directProtocol, 0)
}

func NewSerial(portName string, baudRate int, debug bool) (*SerialConnection, error) {
	mode := &serial.Mode{BaudRate: baudRate}
	p, err := serial.Open(portName, mode)
	if err != nil {
		return nil, err
	}

	serial := &SerialConnection{
		connected:  true,
		port:       p,
		debug:      debug,
		incoming:   make(chan []byte),
		Sessions:   list.New(),
		NextHandle: 1,
	}

	go serial.Write()
	go serial.Read()

	reply1, err := serial.SendReceiveApi(EchoApiRequest{Echo: "CIAO"})
	if err != nil {
		serial.port.Close()
		return nil, err
	}
	echo, ok := reply1.(EchoApiReply)
	if !ok {
		serial.port.Close()
		return nil, errors.New("invalid echo reply type")
	}
	if echo.Echo != "CIAO" {
		serial.port.Close()
		return nil, errors.New("invalid echo reply")
	}

	reply2, err := serial.SendReceiveApi(NodeIdApiRequest{})
	if err != nil {
		serial.port.Close()
		return nil, err
	}
	nodeid, ok := reply2.(NodeIdApiReply)
	if !ok {
		serial.port.Close()
		return nil, errors.New("invalid nodeid reply")
	}

	reply3, err := serial.SendReceiveApi(FirmRevApiRequest{})
	if err != nil {
		serial.port.Close()
		return nil, err
	}
	firmrev, ok := reply3.(FirmRevApiReply)
	if !ok {
		serial.port.Close()
		return nil, errors.New("invalid firmware reply")
	}

	serial.LocalNode = uint32(nodeid.Serial)
	log.WithFields(logrus.Fields{"nodeId": fmt.Sprintf("0x%06X", serial.LocalNode), "firmware": firmrev.Revision}).
		Info("Valid local node found")
	return serial, nil
}
