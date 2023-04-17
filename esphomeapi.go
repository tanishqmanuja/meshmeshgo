package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
)

var allClients map[*ApiConnection]int

type ApiConnection struct {
	// incoming chan string
	handshakeDone bool
	handshakeProg bool
	outgoing      chan []byte
	reader        *bufio.Reader
	writer        *bufio.Writer
	inBuffer      *bytes.Buffer
	conn          net.Conn
	connection    *ApiConnection
}

func (client *ApiConnection) InitConnection(addr string, port int) error {
	fmt.Printf("InitConnection %s:%d\n", addr, port)
	return errors.New("invalid handshake")
}

func (client *ApiConnection) Forward(buffer []byte) {
}

func (client *ApiConnection) Close() {
	client.conn.Close()
	delete(allClients, client)
	if client.connection != nil {
		client.connection.connection = nil
	}

	log.Println("client closed.")
}

func (client *ApiConnection) Read() {
	for {

		var buffer = make([]byte, 1)
		_, err := client.conn.Read(buffer)

		if err == nil {
			if !client.handshakeDone {
				client.inBuffer.Write(buffer)

				if !client.handshakeProg {
					client.handshakeProg = true
				} else {
					if buffer[0] == '\n' {
						line := client.inBuffer.String()
						client.inBuffer.Reset()
						fields := strings.Split(strings.TrimSpace(line), "|")
						if len(fields) == 3 {
							port, err := strconv.Atoi(strings.TrimSpace(fields[1]))
							addr := strings.TrimSpace(fields[2])
							if err == nil {
								err = client.InitConnection(addr, port)
								if err != nil {
									break
								}
							} else {
								break
							}
						} else {
							break
						}
					}
				}

			} else {
				client.Forward(buffer)
			}

		} else {
			log.Println(err.Error())
			break
		}
	}

	client.Close()
	client = nil
}

/*func (client *ApiConnection) Write() {
	for data := range client.outgoing {
		client.writer.WriteString(data)
		client.writer.Flush()
	}
}*/

func (client *ApiConnection) Listen() {
	go client.Read()
	//go client.Write()
}

func NewApiConnection(connection net.Conn) *ApiConnection {
	//writer := bufio.NewWriter(connection)
	//reader := bufio.NewReader(connection)

	client := &ApiConnection{
		// incoming: make(chan string),
		handshakeDone: false,
		handshakeProg: false,
		outgoing:      make(chan []byte),
		conn:          connection,
		reader:        nil,
		writer:        nil,
		inBuffer:      bytes.NewBuffer([]byte{}),
	}
	client.Listen()

	return client
}

func mainTcp() {
	allClients = make(map[*ApiConnection]int)
	l, err := net.Listen("tcp4", ":6053")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer l.Close()

	for {
		c, err := l.Accept()
		if err != nil {
			log.Println(err)
			continue
		}

		client := NewApiConnection(c)
		for clientList := range allClients {
			if clientList.connection == nil {
				client.connection = clientList
				clientList.connection = client
				fmt.Println("Connected")
			}
		}
		allClients[client] = 1
		fmt.Println(len(allClients))
	}
}
