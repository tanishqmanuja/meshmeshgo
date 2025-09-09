package graph

import (
	"fmt"
	"strings"

	"leguru.net/m/v2/logger"
	"leguru.net/m/v2/utils"
)

func FmtDeviceId(device NodeDevice) string {
	return utils.FmtNodeId(device.ID())
}


func FmtDeviceIdHass(device NodeDevice) string {
	return utils.FmtNodeIdHass(device.ID())
}

func FmtNodePath(network *Network, device NodeDevice) string {
	var _path string
	path, _, err := network.GetPath(device)
	if err == nil {
		_path = utils.FmtPath2Str(path)
	}
	return _path
}

func PrintTable(network *Network) {
	if !network.NodeIdExists(network.localDeviceId) {
		logger.WithField("node", utils.FmtNodeId(network.localDeviceId)).Fatal("Local node does not exists in grpah")
	}
	fmt.Println("")
	fmt.Printf("|%s|\n", strings.Repeat("-", 116))
	fmt.Printf("| Coordinator node is: %-94s|\n", utils.FmtNodeId(network.localDeviceId))

	fmt.Printf("|%s|\n", strings.Repeat("-", 116))
	fmt.Println("| Node Id  | Node Address    | Node Tag           | Port | Path                                             | Wei. |")
	fmt.Printf("|%s|\n", strings.Repeat("-", 116))

	devices := network.Nodes()
	for devices.Next() {
		device := devices.Node().(NodeDevice)

		var _path string
		path, weight, err := network.GetPath(device)
		if err == nil {
			for _, p := range path {
				if len(_path) > 0 {
					_path += " > "
				}
				_path += utils.FmtNodeId(p)
			}
		}

		fmt.Printf("| %-8s | %-15s | %-18s | %6d | %-48s | %3.2f |\n", FmtDeviceId(device), FmtDeviceIdHass(device), device.Device().Tag(), 6053, _path, weight)
	}
	fmt.Printf("|%s|\n", strings.Repeat("-", 116))
	fmt.Println("")
}
