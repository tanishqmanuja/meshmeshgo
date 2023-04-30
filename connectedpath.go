package main

import (
	"errors"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

const connectedPathOpenConnectionRequest uint8 = 1
const connectedPathInvalidHandleReply uint8 = 4
const connectedPathSendDataRequest uint8 = 5
const connectedPathOpenConnectionAck uint8 = 6
const connectedPathOpenConnectionNack uint8 = 7
const CONNPATH_DISCONNECT_REQ uint8 = 8
const connectedPathSendDataError uint8 = 9
const connectedPathClearConnections uint8 = 10

const (
	connPathConnectionStateInit uint8 = iota
	connPathConnectionStateHandshakeStarted
	connPathConnectionStateHandshakeFailed
	connPathConnectionStateActive
	connPathConnectionStateInvalid
)

type ConnPathConnection struct {
	connState uint8
	serial    *SerialConnection
	handle    uint16
	sequence  uint16
	graph     *GraphPath
}

func ParseAddress(address string) (MeshNodeId, error) {
	fields := strings.Split(address, ".")
	if len(fields) != 4 {
		return 0, errors.New("invalid address string")
	}

	var err error
	addr := make([]byte, 4)
	for i, field := range fields {
		data, err := strconv.Atoi(field)
		if err != nil {
			break
		}
		addr[i] = byte(data)
	}

	if err != nil {
		return 0, err
	} else {
		return (MeshNodeId(addr[1]) << 16) + (MeshNodeId(addr[2]) << 8) + MeshNodeId(addr[3]), nil
	}
}

func (client *ConnPathConnection) getNextSequence() uint16 {
	client.sequence += 1
	return client.sequence
}

func (client *ConnPathConnection) SendData(data []byte) error {
	err := client.serial.SendApi(ConnectedPathApiRequest{
		Protocol: meshmeshProtocolConnectedPath,
		Command:  connectedPathSendDataRequest,
		Handle:   client.handle,
		Dummy:    0,
		Sequence: client.getNextSequence(),
		DataSize: uint16(len(data)),
		Data:     data,
	})

	return err
}

func ClearConnections(serial *SerialConnection) error {
	err := serial.SendApi(ConnectedPathApiRequest{
		Protocol: meshmeshProtocolConnectedPath,
		Command:  connectedPathClearConnections,
		Handle:   0,
		Dummy:    0,
		Sequence: 0,
		DataSize: 0,
		Data:     []byte{},
	})

	return err
}

func (client *ConnPathConnection) OpenConnectionAsync(textaddr string, port uint16) error {
	addr, err := ParseAddress(textaddr)
	if err != nil {
		return err
	}

	log.WithFields(logrus.Fields{"addr": FmtNodeId(MeshNodeId(addr)), "port": port, "handle": client.handle}).
		Debug("ConnPathConnection.OpenConnectionAsync")

	nodes, err := client.graph.GetPath(int64(addr))
	if err != nil {
		return err
	}

	if len(nodes) == 1 {
		return errors.New("speak with local node is not yet supported")
	}

	nodes = nodes[1:]
	path := make([]int32, len(nodes))
	for i, item := range nodes {
		path[i] = int32(item.ID())
	}

	client.connState = connPathConnectionStateHandshakeStarted
	err = client.serial.SendApi(ConnectedPathApiRequest2{
		Protocol: meshmeshProtocolConnectedPath,
		Command:  connectedPathOpenConnectionRequest,
		Handle:   client.handle,
		Dummy:    0,
		Sequence: client.getNextSequence(),
		DataSize: uint16(len(nodes)*4 + 3),
		Port:     port,
		PathLen:  uint8(len(nodes)),
		Path:     path,
	})

	return err
}

func (client *ConnPathConnection) handleIncomingOpenConnAck(v *ConnectedPathApiReply) {
	if client.connState != connPathConnectionStateHandshakeStarted {
		client.connState = connPathConnectionStateInvalid
		log.Error("handleIncomingOpenConnAck received while not in handshake state")
	} else {
		log.WithField("handle", client.handle).Debug("Accpeted connection")
		client.connState = connPathConnectionStateActive

	}
}

func (client *ConnPathConnection) handleIncomingOpenConnNack(v *ConnectedPathApiReply) {
	client.connState = connPathConnectionStateInvalid
}

func (client *ConnPathConnection) HandleIncomingReply(v *ConnectedPathApiReply) {
	log.WithFields(logrus.Fields{"handle": v.Handle, "reply": v.Command}).Debug("HandleIncomingReply")
	if v.Command == connectedPathOpenConnectionAck {
		client.handleIncomingOpenConnAck(v)
	} else if v.Command == connectedPathOpenConnectionNack {
		client.handleIncomingOpenConnNack(v)
	} else if v.Command == connectedPathSendDataError {
		log.WithField("handle", v.Handle).Error("HandleIncomingReply: SendDataError")
		client.connState = connPathConnectionStateInvalid
	} else if v.Command == connectedPathInvalidHandleReply {
		log.WithField("handle", v.Handle).Error("HandleIncomingReply: InvalidHandleReply")
		client.connState = connPathConnectionStateInvalid
	} else {
		log.WithFields(logrus.Fields{"handle": v.Handle, "reply": v.Command}).
			Error("HandleIncomingReply: unknow command reply received", v.Command, v.Handle)
	}
}

func NewConnPathConnection(serial *SerialConnection, graph *GraphPath) *ConnPathConnection {
	conn := &ConnPathConnection{
		serial:    serial,
		handle:    serial.GetNextHandle(),
		graph:     graph,
		connState: connPathConnectionStateInit}
	return conn

}
