package main

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// var devices []string = []string{"0.244.38.24", "0.149.58.251", "0.116.78.13", "0.112.83.1"}
var allClients map[*ApiConnection]int
var _allStats *EspApiStats
var _serial *SerialConnection

const (
	esphomeapiWaitPacketHead int = 0
	esphomeapiWaitPacketSize int = 1
	esphomeapiWaitPacketData int = 3
)

type ApiConnection struct {
	inState         int
	inAwaitSize     int
	tmpBuffer       *bytes.Buffer
	inBuffer        *bytes.Buffer
	socketOpen      bool
	socket          net.Conn
	socketWaitGroup sync.WaitGroup
	connpath        *ConnPathConnection
	reqAddress      MeshNodeId
	reqPort         int
	stats           *EspApiConnectionStats
	debugThisNode   bool
	timeout         time.Time
}

func (client *ApiConnection) forward(lastbyte byte) {
	client.inBuffer.WriteByte(lastbyte)
	if client.inState == esphomeapiWaitPacketHead {
		if lastbyte == 0x00 {
			client.inState = esphomeapiWaitPacketSize
		} else {
			client.inBuffer.Reset()
		}
	} else if client.inState == esphomeapiWaitPacketSize {
		client.inAwaitSize = int(lastbyte) + 3
		client.inState = esphomeapiWaitPacketData
	} else {
		if client.inBuffer.Len() == client.inAwaitSize {
			client.inState = esphomeapiWaitPacketHead
			log.WithField("handle", client.connpath.handle).
				Trace(fmt.Sprintf("HA-->SE: %s", hex.EncodeToString(client.inBuffer.Bytes())))
			err := client.connpath.SendData(client.inBuffer.Bytes())
			if err != nil {
				log.Error(fmt.Sprintf("Error writng on socket: %s", err.Error()))
			}
			client.inBuffer.Reset()
		}
	}
}

func (client *ApiConnection) startHandshake() error {
	var err error

	line := strings.TrimSpace(client.inBuffer.String())
	client.inBuffer.Reset()

	log.WithFields(logrus.Fields{"line": line}).Debug("startHandshake: Requested handshake from HA")
	fields := strings.Split(strings.TrimSpace(line), "|")

	if len(fields) == 3 {
		var port int
		addr := strings.TrimSpace(fields[1])
		port, err = strconv.Atoi(strings.TrimSpace(fields[2]))
		if err == nil {
			var _addr int64
			_addr, err = parseNodeId(addr)
			if err == nil {
				err = client.startHandshake2(MeshNodeId(_addr), port)
			}
		}
	} else {
		err = errors.New("wrong hadshake header")
	}

	if err != nil {
		log.WithFields(logrus.Fields{"handle": client.connpath.handle, "err": err}).Error("startHandshake open connection error")
		client.finishHandshake(false)
	}

	return err
}

func (client *ApiConnection) startHandshake2(addr MeshNodeId, port int) error {
	client.reqAddress = addr
	client.reqPort = port
	err := client.connpath.OpenConnectionAsync(addr, uint16(port))
	if err == nil {
		client.stats = _allStats.StartConnection(addr)
		if addr == MeshNodeId(debugNodeId) {
			client.debugThisNode = true
			log.WithFields(logrus.Fields{"id": fmt.Sprintf("%02X", addr)}).Info("startHandshake and debug for node")
		}
	}
	return err
}

func (client *ApiConnection) finishHandshake(result bool) {
	log.WithField("res", result).Debug("finishHandshake")
	if !result {
		log.WithFields(logrus.Fields{"addr": client.reqAddress, "port": client.reqPort, "err": nil}).
			Warning("ApiConnection.finishHandshake failed")
	} else {
		log.WithFields(logrus.Fields{"addr": client.reqAddress, "port": client.reqPort, "handle": client.connpath.handle}).
			Info("ApiConnection.handshake OpenConnection succesfull")
		client.flushBuffer()
		client.stats.GotHandle(client.connpath.handle)
	}
}

func (client *ApiConnection) flushBuffer() {
	if client.tmpBuffer.Len() > 0 {
		_b := client.tmpBuffer.Bytes()
		for i := 0; i < len(_b); i++ {
			client.forward(_b[i])
		}
	}
}

func (client *ApiConnection) Close() {
	client.socketOpen = false
	client.socket.Close()
	ForceDebug(client.debugThisNode, "Waiting for read go-routine to terminate...")
	client.socketWaitGroup.Wait()
	ForceDebugEntry(log.WithFields(
		logrus.Fields{"handle": client.connpath.handle, "size": len(allClients) - 1}),
		client.debugThisNode,
		"Closed EspHomeApi connection")
	delete(allClients, client)
}

