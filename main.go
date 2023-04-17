package main

import (
	"log"
	"time"
)

func main() {
	go mainTcp()

	_, err := NewGraphPath("meshmesh.graphml")
	if err != nil {
		log.Fatal("GraphPath error: ", err)
	}

	const portName string = "/dev/ttyUSB0"
	const baudRate int = 460800

	serialPort, err := NewSerial(portName, baudRate)
	if err != nil {
		log.Fatal("Serial port error: ", err)
	}

	log.Printf("Valid local node found %s@%d", portName, baudRate)

	for {
		time.Sleep(1 * time.Second)
		if !serialPort.connected {
			break
		}
	}
}
