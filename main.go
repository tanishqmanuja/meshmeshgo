package main

import (
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

var log = &logrus.Logger{
	Out:       os.Stderr,
	Formatter: &logrus.TextFormatter{DisableTimestamp: false, FullTimestamp: true, DisableColors: false},
	Hooks:     make(logrus.LevelHooks),
	Level:     logrus.WarnLevel,
}

func main() {
	config, err := NewConfig()
	if err != nil {
		log.Fatal("Invalid config options: ", err)
	}

	if config.WantHelp {
		os.Exit(0)
	}

	if config.VerboseLevel > 3 {
		log.SetLevel(logrus.TraceLevel)
	} else if config.VerboseLevel > 2 {
		log.SetLevel(logrus.DebugLevel)
	} else if config.VerboseLevel > 1 {
		log.SetLevel(logrus.InfoLevel)
	}

	log.WithFields(logrus.Fields{"portName": config.SerialPortName, "baudRate": config.SerialPortBaudRate}).
		Debug("Opening serial port")

	serialPort, err := NewSerial(config.SerialPortName, config.SerialPortBaudRate, false)
	if err != nil {
		log.Fatal("Serial port error: ", err)
	}

	graph, err := NewGraphPath("meshmesh.graphml", int64(serialPort.LocalNode))
	if err != nil {
		log.Fatal("GraphPath error: ", err)
	}

	if len(config.FirmwarePath) > 0 {
		if _, err := os.Stat(config.FirmwarePath); err != nil {
			log.WithField("err", err).Error("Check firmware file failed")
			os.Exit(-1)
		}

		err = UploadFirmware(MeshNodeId(config.TargetNode), config.FirmwarePath, serialPort)
		if err != nil {
			log.WithField("err", err).Error("Upload firmware failed")
			os.Exit(-1)
		}

		os.Exit(0)
	}

	go ListenToApiConnetions(serialPort, graph)

	for {
		time.Sleep(1 * time.Second)
		if !serialPort.connected {
			break
		}
	}
}