func (client *ApiConnection) CheckTimeout() {
	for {
		if !client.socketOpen {
			break
		}
		if client.connpath.connState == connPathConnectionStateInit || client.connpath.connState == connPathConnectionStateHandshakeStarted {
			if time.Since(client.timeout).Milliseconds() > 3000 {
				log.Error("Closing connection beacuse timeout in connPathConnectionStateInit")
				client.Close()
			}
		}
		time.Sleep(100 * time.Millisecond)
	}

	log.Debug("ApiConnection.CheckTimeout exited")
}

func (client *ApiConnection) Read() {
	var err error

	for {
		var buffer = make([]byte, 1)
		_, err = client.socket.Read(buffer)

		if err == nil {
			// FIXME check for if buffer grown outside limits
			if client.connpath.connState == connPathConnectionStateInit {
				client.inBuffer.WriteByte(buffer[0])
				if buffer[0] == '\n' {
					err = client.startHandshake()
					if err != nil {
						break
					}
				}
			} else if client.connpath.connState == connPathConnectionStateHandshakeStarted {
				// FIXME check for if buffer grown outside limits
				client.tmpBuffer.WriteByte(buffer[0])
			} else if client.connpath.connState == connPathConnectionStateActive {
				// FIXME handle error
				client.forward(buffer[0])
			} else {
				log.WithField("state", client.connpath.connState).
					Error("Readed data while in wrong connection state")
			}
		} else {
			break
		}
	}

	if err != nil {
		log.WithFields(logrus.Fields{"handle": client.connpath.handle, "err": err}).
			Warn("ApiConnection.Read exit with error")
	}

	client.socketWaitGroup.Done()
	if client.socketOpen {
		client.Close()
	}
}

func (client *ApiConnection) ForwardData(data []byte) error {
	log.WithField("handle", client.connpath.handle).
		Trace(fmt.Sprintf("SE-->HA: %s", hex.EncodeToString(data)))
	n, err := client.socket.Write(data)
	if err != nil {
		return err
	}

	if n < len(data) {
		return errors.New("socket can't receive all bytes")
	}

	return nil
}

func NewApiConnection(connection net.Conn, serial *SerialConnection, graph *GraphPath) *ApiConnection {
	client := &ApiConnection{
		connpath:   NewConnPathConnection(serial, graph),
		socketOpen: true,
		socket:     connection,
		tmpBuffer:  bytes.NewBuffer([]byte{}),
		inBuffer:   bytes.NewBuffer([]byte{}),
		timeout:    time.Now(),
	}

	client.socketWaitGroup.Add(1)
	go client.Read()
	go client.CheckTimeout()

	return client
}

func HandleConnectedPathReply(v *ConnectedPathApiReply) {
	/*log.Printf("NODE: received cmd:%d handle:%d size:%d %s", v.Command, v.Handle, len(v.Data), hex.EncodeToString(v.Data))
	if v.Command == 5 && len(v.Data) >= 2 {
		msgsize := v.Data[1]
		msgtype := v.Data[2]
		log.Printf("      proto size:%d type:%d %s", msgsize, msgtype, hex.EncodeToString(v.Data[3:]))
	}*/
	var handled bool = false
	for client := range allClients {
		if client.connpath.handle == v.Handle {
			handled = true
			if v.Command == connectedPathSendDataRequest {
				if len(v.Data) > 0 {
					err := client.ForwardData(v.Data)
					if err != nil {
						log.Printf("HandleConnectedPathReply: ForwardData error on handle %d.", v.Handle)
						client.Close()
					}
				}
			} else {
				oldConnState := client.connpath.connState
				client.connpath.HandleIncomingReply(v)
				if oldConnState != client.connpath.connState {
					if client.connpath.connState == connPathConnectionStateActive {
						client.finishHandshake(true)
					}
					if client.connpath.connState == connPathConnectionStateInvalid {
						client.Close()
					}
				}
			}
		}
	}
	if !handled {
		log.WithFields(logrus.Fields{"cmd": v.Command, "handle": v.Handle}).
			Error("HandleConnectedPathReply: Connection not found for this handle")
		SendInvalidHandle(_serial, v.Handle)
	}
}

func PrintStats() {
	if _allStats != nil {
		_allStats.PrintStats()
	}
}

func ListenToApiConnetions(serial *SerialConnection, graph *GraphPath, host string, port int, addr MeshNodeId) {
	serial.ConnPathFn = HandleConnectedPathReply
	allClients = make(map[*ApiConnection]int)
	l, err := net.Listen("tcp4", fmt.Sprintf("%s:%d", host, port))
	log.WithFields(logrus.Fields{"port": port, "addr": addr}).Debug("Start listening on port for direct node connection")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer l.Close()

	_serial = serial
	_allStats = NewEspApiStats()
	SendClearConnections(serial)

	for {
		c, err := l.Accept()
		if err != nil {
			log.Println(err)
			continue
		}

		log.WithFields(logrus.Fields{"active": len(allClients), "port": port}).Debug("EspHome connection accepted")
		client := NewApiConnection(c, serial, graph)
		allClients[client] = 1
		client.startHandshake2(addr, 6053)
	}
}
