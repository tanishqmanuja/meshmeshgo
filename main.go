package main

import (
	"fmt"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

var debugNodeId int = 0

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

	_debugNodeId, err := parseNodeId(config.DebugNodeAddr)
	if err == nil {
		debugNodeId = int(_debugNodeId)
		log.WithFields(logrus.Fields{"id": debugNodeId}).Info("Enabling debug of node")
	}

	graph, err := NewGraphPathFromFile("meshmesh.graphml", int64(serialPort.LocalNode))
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
	} else if config.Discovery {
		err = DoDiscovery(serialPort)
		if err != nil {
			log.WithField("err", err).Error("Error during discovery")
			os.Exit(-1)
		}

		os.Exit(0)
	}

	if !graph.NodeExists(graph.SourceNode) {
		log.WithField("node", graph.SourceNode).Fatal("Local node does not exists in grpah")
	}

	fmt.Println("Coordinator node is " + FmtNodeId(MeshNodeId(graph.SourceNode)))

	fmt.Println("|----------|----------------|--------------------|------|--------------------------------------------------|------|")
	fmt.Println("| Node Id  | Node Address   | Node Tag           | Port | Path                                             | Wei. |")
	fmt.Println("|----------|----------------|--------------------|------|--------------------------------------------------|------|")

	inuse := graph.GetAllInUse()
	for _, d := range inuse {
		nid := MeshNodeId(d)

		var _path string
		path, weight, err := graph.GetPath(d)
		if err == nil {
			for _, p := range path {
				if len(_path) > 0 {
					_path += " > "
				}
				_path += FmtNodeId(MeshNodeId(p))
			}
		}

		fmt.Printf("| %s | %14s | %-18s | %4d | %-48s | %3.2f |\n", FmtNodeId(nid), FmtNodeIdHass(nid), graph.NodeTag(d), 6053, _path, weight)
		go ListenToApiConnetions(serialPort, graph, FmtNodeIdHass(nid), 6053, nid)
	}

	fmt.Println("|----------|----------------|--------------------|------|--------------------------------------------------|------|")
	fmt.Println("")

	var lastStatsTime time.Time
	for {
		time.Sleep(1 * time.Second)
		if !serialPort.connected {
			break
		}
		if time.Since(lastStatsTime) > 1*time.Minute {
			lastStatsTime = time.Now()
			PrintStats()
		}
	}
}
