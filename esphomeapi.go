package main

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
)

var allClients map[*ApiConnection]int

const esphomeapiWaitPacketHead int = 0
const esphomeapiWaitPacketSize int = 1
const esphomeapiWaitPacketData int = 3

type ApiConnection struct {
	handshakeDone bool
	inState       int
	inAwaitSize   int
	inBuffer      *bytes.Buffer
	conn          net.Conn
	connpath      *ConnPathConnection
}

func (client *ApiConnection) forward(lastbyte byte) {
	if client.inState == esphomeapiWaitPacketHead {
		if lastbyte == 0x00 {
			client.inState = esphomeapiWaitPacketSize
		}
	} else if client.inState == esphomeapiWaitPacketSize {
		client.inAwaitSize = int(lastbyte) + 3
		client.inState = esphomeapiWaitPacketData
	} else {
		if client.inBuffer.Len() == client.inAwaitSize {
			client.inState = esphomeapiWaitPacketHead
			if client.connpath.serial.debug {
				log.Println("HA: ", hex.EncodeToString(client.inBuffer.Bytes()))
			}
			err := client.connpath.SendData(client.inBuffer.Bytes())
			if err != nil {
				log.Printf("Warning: %s", err.Error())
			}
			client.inBuffer.Reset()
		}
	}
}

func (client *ApiConnection) handshake() error {
	line := strings.TrimSpace(client.inBuffer.String())
	client.inBuffer.Reset()

	log.Println("ApiConnection.handshake", line)
	fields := strings.Split(strings.TrimSpace(line), "|")
	if len(fields) == 3 {
		var port int
		addr := strings.TrimSpace(fields[1])
		port, err := strconv.Atoi(strings.TrimSpace(fields[2]))
		if err == nil {
			err = client.connpath.OpenConnection(addr, uint16(port))
			if err != nil {
				client.conn.Write([]byte{'!', '!', 'K', 'O', '!'})
				log.Printf("ApiConnection.handshake OpenConnection failed")
				return err
			} else {
				client.handshakeDone = true
				log.Printf("ApiConnection.handshake OpenConnection succesfull %s:%d with handle %d", addr, port, client.connpath.handle)
				client.conn.Write([]byte{'!', '!', 'O', 'K', '!'})
			}
		} else {
			return err
		}
	} else {
		return errors.New("wrong hadshake header")
	}

	return nil
}

func (client *ApiConnection) Close() {
	client.conn.Close()
	delete(allClients, client)
	log.Printf("ApiConnection.Close remaining %d active connections", len(allClients))
}

func (client *ApiConnection) Read() {
	var err error

	for {
		var buffer = make([]byte, 1)
		_, err = client.conn.Read(buffer)
		if err == nil {
			client.inBuffer.WriteByte(buffer[0])
			// FIXME check for if buffer grown outside limits
			if client.handshakeDone {
				client.forward(buffer[0])
			} else {
				if buffer[0] == '\n' {
					err = client.handshake()
					client.inState = esphomeapiWaitPacketHead
					if err != nil {
						break
					}
				}
			}
		} else {
			break
		}
	}

	if err != nil {
		log.Printf("ApiConnection.Read closing connection with error: %s", err)
	}

	client.Close()
}

func (client *ApiConnection) ForwardData(data []byte) error {
	n, err := client.conn.Write(data)
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
		handshakeDone: false,
		connpath:      NewConnPathConnection(serial, graph),
		conn:          connection,
		inBuffer:      bytes.NewBuffer([]byte{}),
	}

	go client.Read()

	return client
}

func HandleConnectedPathReply(v *ConnectedPathApiReply) {
	/*log.Printf("NODE: received cmd:%d handle:%d size:%d %s", v.Command, v.Handle, len(v.Data), hex.EncodeToString(v.Data))
	if v.Command == 5 && len(v.Data) >= 2 {
		msgsize := v.Data[1]
		msgtype := v.Data[2]
		log.Printf("      proto size:%d type:%d %s", msgsize, msgtype, hex.EncodeToString(v.Data[3:]))
	}*/
	if len(v.Data) > 0 {
		var handled bool = false
		for client := range allClients {
			if client.connpath.handle == v.Handle {
				handled = true
				if v.Command == connectedPathSendDataRequest {
					err := client.ForwardData(v.Data)
					if err != nil {
						client.Close()
					}
				} else if v.Command == connectedPathSendDataError {
					log.Printf("HandleConnectedPathReply: SendDataError on handle %d", v.Handle)
				} else if v.Command == connectedPathInvalidHandleReply {
					log.Printf("HandleConnectedPathReply: InvalidHandleReply on handle %d", v.Handle)
					client.Close()
				} else if v.Command == connectedPathOpenConnectionAck || v.Command == connectedPathOpenConnectionNack {
					log.Printf("Unscheduled OpenConnectionAck/Nack received %d wuth handle %d", v.Command, v.Handle)
				} else {
					log.Printf("HandleConnectedPathReply: unknow command received cmd %d handle %d", v.Command, v.Handle)
				}
			}
		}
		if !handled {
			log.Printf("HandleConnectedPathReply: Warning cmd %d with handle %d is not handled", v.Command, v.Handle)
		}
	}
}

func ListenToApiConnetions(serial *SerialConnection, graph *GraphPath) {
	serial.ConnPathFn = HandleConnectedPathReply
	allClients = make(map[*ApiConnection]int)
	l, err := net.Listen("tcp4", ":6053")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer l.Close()

	ClearConnections(serial)

	for {
		c, err := l.Accept()
		if err != nil {
			log.Println(err)
			continue
		}

		client := NewApiConnection(c, serial, graph)
		allClients[client] = 1
		log.Printf("ListenToApiConnetions: connection added, %d active connections", len(allClients))
	}
}
