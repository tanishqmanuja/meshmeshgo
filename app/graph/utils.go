package graph

import (
	"errors"
	"fmt"
	"strconv"

	"leguru.net/m/v2/logger"
	"leguru.net/m/v2/utils"
)

func FmtDeviceId(device NodeDevice) string {
	return fmt.Sprintf("0x%06X", device.ID())
}

func ParseDeviceId(id string) (int64, error) {
	if len(id) < 1 {
		return 0, errors.New("invalid id string")
	}
	return strconv.ParseInt(id, 0, 32)
}

func FmtDeviceIdHass(device NodeDevice) string {
	id := device.ID()
	return fmt.Sprintf("127.%d.%d.%d", (id>>16)&0xFF, (id>>8)&0xFF, id&0xFF)
}

func FmtNodePath(network *Network, device NodeDevice) string {
	var _path string
	path, _, err := network.GetPath(device)
	if err == nil {
		for _, p := range path {
			if len(_path) > 0 {
				_path += " > "
			}
			_path += utils.FmtNodeId(p)
		}
	}
	return _path
}

func PrintTable(network *Network) {
	if !network.NodeIdExists(network.localDeviceId) {
		logger.WithField("node", utils.FmtNodeId(network.localDeviceId)).Fatal("Local node does not exists in grpah")
	}

	fmt.Println("Coordinator node is " + utils.FmtNodeId(network.localDeviceId))

	fmt.Println("|----------|----------------|--------------------|------|--------------------------------------------------|------|")
	fmt.Println("| Node Id  | Node Address   | Node Tag           | Port | Path                                             | Wei. |")
	fmt.Println("|----------|----------------|--------------------|------|--------------------------------------------------|------|")

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

		fmt.Printf("| %s | %15s | %-18s | %4d | %-48s | %3.2f |\n", FmtDeviceId(device), FmtDeviceIdHass(device), device.Device().Tag(), 6053, _path, weight)
	}

}
