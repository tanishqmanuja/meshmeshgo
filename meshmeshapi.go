package main

import (
	"container/list"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"go.bug.st/serial"
)

type SerialSession struct {
	Request   *ApiFrame
	Reply     *ApiFrame
	WaitReply uint8
	Wait      sync.WaitGroup
}

func NewSerialSession(request *ApiFrame) *SerialSession {
	s := SerialSession{Request: request, WaitReply: request.AwaitedReply()}
	return &s
}

type SerialConnection struct {
	connected bool
	port      serial.Port
	incoming  chan []byte
	session   *SerialSession
	Sessions  *list.List
}

const (
	waitStartByte  = iota
	escapeNextByte = iota
	waitEndByte    = iota
)

func (serialConn *SerialConnection) ReadFrame(buffer []byte, position int) {
	frame := NewApiFrame(buffer[0:position], true)
	if serialConn.session != nil && frame.AssertType(serialConn.session.WaitReply) {
		serialConn.session.Reply = frame
		serialConn.session.Wait.Done()
		serialConn.session = nil
	} else {
		v, err := frame.Decode()
		if err != nil {
			log.Println("Can't decoded packet!!!", err)
		} else {
			switch t := v.(type) {
			case LogEventApiReply:
				log.Printf("From %06X Log %s\n", t.From, t.Line)
			default:
				log.Printf("Received packet %v", v)
			}
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
		serialConn.port.SetReadTimeout(100 * time.Millisecond)
		n, err := serialConn.port.Read(buffer)
		if err != nil {
			break
		}

		if n > 0 {
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
				serialConn.session = serialConn.Sessions.Front().Value.(*SerialSession)
				serialConn.Sessions.Remove(serialConn.Sessions.Front())

				b := serialConn.session.Request.Output()
				fmt.Printf("--> %s\n", hex.EncodeToString(b))
				n, err := serialConn.port.Write(b)
				if err != nil {
					log.Println(err)
					break
				}
				if n < len(b) {
					log.Println("not sent all bytes")
					break
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

func (serialConn *SerialConnection) WriteApiFrame(frame *ApiFrame) error {
	b := frame.Output()
	fmt.Printf("--> %s\n", hex.EncodeToString(b))
	n, err := serialConn.port.Write(b)
	if n < len(b) {
		err = errors.New("not sent all bytes")
	}
	return err
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
	v, err := session.Reply.Decode()

	return v, err
}

func NewSerial(portName string, baudRate int) (*SerialConnection, error) {
	mode := &serial.Mode{BaudRate: baudRate}
	p, err := serial.Open(portName, mode)
	if err != nil {
		return nil, err
	}

	client := &SerialConnection{
		connected: true,
		port:      p,
		incoming:  make(chan []byte),
		Sessions:  list.New(),
	}

	go client.Write()
	go client.Read()

	reply1, err := client.SendReceiveApi(EchoApiRequest{Echo: "CIAO"})
	if err != nil {
		client.port.Close()
		return nil, err
	}
	echo, ok := reply1.(EchoApiReply)
	if !ok {
		client.port.Close()
		return nil, errors.New("invalid echo reply type")
	}
	if echo.Echo != "CIAO" {
		client.port.Close()
		return nil, errors.New("invalid echo reply")
	}

	reply2, err := client.SendReceiveApi(NodeIdApiRequest{})
	if err != nil {
		client.port.Close()
		return nil, err
	}
	nodeid, ok := reply2.(NodeIdApiReply)
	if !ok {
		client.port.Close()
		return nil, errors.New("invalid nodeid reply")
	}

	reply3, err := client.SendReceiveApi(FirmRevApiRequest{})
	if err != nil {
		client.port.Close()
		return nil, err
	}
	firmrev, ok := reply3.(FirmRevApiReply)
	if !ok {
		client.port.Close()
		return nil, errors.New("invalid firmware reply")
	}

	log.Printf("NodeId is %06X with firmware %s\n", nodeid.Serial, firmrev.Revision)
	return client, nil
}
