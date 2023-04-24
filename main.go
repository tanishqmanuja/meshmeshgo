package main

import (
	"log"
	"time"
)

func main() {
	const portName string = "/dev/ttyUSB0"
	const baudRate int = 460800

	serialPort, err := NewSerial(portName, baudRate)
	if err != nil {
		log.Fatal("Serial port error: ", err)
	}

	log.Printf("Valid local node found 0x%06X in %s@%d", serialPort.LocalNode, portName, baudRate)

	graph, err := NewGraphPath("meshmesh.graphml", int64(serialPort.LocalNode))
	if err != nil {
		log.Fatal("GraphPath error: ", err)
	}

	go ListenToApiConnetions(serialPort, graph)

	for {
		time.Sleep(1 * time.Second)
		if !serialPort.connected {
			break
		}
	}
}
