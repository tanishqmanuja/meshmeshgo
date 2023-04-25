package main

import (
	"fmt"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

var log = &logrus.Logger{
	Out:       os.Stderr,
	Formatter: &logrus.TextFormatter{FullTimestamp: false, DisableColors: false},
	Hooks:     make(logrus.LevelHooks),
	Level:     logrus.DebugLevel,
}

func main() {
	config, err := NewConfig()
	if err != nil {
		log.Fatal("Invalid config options: ", err)
	}

	if config.WantHelp {
		os.Exit(0)
	}

	serialPort, err := NewSerial(config.SerialPortName, config.SerialPortBaudRate, false)
	if err != nil {
		log.Fatal("Serial port error: ", err)
	}

	log.WithFields(logrus.Fields{
		"nodeId":   fmt.Sprintf("0x%06X", serialPort.LocalNode),
		"portName": config.SerialPortName,
		"baudRate": config.SerialPortBaudRate,
	}).Info("Valid local node found")

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
