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
	return nh
}

func (serialConn *SerialConnection) ReadFrame(buffer []byte, position int) {
	frame := NewApiFrame(buffer[0:position], true)
	if serialConn.debug {
		fmt.Printf("<-- %s\n", hex.EncodeToString(frame.data))
	}
	if serialConn.session != nil {
		if serialConn.session.WaitReply > 0 {
			if frame.AssertType(serialConn.session.WaitReply) {
				serialConn.session.Reply = frame
				serialConn.session.Wait.Done()
				serialConn.session = nil
			}
		}
	} else {
		v, err := frame.Decode()
		if err != nil {
			log.Println("Can't decoded packet!!!", err)
		} else {
			switch t := v.(type) {
			case LogEventApiReply:
				log.WithFields(logrus.Fields{"from": t.From}).Info(t.Line)
			case ConnectedPathApiReply:
				if serialConn.ConnPathFn != nil {
					serialConn.ConnPathFn(&t)
				}
			default:
				log.Printf("Received packet %v", v)
			}
		}
	}
}

func (conn *SerialConnection) checkTimeout() {
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
		serialConn.port.SetReadTimeout(50 * time.Millisecond)
		n, err := serialConn.port.Read(buffer)
		if err != nil {
			break
		}

		if n == 0 {
			serialConn.checkTimeout()
		} else if n > 0 {
			b := buffer[0]
			//fmt.Println(hex.EncodeToString(buffer))
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
		if serialConn.session == nil {
			if serialConn.Sessions.Len() == 0 {
				time.Sleep(50 * time.Millisecond)
			} else {
				session := serialConn.Sessions.Front().Value.(*SerialSession)
				serialConn.Sessions.Remove(serialConn.Sessions.Front())

				b := session.Request.Output()
				if serialConn.debug {
					fmt.Printf("--> %s\n", hex.EncodeToString(b))
				}
				n, err := serialConn.port.Write(b)

				if err != nil {
					log.Println(err)
					break
				}

				if n < len(b) {
					log.Println("not sent all bytes")
					break
				}

				if session.WaitReply > 0 {
					serialConn.session = session
				}
			}
		} else {
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
	frame, err := NewApiFrameFromStruct(cmd)
	if err != nil {
		return err
	}

	session := NewSimpleSerialSession(frame)
	serialConn.QueueApiSession(session)
	return nil
}

func (serialConn *SerialConnection) SendReceiveApi(cmd interface{}) (interface{}, error) {
	frame, err := NewApiFrameFromStruct(cmd)
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

	serial.LocalNode = nodeid.Serial
	log.Printf("NodeId is %06X/%06X with firmware %s\n", nodeid.Serial, serial.LocalNode, firmrev.Revision)
	return serial, nil
}
