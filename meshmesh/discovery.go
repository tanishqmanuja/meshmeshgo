package meshmesh

import (
	"errors"
	"math"
	"time"

	"leguru.net/m/v2/logger"

	gra "leguru.net/m/v2/graph"
	"leguru.net/m/v2/utils"
)

type DiscoveryProcedure struct {
	serial        *SerialConnection
	network       *gra.Network
	currentDevice *gra.Device
	Neighbors     map[int64]discWeights
}

func (d *DiscoveryProcedure) CurrentNode() int64 {
	if d.currentDevice == nil {
		return 0
	}
	return d.currentDevice.ID()
}

type discWeights struct {
	Current float64
	Next    float64
}

func _rssi2weight(rssi int16) float64 {
	if rssi <= 0 {
		rssi *= -2
	} else if rssi > 44 {
		rssi = 44
	}
	return 1.0 - float64(rssi)/45.0
}

func neighborsFromGraph(g *gra.Network, n *gra.Device, w map[int64]discWeights) error {
	neighbors := g.From(n.ID())
	for neighbors.Next() {
		neighbor := neighbors.Node().(*gra.Device)
		weightTo, ok := g.Weight(n.ID(), neighbor.ID())
		if !ok {
			return errors.New("corrupted graph")
		}
		weightFrom, ok := g.Weight(neighbor.ID(), n.ID())
		if !ok {
			weightFrom = weightTo
			logger.WithFields(logger.Fields{"from": gra.FmtDeviceId(n), "to": gra.FmtDeviceId(neighbor), "weightTo": weightTo, "weightFrom": weightFrom}).Warn("Missing return edge")
			//return errors.New("corrupted graph")
		}
		w[neighbor.ID()] = discWeights{Current: math.Min(weightTo, weightFrom), Next: 1.0}
	}
	return nil
}

func neighborsAdavance(w map[int64]discWeights) {
	for i, d := range w {
		w[i] = discWeights{Current: d.Next, Next: 1.0}
	}
}

func neighborsToGraph(g *gra.Network, n *gra.Device, w map[int64]discWeights) {
	nodes := g.From(n.ID())

	for nodes.Next() {
		neighbor := nodes.Node()
		g.RemoveEdge(n.ID(), neighbor.ID())
		g.RemoveEdge(neighbor.ID(), n.ID())
	}

	for id, d := range w {
		logger.WithFields(logger.Fields{"id": id, "weight": d, "exists": g.NodeIdExists(id)}).Info("neighbor to graph")
		g.ChangeEdgeWeight(n.ID(), id, d.Next, d.Next)
	}
}

func _updateNeighbor(w map[int64]discWeights, id int64, rssi1, rssi2 float64) error {
	if _, exists := w[id]; exists {
		w[id] = discWeights{Current: w[id].Current, Next: math.Min(rssi1, rssi2)}
	} else {
		w[id] = discWeights{Current: 1.0, Next: math.Min(rssi1, rssi2)}
	}
	return nil
}

func _findNextNode(g *gra.Network) *gra.Device {
	nodes := g.Nodes()
	var found_node *gra.Device
	var found_weight float64 = 1e9
	for nodes.Next() {
		dev := nodes.Node().(*gra.Device)
		if dev.InUse() && !dev.Discovered() && dev.Seen() {
			path, weight, err := g.GetPath(dev)
			if weight < found_weight {
				found_weight = weight
				found_node = dev
			}
			logger.Log().Println("path", path, weight, err)
		}
	}
	return found_node
}

func (d *DiscoveryProcedure) Init() error {
	if d.network == nil {
		d.network = gra.NewNetwork(int64(d.serial.LocalNode))
	}
	if d.currentDevice == nil {
		d.currentDevice = _findNextNode(d.network)
	}
	if d.currentDevice == nil {
		d.Clear()
		d.currentDevice = _findNextNode(d.network)
	}
	if d.currentDevice == nil {
		return errors.New("no nodes to discover")
	}
	d.Neighbors = make(map[int64]discWeights)
	neighborsFromGraph(d.network, d.currentDevice, d.Neighbors)
	return nil
}

func (d *DiscoveryProcedure) Step() error {
	protocol := DirectProtocol
	if d.currentDevice.ID() != d.network.LocalDevice().ID() {
		protocol = UnicastProtocol
	}

	logger.Log().Printf("Start dicover of node %06X", d.currentDevice.ID())

	_, err := d.serial.SendReceiveApiProt(DiscResetTableApiRequest{}, protocol, MeshNodeId(d.currentDevice.ID()))
	if err != nil {
		return err
	}

	_, err = d.serial.SendReceiveApiProt(DiscStartDiscoverApiRequest{Mask: 0, Filter: 0, Slotnum: 100}, protocol, MeshNodeId(d.currentDevice.ID()))
	if err != nil {
		return err
	}

	time.Sleep(3 * time.Second)

	reply1, err := d.serial.SendReceiveApiProt(DiscTableSizeApiRequest{}, protocol, MeshNodeId(d.currentDevice.ID()))
	if err != nil {
		return err
	}
	tableSize, ok := reply1.(DiscTableSizeApiReply)
	if !ok {
		return errors.New("comunication error")
	}

	for i := uint8(0); i < tableSize.Size; i++ {

		reply1, err = d.serial.SendReceiveApiProt(DiscTableItemGetApiRequest{Index: i}, protocol, MeshNodeId(d.currentDevice.ID()))
		if err != nil {
			return err
		}
		tableItem, ok := reply1.(DiscTableItemGetApiReply)
		if !ok {
			return errors.New("comunication error")
		}

		logger.Log().Printf("Query of row %d node %s rssi1 %d rssi2 %d", i, utils.FmtNodeId(int64(tableItem.NodeId)), tableItem.Rssi1, tableItem.Rssi2)
		_updateNeighbor(d.Neighbors, int64(tableItem.NodeId), _rssi2weight(tableItem.Rssi1), _rssi2weight(tableItem.Rssi2))
	}

	return err
}

func (d *DiscoveryProcedure) Clear() {
	d.network.SetAllNodesUnseen()
	d.network.LocalDevice().SetSeen(true)
}

func (d *DiscoveryProcedure) Save() error {
	if d.currentDevice.ID() == -1 {
		return errors.New("discovery is inactive")
	}
	d.currentDevice.SetDiscovered(true)
	neighborsToGraph(d.network, d.currentDevice, d.Neighbors)
	d.network.SaveToFile("discovery.graphml")
	neighborsAdavance(d.Neighbors)
	return nil

}

func NewDiscoveryProcedure(serial *SerialConnection, network *gra.Network, nodeid int64) *DiscoveryProcedure {
	var currentDevice *gra.Device
	if network != nil {
		currentDevice = network.GetDevice(nodeid)
	}
	return &DiscoveryProcedure{serial: serial, network: network, currentDevice: currentDevice}
}
