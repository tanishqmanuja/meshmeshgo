package meshmesh

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"leguru.net/m/v2/graph"
	"leguru.net/m/v2/utils"
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
			logrus.WithField("handle", client.connpath.handle).
				Trace(fmt.Sprintf("HA-->SE: %s", hex.EncodeToString(client.inBuffer.Bytes())))
			err := client.connpath.SendData(client.inBuffer.Bytes())
			if err != nil {
				logrus.Error(fmt.Sprintf("Error writng on socket: %s", err.Error()))
			}
			client.inBuffer.Reset()
		}
	}
}

func (client *ApiConnection) startHandshake(addr MeshNodeId, port int) error {
	client.reqAddress = addr
	client.reqPort = port
	err := client.connpath.OpenConnectionAsync(addr, uint16(port))
	if err == nil {
		client.stats = _allStats.StartConnection(addr)
		if addr == MeshNodeId(0) {
			client.debugThisNode = true
			logrus.WithFields(logrus.Fields{"id": fmt.Sprintf("%02X", addr)}).Info("startHandshake and debug for node")
		}
	}
	return err
}

func (client *ApiConnection) finishHandshake(result bool) {
	logrus.WithField("res", result).Debug("finishHandshake")
	if !result {
		logrus.WithFields(logrus.Fields{"addr": client.reqAddress, "port": client.reqPort, "err": nil}).
			Warning("ApiConnection.finishHandshake failed")
	} else {
		logrus.WithFields(logrus.Fields{"addr": client.reqAddress, "port": client.reqPort, "handle": client.connpath.handle}).
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
	utils.ForceDebug(client.debugThisNode, "Waiting for read go-routine to terminate...")
	client.socketWaitGroup.Wait()
	utils.ForceDebugEntry(logrus.WithFields(
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
				logrus.Error("Closing connection beacuse timeout in connPathConnectionStateInit")
				client.Close()
			}
		}
		time.Sleep(100 * time.Millisecond)
	}

	logrus.Debug("ApiConnection.CheckTimeout exited")
}

func (client *ApiConnection) Read() {
	var err error

	for {
		var buffer = make([]byte, 1)
		_, err = client.socket.Read(buffer)

		if err == nil {
			if client.connpath.connState == connPathConnectionStateHandshakeStarted {
				// FIXME check for if buffer grown outside limits
				client.tmpBuffer.WriteByte(buffer[0])
			} else if client.connpath.connState == connPathConnectionStateActive {
				// FIXME handle error
				client.forward(buffer[0])
			} else {
				logrus.WithField("state", client.connpath.connState).
					Error(fmt.Errorf("readed data while in wrong connection state %d", client.connpath.connState))
			}
		} else {
			logrus.WithFields(logrus.Fields{"handle": client.connpath.handle, "err": err}).Warn("ApiConnection.Read exit with error")
			break
		}
	}

	client.socketWaitGroup.Done()
	if client.socketOpen {
		client.Close()
	}
}

func (client *ApiConnection) ForwardData(data []byte) error {
	logrus.WithField("handle", client.connpath.handle).
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

func NewApiConnection(connection net.Conn, serial *SerialConnection, graph *graph.GraphPath) *ApiConnection {
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
	/*logrus.Printf("NODE: received cmd:%d handle:%d size:%d %s", v.Command, v.Handle, len(v.Data), hex.EncodeToString(v.Data))
	if v.Command == 5 && len(v.Data) >= 2 {
		msgsize := v.Data[1]
		msgtype := v.Data[2]
		logrus.Printf("      proto size:%d type:%d %s", msgsize, msgtype, hex.EncodeToString(v.Data[3:]))
	}*/
	var handled bool = false
	for client := range allClients {
		if client.connpath.handle == v.Handle {
			handled = true
			if v.Command == connectedPathSendDataRequest {
				if len(v.Data) > 0 {
					err := client.ForwardData(v.Data)
					if err != nil {
						logrus.Printf("HandleConnectedPathReply: ForwardData error on handle %d.", v.Handle)
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
		logrus.WithFields(logrus.Fields{"cmd": v.Command, "handle": v.Handle}).
			Error("HandleConnectedPathReply: Connection not found for this handle")
		SendInvalidHandle(_serial, v.Handle)
	}
}

func PrintStats() {
	if _allStats != nil {
		_allStats.PrintStats()
	}
}

func ListenToApiConnetions(serial *SerialConnection, graph *graph.GraphPath, host string, port int, addr MeshNodeId) {
	serial.ConnPathFn = HandleConnectedPathReply
	allClients = make(map[*ApiConnection]int)
	l, err := net.Listen("tcp4", fmt.Sprintf("%s:%d", host, port))
	logrus.WithFields(logrus.Fields{"port": port, "addr": addr}).Debug("Start listening on port for direct node connection")
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
			logrus.Println(err)
			continue
		}

		logrus.WithFields(logrus.Fields{"active": len(allClients), "port": port}).Debug("EspHome connection accepted")
		client := NewApiConnection(c, serial, graph)
		allClients[client] = 1
		client.startHandshake(addr, 6053)
	}
}
