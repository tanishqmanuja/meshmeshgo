package meshmesh

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"golang.org/x/exp/slices"
	"leguru.net/m/v2/graph"
	"leguru.net/m/v2/logger"
	"leguru.net/m/v2/utils"
)

// var devices []string = []string{"0.244.38.24", "0.149.58.251", "0.116.78.13", "0.112.83.1"}
// var allClients map[*ApiConnection]int
var _allStats *EspApiStats

//var _serial *SerialConnection

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
	clientClosed    func(client *ApiConnection)
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
			logger.WithField("handle", client.connpath.handle).
				Trace(fmt.Sprintf("HA-->SE: %s", hex.EncodeToString(client.inBuffer.Bytes())))
			err := client.connpath.SendData(client.inBuffer.Bytes())
			client.stats.SentBytes(client.inBuffer.Len())
			if err != nil {
				logger.Log().Error(fmt.Sprintf("Error writng on socket: %s", err.Error()))
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
		client.stats.Start()
		if addr == MeshNodeId(0) {
			client.debugThisNode = true
			logger.WithFields(logger.Fields{"id": fmt.Sprintf("%02X", addr)}).Info("startHandshake and debug for node")
		}
	}
	return err
}

func (client *ApiConnection) finishHandshake(result bool) {
	logger.WithField("res", result).Debug("finishHandshake")
	if !result {
		logger.WithFields(logger.Fields{"addr": client.reqAddress, "port": client.reqPort, "err": nil}).
			Warning("ApiConnection.finishHandshake failed")
	} else {
		logger.WithFields(logger.Fields{"addr": client.reqAddress, "port": client.reqPort, "handle": client.connpath.handle}).
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

func (client *ApiConnection) SetClosedCallback(cb func(client *ApiConnection)) {
	client.clientClosed = cb
}

func (client *ApiConnection) Close() {
	client.socketOpen = false
	client.socket.Close()
	utils.ForceDebug(client.debugThisNode, "Waiting for read go-routine to terminate...")
	client.socketWaitGroup.Wait()
	client.clientClosed(client)
	client.stats.Stop()
}

func (client *ApiConnection) CheckTimeout() {
	for {
		if !client.socketOpen {
			break
		}
		if client.connpath.connState == connPathConnectionStateInit || client.connpath.connState == connPathConnectionStateHandshakeStarted {
			if time.Since(client.timeout).Milliseconds() > 3000 {
				logger.Error("Closing connection beacuse timeout in connPathConnectionStateInit")
				client.Close()
			}
		}
		time.Sleep(100 * time.Millisecond)
	}

	logger.Debug("ApiConnection.CheckTimeout exited")
}

func (client *ApiConnection) Read() {
	var err error

	for {
		var buffer = make([]byte, 1)
		_, err = client.socket.Read(buffer)
		client.stats.ReceivedBytes(1)

		if err == nil {
			if client.connpath.connState == connPathConnectionStateHandshakeStarted {
				// FIXME check for if buffer grown outside limits
				client.tmpBuffer.WriteByte(buffer[0])
			} else if client.connpath.connState == connPathConnectionStateActive {
				// FIXME handle error
				client.forward(buffer[0])
			} else {
				logger.WithField("state", client.connpath.connState).
					Error(fmt.Errorf("readed data while in wrong connection state %d", client.connpath.connState))
			}
		} else {
			logger.WithFields(logger.Fields{"handle": client.connpath.handle, "err": err}).Warn("ApiConnection.Read exit with error")
			break
		}
	}

	client.socketWaitGroup.Done()
	if client.socketOpen {
		client.Close()
	}
}

func (client *ApiConnection) ForwardData(data []byte) error {
	logger.WithFields(logger.Fields{"handle": client.connpath.handle, "meshid": utils.FmtNodeId(int64(client.reqAddress))}).
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

func NewApiConnection(connection net.Conn, serial *SerialConnection, graph *graph.GraphPath, addr MeshNodeId, closedCb func(*ApiConnection)) (*ApiConnection, error) {
	client := &ApiConnection{
		connpath:     NewConnPathConnection(serial, graph),
		socketOpen:   true,
		socket:       connection,
		tmpBuffer:    bytes.NewBuffer([]byte{}),
		inBuffer:     bytes.NewBuffer([]byte{}),
		timeout:      time.Now(),
		clientClosed: closedCb,
		stats:        _allStats.Stats(addr),
	}

	err := client.startHandshake(addr, 6053)
	if err != nil {
		return nil, err
	}

	client.socketWaitGroup.Add(1)
	go client.Read()
	go client.CheckTimeout()

	return client, nil
}

/*
func HandleConnectedPathReply(v *ConnectedPathApiReply) {
	var handled bool = false
	for client := range allClients {
		if client.connpath.handle == v.Handle {
			handled = true
			if v.Command == connectedPathSendDataRequest {
				if len(v.Data) > 0 {
					err := client.ForwardData(v.Data)
					if err != nil {
						logger.Printf("HandleConnectedPathReply: ForwardData error on handle %d.", v.Handle)
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
		logger.WithFields(logger.Fields{"cmd": v.Command, "handle": v.Handle}).
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
	li, err := net.Listen("tcp4", fmt.Sprintf("%s:%d", host, port))
	logger.WithFields(logger.Fields{"port": port, "addr": addr}).Debug("Start listening on port for node connection")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer li.Close()

	_serial = serial
	_allStats = NewEspApiStats()
	SendClearConnections(serial)

	for {
		c, err := li.Accept()
		if err != nil {
			logger.Error(err)
			continue
		}

		logger.WithFields(logger.Fields{"active": len(allClients), "port": port}).Debug("EspHome connection accepted")
		client, err := NewApiConnection(c, serial, graph, addr)
		if err != nil {
			logger.Error(err)
			c.Close()
		} else {
			allClients[client] = 1
		}
	}
}*/

type ServerApi struct {
	Address  MeshNodeId
	Clients  []*ApiConnection
	listener net.Listener
}

func (m *ServerApi) HandleConnectedPathReply(v *ConnectedPathApiReply) bool {
	handled := false
	for _, client := range m.Clients {
		if client.connpath.handle == v.Handle {
			handled = true
			if v.Command == connectedPathSendDataRequest {
				if len(v.Data) > 0 {
					err := client.ForwardData(v.Data)
					if err != nil {
						logger.Printf("HandleConnectedPathReply: ForwardData error on handle %d.", v.Handle)
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
	return handled
}

func (s *ServerApi) ClientClosedCb(client *ApiConnection) {
	// Remove client from clients list
	idx := slices.Index(s.Clients, client)
	if idx >= 0 {
		s.Clients = append(s.Clients[:idx], s.Clients[idx+1:]...)
	}
	utils.ForceDebugEntry(logger.WithFields(logger.Fields{"handle": client.connpath.handle}), client.debugThisNode, "Closed EspHomeApi connection")
}

func (s *ServerApi) ListenAndServe(serial *SerialConnection, graph *graph.GraphPath) {
	for {
		c, err := s.listener.Accept()
		if err != nil {
			logger.Error(err)
			continue
		}

		logger.WithFields(logger.Fields{"nodeId": s.Address, "active": len(s.Clients)}).Debug("EspHome connection accepted")
		client, err := NewApiConnection(c, serial, graph, s.Address, s.ClientClosedCb)
		if err != nil {
			logger.Error(err)
			c.Close()
		} else {
			s.Clients = append(s.Clients, client)
			logger.WithFields(logger.Fields{"nodeId": utils.FmtNodeId(int64(s.Address)), "clients": len(s.Clients)}).Debug("Added new client")
		}
	}
}

func (s *ServerApi) ShutDown() {
	s.listener.Close()
}

func NewServerApi(serial *SerialConnection, graph *graph.GraphPath, address MeshNodeId) (*ServerApi, error) {
	server := ServerApi{Address: address}
	listener, err := net.Listen("tcp4", fmt.Sprintf("%s:%d", utils.FmtNodeIdHass(int64(address)), 6053))
	logger.WithField("addr", address).Debug("Start listening on port for node connection")
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	server.listener = listener
	go server.ListenAndServe(serial, graph)
	return &server, nil
}

func (m *MultiServerApi) HandleConnectedPathReply(v *ConnectedPathApiReply) {
	var handled bool = false
	for _, server := range m.Servers {
		handled = server.HandleConnectedPathReply(v)
		if handled {
			break
		}
	}
	if !handled {
		logger.WithFields(logger.Fields{"cmd": v.Command, "handle": v.Handle}).
			Error("HandleConnectedPathReply: Connection not found for this handle")
		SendInvalidHandle(m.serial, v.Handle)
	}
}

func (m *MultiServerApi) Stats() *EspApiStats {
	return m.espApiStats
}

func (m *MultiServerApi) PrintStats() {
	m.espApiStats.PrintStats()
}

type MultiServerApi struct {
	espApiStats *EspApiStats
	serial      *SerialConnection
	Servers     []*ServerApi
}

func NewMultiServerApi(serial *SerialConnection, graph *graph.GraphPath) *MultiServerApi {
	_allStats = NewEspApiStats()
	multisrv := MultiServerApi{serial: serial, espApiStats: _allStats}
	SendClearConnections(serial)
	inuse := graph.GetAllInUse()
	multisrv.serial.ConnPathFn = multisrv.HandleConnectedPathReply

	for _, address := range inuse {
		server, err := NewServerApi(serial, graph, MeshNodeId(address))
		if err != nil {
			log.Error(err)
		} else {
			multisrv.Servers = append(multisrv.Servers, server)
		}
	}
	return &multisrv
}
