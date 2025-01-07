package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"leguru.net/m/v2/config"
	gra "leguru.net/m/v2/graph"
	"leguru.net/m/v2/logger"
	"leguru.net/m/v2/meshmesh"
	"leguru.net/m/v2/rpc"
	"leguru.net/m/v2/tui"
)

const (
	programName        = "meshmeshgo"
	programDescription = "hub server for meshmesh network"
)

var (
	vcsHash  string
	vcsTime  time.Time
	vcsDirty bool
)

var quitProgram bool = false
var debugNodeId *gra.Device

func waitForTermination() {
	terminationRequested := make(chan os.Signal, 1)
	signal.Notify(terminationRequested, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-terminationRequested
	logger.Info("Program termination requested")
	quitProgram = true
}

func getBuildInfo() {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		logger.Fatal("Failed to read build info")
	}

	for _, kv := range bi.Settings {
		switch kv.Key {
		case "vcs.revision":
			vcsHash = kv.Value
		case "vcs.time":
			vcsTime, _ = time.Parse(time.RFC3339, kv.Value)
		case "vcs.modified":
			vcsDirty = kv.Value == "true"
		}
	}
}

func initConfig() *config.Config {
	config.InitINIConfig()
	c, err := config.NewConfig()
	if err != nil {
		logger.Fatal("Invalid config options: ", err)
	}

	if c.WantHelp {
		os.Exit(0)
	}

	if c.VerboseLevel > 3 {
		logger.SetLevel(logrus.TraceLevel)
	} else if c.VerboseLevel > 2 {
		logger.SetLevel(logrus.DebugLevel)
	} else if c.VerboseLevel > 1 {
		logger.SetLevel(logrus.InfoLevel)
	}

	return c
}

func main() {
	getBuildInfo()
	go waitForTermination()
	config := initConfig()

	logger.WithFields(logger.Fields{"programName": programName, "vcsHash": vcsHash, "vcsTime": vcsTime, "vcsDirty": vcsDirty}).Info("Starting program")
	logger.Info(programDescription)

	logger.WithFields(logger.Fields{"portName": config.SerialPortName, "baudRate": config.SerialPortBaudRate}).Debug("Opening serial port")
	serialPort, err := meshmesh.NewSerial(config.SerialPortName, config.SerialPortBaudRate, false)
	if err != nil {
		logger.Log().Fatal("Serial port error: ", err)
	}

	var network *gra.Network
	if _, err := os.Stat("meshmesh.graphml"); err == nil {
		network, err = gra.NewNeworkFromFile("meshmesh.graphml", int64(serialPort.LocalNode))
		if err != nil {
			logger.Log().Fatal("Graph read error: ", err)
		}
	} else {
		network = gra.NewNetwork(int64(serialPort.LocalNode))
		network.SaveToFile("meshmesh.graphml")
	}

	if len(config.DebugNodeAddr) > 0 {
		_debugNodeId, err := gra.ParseDeviceId(config.DebugNodeAddr)
		if err != nil {
			logger.WithField("err", err).Fatal("Invalid debug node id")
		}
		debugNodeId = network.Node(_debugNodeId).(*gra.Device)
		if debugNodeId == nil {
			logger.WithField("id", _debugNodeId).Fatal("Debug node not found in graph")
		}
		logger.WithFields(logger.Fields{"id": debugNodeId}).Info("Enabling debug of node")
	}

	logger.Log().Info("Coordinator node is " + gra.FmtDeviceId(network.LocalDevice()))
	meshmesh.SetNetworkGraph(network)
	gra.PrintTable(network)
	// Initialize Esphome to HomeAssistant Server
	esphomeapi := meshmesh.NewMultiServerApi(serialPort, network)
	// Initialize SSH Server
	sshsrv, err := tui.NewSshServer("0.0.0.0", "2024", network, serialPort, esphomeapi)
	if err != nil {
		logger.Error(err)
	}
	// Start RPC Server
	rpcServer := rpc.NewRpcServer(":50051")
	rpcServer.Start(fmt.Sprintf("%s - %s", programName, programDescription), fmt.Sprintf("%s - %s", vcsHash[:8], vcsTime.Format(time.RFC3339)), serialPort, network)
	defer rpcServer.Stop()

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
