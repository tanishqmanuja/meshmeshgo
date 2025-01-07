package graph

import (
	"fmt"
	"math"

	"github.com/sirupsen/logrus"
	"gonum.org/v1/gonum/graph/path"
	"gonum.org/v1/gonum/graph/simple"
	"leguru.net/m/v2/logger"
)

type Device struct {
	id         int64
	inuse      bool
	discovered bool
	seen       bool
	tag        string
	address    string
}

func (d Device) ID() int64 {
	return d.id
}

func (d Device) InUse() bool {
	return d.inuse
}

func (d Device) Discovered() bool {
	return d.discovered
}

func (d *Device) SetDiscovered(discovered bool) {
	d.discovered = discovered
}

func (d Device) Seen() bool {
	return d.seen
}

func (d *Device) SetSeen(seen bool) {
	d.seen = seen
}

func (d Device) Tag() string {
	return d.tag
}

func (d Device) Address() string {
	return d.address
}

func NewDevice(id int64, inuse bool, tag string) *Device {
	return &Device{id: id, inuse: inuse, tag: tag}
}

type Network struct {
	simple.WeightedDirectedGraph
	localDevice *Device
}

func (g *Network) LocalDevice() *Device {
	return g.localDevice
}

func (g *Network) GetDevice(id int64) *Device {
	node := g.Node(id)
	if node == nil {
		return nil
	}
	return node.(*Device)
}

func (g *Network) NodeIdExists(id int64) bool {
	return g.Node(id) != nil
}

func (g *Network) SetAllNodesUnseen() {
	nodes := g.Nodes()
	for nodes.Next() {
		node := nodes.Node().(*Device)
		node.SetSeen(false)
	}
}

func (g *Network) ChangeEdgeWeight(fromId int64, toId int64, weightFrom float64, weightTo float64) {
	fromNode := g.GetDevice(fromId)
	toNode := g.GetDevice(toId)
	if toNode == nil {
		toNode = NewDevice(toId, true, "")
		g.AddNode(toNode)
	}

	toNode.SetSeen(true)
	if !g.HasEdgeFromTo(fromId, toId) {
		g.SetWeightedEdge(g.NewWeightedEdge(fromNode, toNode, weightTo))
	} else {
		edgeTo := g.WeightedEdge(fromId, toId).(simple.WeightedEdge)
		edgeTo.W = weightTo
		g.SetWeightedEdge(edgeTo)
	}
}

// GetPath returns the shortest path from the local device to the target device, along with the total path weight.
//
// Parameters:
//   - to: The target Device to find a path to
//
// Returns:
//   - []int64: Array of node IDs representing the path from local device to target
//   - float64: Total weight/cost of the path
//   - error: Error if no path exists or target device is not active
//
// The path returned will be the shortest path based on edge weights using Dijkstra's algorithm.
// Returns an error if:
// - The target device is not marked as in use/active
// - No valid path exists between the local device and target
// The weight returned is currently always 0 (not implemented).

func (g *Network) GetPath(to *Device) ([]int64, float64, error) {
	if !to.InUse() {
		return nil, 0, fmt.Errorf("node is 0x%06X is not active", to.ID())
	}
	allShortest := path.DijkstraAllPaths(g)
	allBetween, weight := allShortest.AllBetween(g.localDevice.ID(), to.ID())
	if len(allBetween) == 0 {
		return nil, 0, fmt.Errorf("no path found between 0x%06X and 0x%06X", g.localDevice.ID(), to.ID())
	}
	logrus.WithFields(logrus.Fields{"length": len(allBetween[0]), "weight": weight}).
		Debug(fmt.Sprintf("Get path from 0x%06X to 0x%06X", g.localDevice.ID(), to.ID()))

	nodes := allBetween[0]
	path := make([]int64, len(nodes))
	for i, item := range nodes {
		item := item.(*Device)
		path[i] = item.ID()
	}

	return path, 0, nil
}

func (g *Network) SaveToFile(filename string) error {
	return g.writeGraph(filename)
}

func NewNetwork(localDeviceId int64) *Network {
	network := Network{localDevice: NewDevice(localDeviceId, true, "local")}
	network.WeightedDirectedGraph = *simple.NewWeightedDirectedGraph(0, math.Inf(1))
	network.AddNode(network.localDevice)
	return &network
}

func NewNeworkFromFile(filename string, localDeviceId int64) (*Network, error) {
	network := Network{}
	network.WeightedDirectedGraph = *simple.NewWeightedDirectedGraph(0, math.Inf(1))
	err := network.readGraph(filename)
	if err != nil {
		return nil, err
	}

	network.localDevice = network.GetDevice(localDeviceId)
	if network.localDevice == nil {
		network.localDevice = NewDevice(localDeviceId, true, "local")
		logger.WithField("device", FmtDeviceId(network.localDevice)).Warn("Local device not found in graph, adding it. Will be an isolated node")
		network.AddNode(network.localDevice)
	}
	return &network, nil
}
