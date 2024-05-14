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

	go ListenToApiConnetions(serialPort, graph, 6053)

	fmt.Println("Coordinator node is " + FmtNodeId(MeshNodeId(graph.SourceNode)))

	fmt.Println("|----------|----------------|--------------------|------|--------------------------------------------------|------|")
	fmt.Println("| Node Id  | Node Address   | Node Tag           | Port | Path                                             | Wei. |")
	fmt.Println("|----------|----------------|--------------------|------|--------------------------------------------------|------|")

	changed := false
	lastPort := int16(6054)
	directs := graph.GetAllDirectId()
	for _, d := range directs {
		nid := MeshNodeId(d)
		port := graph.NodeDirectPort(d)
		if port == 0 {
			port = lastPort
			lastPort += 1
			graph.SetNodeDirectPort(d, port)
			changed = true
		} else if port > 0 {
			if port > lastPort {
				lastPort = port + 1
			}
		}

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

		fmt.Printf("| %s | %14s | %-18s | %4d | %-48s | %3.2f |\n", FmtNodeId(nid), FmtNodeIdHass(nid), graph.NodeTag(d), port, _path, weight)
		go ListenToDirectApiConnetions(serialPort, graph, int(port), FmtNodeIdHass(nid))
	}

	fmt.Println("|----------|----------------|--------------------|------|--------------------------------------------------|------|")
	fmt.Println("")

	if changed {
		graph.writeGraphXml("meshmesh.graphml")
	}

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
