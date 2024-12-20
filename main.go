package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"leguru.net/m/v2/graph"
	gra "leguru.net/m/v2/graph"
	"leguru.net/m/v2/logger"
	"leguru.net/m/v2/meshmesh"
	"leguru.net/m/v2/tui"
	"leguru.net/m/v2/utils"
)

var quitProgram bool = false
var debugNodeId int = 0

func waitForTermination() {
	terminationRequested := make(chan os.Signal, 1)
	signal.Notify(terminationRequested, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-terminationRequested
	logger.Log().Info("Program termination requested")
	quitProgram = true
}

func main() {
	go waitForTermination()

	config, err := NewConfig()
	if err != nil {
		logger.Log().Fatal("Invalid config options: ", err)
	}

	if config.WantHelp {
		os.Exit(0)
	}

	if config.VerboseLevel > 3 {
		logger.SetLevel(logrus.TraceLevel)
	} else if config.VerboseLevel > 2 {
		logger.SetLevel(logrus.DebugLevel)
	} else if config.VerboseLevel > 1 {
		logger.SetLevel(logrus.InfoLevel)
	}

	logger.WithFields(logger.Fields{"portName": config.SerialPortName, "baudRate": config.SerialPortBaudRate}).Debug("Opening serial port")
	serialPort, err := meshmesh.NewSerial(config.SerialPortName, config.SerialPortBaudRate, false)
	if err != nil {
		logger.Log().Fatal("Serial port error: ", err)
	}

	_debugNodeId, err := gra.ParseNodeIdForGrpah(config.DebugNodeAddr)
	if err == nil {
		debugNodeId = int(_debugNodeId)
		logger.WithFields(logger.Fields{"id": debugNodeId}).Info("Enabling debug of node")
	}

	graphpath, err := gra.NewGraphPathFromFile("meshmesh.graphml", int64(serialPort.LocalNode))
	if err != nil {
		logger.Log().Fatal("GraphPath error: ", err)
	}

	if len(config.FirmwarePath) > 0 {
		if _, err := os.Stat(config.FirmwarePath); err != nil {
			logger.Log().WithField("err", err).Error("Check firmware file failed")
			os.Exit(-1)
		}

		err = meshmesh.UploadFirmware(meshmesh.MeshNodeId(config.TargetNode), config.FirmwarePath, serialPort)
		if err != nil {
			logger.Log().WithField("err", err).Error("Upload firmware failed")
			os.Exit(-1)
		}

		os.Exit(0)
	} else if config.Discovery {
		err = meshmesh.DoDiscovery(serialPort)
		if err != nil {
			logger.Log().WithField("err", err).Error("Error during discovery")
			os.Exit(-1)
		}

		os.Exit(0)
	}

	if !graphpath.NodeExists(graphpath.SourceNode) {
		logger.Log().WithField("node", graphpath.SourceNode).Fatal("Local node does not exists in grpah")
	}

	logger.Log().Info("Coordinator node is " + utils.FmtNodeId(graphpath.SourceNode))
	graph.PrintTable(graphpath)

	esphomeapi := meshmesh.NewMultiServerApi(serialPort, graphpath)

	sshsrv, err := tui.NewSshServer("0.0.0.0", "2024", graphpath, serialPort, esphomeapi)
	if err != nil {
		logger.Error(err)
	}

	var lastStatsTime time.Time
	for {
		time.Sleep(1 * time.Second)
		if quitProgram {
			break
		}
		if !serialPort.IsConnected() {
			break
		}
		if time.Since(lastStatsTime) > 1*time.Minute {
			lastStatsTime = time.Now()
			esphomeapi.PrintStats()
		}
	}

	tui.ShutdownSshServer(sshsrv)
}
