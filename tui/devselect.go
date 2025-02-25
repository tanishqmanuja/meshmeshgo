package tui

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"leguru.net/m/v2/graph"
)

type itemDeviceDelegate struct {
	device *graph.Device
}

func (i itemDeviceDelegate) Title() string {
	return graph.FmtDeviceId(i.device)
}

func (i itemDeviceDelegate) Descrition() string {
	return i.device.Tag()
}

func (i itemDeviceDelegate) FilterValue() string {
	return i.device.Tag()
}

type deviceSelectModel struct {
	list list.Model
}

func (m deviceSelectModel) Init() tea.Cmd {
	return nil
}

func (m deviceSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd
	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m deviceSelectModel) View() string {
	return m.list.View()
}

func newDeviceSelectItemDelegate() list.DefaultDelegate {
	d := list.NewDefaultDelegate()
	return d
}

func createDeviceSelectModel(network *graph.Network) deviceSelectModel {
	devices := make([]list.Item, network.Nodes().Len())
	nodes := network.Nodes()

	i := 0
	for nodes.Next() {
		device := nodes.Node().(*graph.Device)
		devices[i] = itemDeviceDelegate{device: device}
		i += 1
	}

	return deviceSelectModel{list: list.New(devices, newDeviceSelectItemDelegate(), 0, 0)}
}
