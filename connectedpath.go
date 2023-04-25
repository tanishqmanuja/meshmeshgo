package main

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
)

const connectedPathOpenConnectionRequest uint8 = 1
const connectedPathInvalidHandleReply uint8 = 4
const connectedPathSendDataRequest uint8 = 5
const connectedPathOpenConnectionAck uint8 = 6
const connectedPathOpenConnectionNack uint8 = 7
const CONNPATH_DISCONNECT_REQ uint8 = 8
const connectedPathSendDataError uint8 = 9
const connectedPathClearConnections uint8 = 10

type ConnPathConnection struct {
	connected bool
	serial    *SerialConnection
	handle    uint16
	sequence  uint16
	graph     *GraphPath
}

func parseAddress(address string) (uint32, error) {
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
		return (uint32(addr[1]) << 16) + (uint32(addr[2]) << 8) + uint32(addr[3]), nil
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

func (client *ConnPathConnection) OpenConnection(textaddr string, port uint16) error {

	addr, err := parseAddress(textaddr)
	if err != nil {
		return err
	}

	log.Printf("ConnPathConnection.OpenConnection %06X:%d", addr, port)

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

	i, err := client.serial.SendReceiveApi(ConnectedPathApiRequest2{
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

	if err != nil {
		return err
	}

	v, ok := i.(ConnectedPathApiReply)
	if !ok {
		return errors.New("invalid reply to connected path request")
	}

	if v.Handle == client.handle {
		if v.Command == connectedPathOpenConnectionAck {
			client.connected = true
			log.Printf("Connection accepted from remote party")
			return nil
		} else if v.Command == connectedPathOpenConnectionNack {
			return errors.New("open connection nack")
		} else {
			return errors.New("unwanted reply command")
		}
	} else {
		return fmt.Errorf("invalid handle in reply %d!=%d", v.Handle, client.handle)
	}
}

func NewConnPathConnection(serial *SerialConnection, graph *GraphPath) *ConnPathConnection {
	conn := &ConnPathConnection{serial: serial, handle: serial.GetNextHandle(), graph: graph}
	return conn

}
