package main

import (
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"leguru.net/m/v2/graph"
	gra "leguru.net/m/v2/graph"
	log "leguru.net/m/v2/logger"
	"leguru.net/m/v2/meshmesh"
	"leguru.net/m/v2/tui"
	"leguru.net/m/v2/utils"
)

var debugNodeId int = 0

func main() {
	config, err := NewConfig()
	if err != nil {
		log.Log().Fatal("Invalid config options: ", err)
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

	log.WithFields(log.Fields{"portName": config.SerialPortName, "baudRate": config.SerialPortBaudRate}).Debug("Opening serial port")
	serialPort, err := meshmesh.NewSerial(config.SerialPortName, config.SerialPortBaudRate, false)
	if err != nil {
		log.Log().Fatal("Serial port error: ", err)
	}

	_debugNodeId, err := gra.ParseNodeIdForGrpah(config.DebugNodeAddr)
	if err == nil {
		debugNodeId = int(_debugNodeId)
		log.Log().WithFields(logrus.Fields{"id": debugNodeId}).Info("Enabling debug of node")
	}

	graphpath, err := gra.NewGraphPathFromFile("meshmesh.graphml", int64(serialPort.LocalNode))
	if err != nil {
		log.Log().Fatal("GraphPath error: ", err)
	}

	if len(config.FirmwarePath) > 0 {
		if _, err := os.Stat(config.FirmwarePath); err != nil {
			log.Log().WithField("err", err).Error("Check firmware file failed")
			os.Exit(-1)
		}

		err = meshmesh.UploadFirmware(meshmesh.MeshNodeId(config.TargetNode), config.FirmwarePath, serialPort)
		if err != nil {
			log.Log().WithField("err", err).Error("Upload firmware failed")
			os.Exit(-1)
		}

		os.Exit(0)
	} else if config.Discovery {
		err = meshmesh.DoDiscovery(serialPort)
		if err != nil {
			log.Log().WithField("err", err).Error("Error during discovery")
			os.Exit(-1)
		}

		os.Exit(0)
	}

	if !graphpath.NodeExists(graphpath.SourceNode) {
		log.Log().WithField("node", graphpath.SourceNode).Fatal("Local node does not exists in grpah")
	}

	log.Log().Info("Coordinator node is " + utils.FmtNodeId(graphpath.SourceNode))
	graph.PrintTable(graphpath)

	inuse := graphpath.GetAllInUse()
	for _, nid := range inuse {
		go meshmesh.ListenToApiConnetions(serialPort, graphpath, utils.FmtNodeIdHass(nid), 6053, meshmesh.MeshNodeId(nid))
	}

	go tui.TuiStart("0.0.0.0", "2024", graphpath, serialPort)
	var lastStatsTime time.Time
	for {
		time.Sleep(1 * time.Second)
		if !serialPort.IsConnected() {
			break
		}
		if time.Since(lastStatsTime) > 1*time.Minute {
			lastStatsTime = time.Now()
			meshmesh.PrintStats()
		}
	}
}
