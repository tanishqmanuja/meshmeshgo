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
	"leguru.net/m/v2/rest"
	"leguru.net/m/v2/rpc"
	"leguru.net/m/v2/utils"
)

const (
	programName        = "meshmeshgo"
	programDescription = "hub server for meshmesh network"
	graphFilename      = "meshmesh.graphml"
)

var (
	vcsHash  string
	vcsTime  time.Time
	vcsDirty bool
)

var quitProgram bool = false
var debugNodeId gra.NodeDevice

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
	if _, err := os.Stat(graphFilename); err == nil {
		network, err = gra.NewNeworkFromFile(graphFilename, int64(serialPort.LocalNode))
		if err != nil {
			logger.Log().Fatal("Graph read error: ", err)
		}
	} else {
		network = gra.NewNetwork(int64(serialPort.LocalNode))
		network.SaveToFile(graphFilename)
	}
	network.SetNetworkChangedCb(func() {
		network.SaveToFile(graphFilename)
	})

	if len(config.DebugNodeAddr) > 0 {
		_debugNodeId, err := gra.ParseDeviceId(config.DebugNodeAddr)
		if err != nil {
			logger.WithField("err", err).Fatal("Invalid debug node id")
		}
		debugNodeId, err = network.GetNodeDevice(int64(_debugNodeId))
		if err != nil {
			logger.WithField("id", _debugNodeId).Fatal("Debug node not found in graph")
		}
		logger.WithFields(logger.Fields{"id": debugNodeId}).Info("Enabling debug of node")
	}

	logger.Log().Info("Coordinator node is " + utils.FmtNodeId(network.LocalDeviceId()))
	gra.PrintTable(network)

	// Handle DiscAssociateReply received from other nodes
	serialPort.DiscAssociateFn = func(v *meshmesh.DiscAssociateApiReply) {
		logger.WithFields(logger.Fields{"server": utils.FmtNodeId(int64(v.Server)), "source": utils.FmtNodeId(int64(v.Source))}).Debug("DiscAssociateReply received")
		source, err := network.GetNodeDevice(int64(v.Source))
		if err != nil {
			source = gra.NewNodeDevice(int64(v.Source), true, "")
			network.AddNode(source)
		}
		for i := range 3 {
			if v.NodeId[i] > 0 {
				node, err := network.GetNodeDevice(int64(v.NodeId[i]))
				if err != nil {
					network.ChangeEdgeWeight(node.ID(), source.ID(), meshmesh.Rssi2weight(v.Rssi[i]), meshmesh.Rssi2weight(v.Rssi[i]))
					logger.WithFields(logger.Fields{"id": utils.FmtNodeId(int64(v.NodeId[i])), "rssi": v.Rssi[i]}).Debug("DiscAssociateReply received")
				}
			}
		}
		network.SaveToFile(graphFilename)
		serialPort.SendReceiveApiProt(meshmesh.NodeIdApiRequest{}, meshmesh.UnicastProtocol, meshmesh.MeshNodeId(source.ID()), nil)
		// ***** TODO: Update network graph with new node
	}

	// Initialize Esphome to HomeAssistant Server
	esphomeapi := meshmesh.NewMultiServerApi(serialPort, network)
	// Start RPC Server
	rpcServer := rpc.NewRpcServer(":50051")
	rpcServer.Start(fmt.Sprintf("%s - %s", programName, programDescription), fmt.Sprintf("%s - %s", vcsHash[:8], vcsTime.Format(time.RFC3339)), serialPort, network)
	defer rpcServer.Stop()

	// Start rest server
	restHandler := rest.NewHandler(serialPort, network, esphomeapi)
	rest.StartRestServer(rest.NewRouter(restHandler))

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
}
