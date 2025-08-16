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

var _allStats *EspApiStats

const (
	esphomeapiWaitPacketHead int = 0
	esphomeapiWaitPacketSize int = 1
	esphomeapiWaitPacketData int = 3
)

type ApiConnection struct {
	Stats           *EspApiConnectionStats
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
	debugThisNode   bool
	timeout         time.Time
	clientClosed    func(client *ApiConnection)
}

func (client *ApiConnection) forward(lastbyte byte) {
	client.inBuffer.WriteByte(lastbyte)
	switch client.inState {
	case esphomeapiWaitPacketHead:
		if lastbyte == 0x00 {
			client.inState = esphomeapiWaitPacketSize
		} else {
			client.inBuffer.Reset()
		}
	case esphomeapiWaitPacketSize:
		client.inAwaitSize = int(lastbyte) + 3
		client.inState = esphomeapiWaitPacketData
	default:
		if client.inBuffer.Len() == client.inAwaitSize {
			client.inState = esphomeapiWaitPacketHead
			logger.WithField("handle", client.connpath.handle).
				Trace(fmt.Sprintf("HA-->SE: %s", hex.EncodeToString(client.inBuffer.Bytes())))
			err := client.connpath.SendData(client.inBuffer.Bytes())
			client.Stats.SentBytes(client.inBuffer.Len())
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
		client.Stats.Start()
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
		client.Stats.GotHandle(client.connpath.handle)
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
	client.Stats.Stop()
	client.connpath.Disconnect()
	client.clientClosed(client)
}

func (client *ApiConnection) CheckTimeout() {
	for {
		if !client.socketOpen {
			break
		}
		if client.connpath.connState == connPathConnectionStateInit || client.connpath.connState == connPathConnectionStateHandshakeStarted {
			if time.Since(client.timeout).Milliseconds() > 3000 {
				logger.Error(fmt.Sprintf("Closing connection beacuse timeout after %dms in connPathConnectionStateInit for handle %d", time.Since(client.timeout).Milliseconds(), client.connpath.handle))
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
		client.Stats.ReceivedBytes(1)

		if err == nil {
			switch client.connpath.connState {
			case connPathConnectionStateHandshakeStarted:
				// FIXME check for if buffer grown outside limits
				client.tmpBuffer.WriteByte(buffer[0])
			case connPathConnectionStateActive:
				// FIXME handle error
				client.forward(buffer[0])
			default:
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
	logger.WithFields(logger.Fields{
		"handle": client.connpath.handle,
		"meshid": utils.FmtNodeId(int64(client.reqAddress)),
		"len":    len(data),
		"data":   utils.EncodeToHexEllipsis(data, 10),
	}).Trace("SE-->HA")
	n, err := client.socket.Write(data)
	if err != nil {
		return err
	}

	if n < len(data) {
		return errors.New("socket can't receive all bytes")
	}

	return nil
}

func NewApiConnection(connection net.Conn, serial *SerialConnection, addr MeshNodeId, closedCb func(*ApiConnection)) (*ApiConnection, error) {
	client := &ApiConnection{
		connpath:     NewConnPathConnection(serial),
		socketOpen:   true,
		socket:       connection,
		tmpBuffer:    bytes.NewBuffer([]byte{}),
		inBuffer:     bytes.NewBuffer([]byte{}),
		timeout:      time.Now(),
		clientClosed: closedCb,
		Stats:        _allStats.Stats(addr),
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

type ServerApi struct {
	Address       MeshNodeId
	Clients       []*ApiConnection
	listener      net.Listener
	listenAddress string
}

func (s *ServerApi) GetListenAddress() string {
	return s.listenAddress
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
	logger.WithFields(logger.Fields{"handle": client.connpath.handle}).Info("Closed EspHomeApi connection")
}

func (s *ServerApi) ListenAndServe(serial *SerialConnection) {
	for {
		c, err := s.listener.Accept()
		if err != nil {
			logger.Error(err)
			continue
		}

		logger.WithFields(logger.Fields{"nodeId": s.Address, "active": len(s.Clients)}).Debug("EspHome connection accepted")
		client, err := NewApiConnection(c, serial, s.Address, s.ClientClosedCb)
		if err != nil {
			logger.Error(err)
			c.Close()
		} else {
			s.Clients = append(s.Clients, client)
			logger.WithFields(logger.Fields{"nodeId": utils.FmtNodeId(int64(s.Address)), "clients": len(s.Clients)}).Debug("Added new client")
		}
	}
}

func (s *ServerApi) CloseConnections() {
	for _, client := range s.Clients {
		client.Close()
	}
}

func (s *ServerApi) ShutDown() {
	s.listener.Close()
}

func NewServerApi(serial *SerialConnection, address MeshNodeId, config *ServerApiConfig) (*ServerApi, error) {
	var bindAddress string = config.BindAddress
	if config.BindAddress == "" || config.BindAddress == "dynamic" {
		bindAddress = utils.FmtNodeIdHass(int64(address))
	}

	bindPort := config.BindPort
	if config.BindPort <= 0 {
		bindPort = utils.HashString(utils.FmtNodeId(int64(address)), config.SizeOfPortsPool) + config.BasePortOffset
	}

	server := ServerApi{Address: address}
	server.listenAddress = fmt.Sprintf("%s:%d", bindAddress, bindPort)
	listener, err := net.Listen("tcp4", server.listenAddress)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	logger.WithFields(logger.Fields{"node": utils.FmtNodeId(int64(address)), "bind": server.listenAddress}).Debug("Start listening on port for node connection")
	server.listener = listener
	go server.ListenAndServe(serial)
	return &server, nil
}

func (m *MultiServerApi) handleUnhandledReply(v *ConnectedPathApiReply) {
	logger.WithFields(logger.Fields{"cmd": v.Command, "handle": v.Handle}).
		Error("handleUnhandledReply: Connection not found for this handle")
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
		m.handleUnhandledReply(v)
	}
}

func (m *MultiServerApi) Stats() *EspApiStats {
	return m.espApiStats
}

func (m *MultiServerApi) PrintStats() {
	m.espApiStats.PrintStats()
}

func (m *MultiServerApi) CloseConnection(addr MeshNodeId) {
	for _, server := range m.Servers {
		if server.Address == addr {
			server.CloseConnections()
		}
	}
}

func (m *MultiServerApi) MainNetworkChanged() {
	nodes := graph.GetMainNetwork().Nodes()
	for nodes.Next() {
		node := nodes.Node().(graph.NodeDevice)
		if node.Device().InUse() {
			found := false
			for _, server := range m.Servers {
				if server.Address == MeshNodeId(node.ID()) {
					found = true
					break
				}
			}
			if !found {
				logger.WithFields(logger.Fields{"node": utils.FmtNodeId(int64(node.ID()))}).Debug("MainNetworkChanged adding esphome connection to new node")
				server, err := NewServerApi(m.serial, MeshNodeId(node.ID()), &m.config)
				if err != nil {
					log.Error(err)
				} else {
					m.Servers = append(m.Servers, server)
				}
			}
		}
	}

	newServers := make([]*ServerApi, 0)
	for _, server := range m.Servers {
		found := false
		nodes = graph.GetMainNetwork().Nodes()
		for nodes.Next() {
			node := nodes.Node().(graph.NodeDevice)
			if server.Address == MeshNodeId(node.ID()) {
				found = true
				break
			}
		}
		if !found {
			logger.WithFields(logger.Fields{"server": server.Address}).Debug("MainNetworkChanged deleting esphome connection to non existing node")
			server.CloseConnections()
		} else {
			newServers = append(newServers, server)
		}
	}
	m.Servers = newServers
}

type ServerApiConfig struct {
	BindAddress     string
	BindPort        int
	BasePortOffset  int
	SizeOfPortsPool int
}

type MultiServerApi struct {
	espApiStats *EspApiStats
	serial      *SerialConnection
	config      ServerApiConfig
	Servers     []*ServerApi
}

func NewMultiServerApi(serial *SerialConnection, config ServerApiConfig) *MultiServerApi {
	_allStats = NewEspApiStats()
	multisrv := MultiServerApi{serial: serial, espApiStats: _allStats, config: config}
	SendClearConnections(serial)
	multisrv.serial.ConnPathFn = multisrv.HandleConnectedPathReply

	nodes := graph.GetMainNetwork().Nodes()
	graph.AddMainNetworkChangedCallback(multisrv.MainNetworkChanged)

	for nodes.Next() {
		node := nodes.Node().(graph.NodeDevice)
		if node.Device().InUse() {
			server, err := NewServerApi(serial, MeshNodeId(node.ID()), &config)
			if err != nil {
				log.Error(err)
			} else {
				multisrv.Servers = append(multisrv.Servers, server)
			}
		}
	}
	return &multisrv
}
