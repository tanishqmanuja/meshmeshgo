package tui

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"leguru.net/m/v2/graph"
)

type termInfo struct {
	term     string
	width    int
	height   int
	renderer *lipgloss.Renderer
}

type HelpModel struct {
	ti termInfo
}

func (m *HelpModel) Init() tea.Cmd {
	var cmd tea.Cmd
	return cmd
}

func (m *HelpModel) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	return m, cmd
}

func (m *HelpModel) View() string {
	var buffer bytes.Buffer
	buffer.WriteString("Commands help\n")
	buffer.WriteString("-- node info 0xAABBCCDD\n")
	buffer.WriteString("-- graph\n")
	buffer.WriteString("-- coordinator\n")
	return buffer.String()
}

func (m *HelpModel) Focused() bool {
	return false
}

func NewHelpModel(ti termInfo) *HelpModel {
	return &HelpModel{ti: ti}
}

type ErrorReplyModel struct {
	ti  termInfo
	err error
}

func (m *ErrorReplyModel) Init() tea.Cmd {
	var cmd tea.Cmd
	return cmd
}

func (m *ErrorReplyModel) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	return m, cmd
}

func (m *ErrorReplyModel) View() string {
	return m.ti.renderer.NewStyle().Foreground(lipgloss.ANSIColor(9)).Render(fmt.Sprintf("Error: %s\n", m.err.Error()))
}

func (m *ErrorReplyModel) Focused() bool {
	return false
}

func NewErrorReplyModel(ti termInfo, err error) *ErrorReplyModel {
	return &ErrorReplyModel{ti: ti, err: err}
}

type CoordinatorInfoModel struct {
}

func (m *CoordinatorInfoModel) Init() tea.Cmd {
	var cmd tea.Cmd
	return cmd
}

func (m *CoordinatorInfoModel) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	return m, cmd
}

func (m *CoordinatorInfoModel) View() string {
	return fmt.Sprintf("Coordinator ID: %s\n", graph.FmtDeviceId(gpath.LocalDevice()))
}

func (m *CoordinatorInfoModel) Focused() bool {
	return false
}

func NewCoordinatorInfoModel(ti termInfo) *CoordinatorInfoModel {
	return &CoordinatorInfoModel{}
}

func tokenize(cmd string) []string {
	return strings.Fields(cmd)
}

func get_node_suggestions(tokens []string) []string {
	sugg := []string{}
	if len(tokens) == 0 {
		sugg = []string{"info"}
	}
	return sugg
}

func get_suggestions(cmd string) []string {
	sugg := []string{}
	tokens := tokenize(cmd)
	if len(tokens) == 0 {
		sugg = []string{"coordinator", "graph", "node"}
	} else {
		token := tokens[0]
		switch token {
		case "node":
			sugg = get_node_suggestions(tokens[1:])
		}
	}
	return sugg
}

func (m model) execute_node_info_command(tokens []string) Model {
	if len(tokens) == 1 {
		id, err := graph.ParseDeviceId(tokens[0])
		if err != nil {
			return NewErrorReplyModel(m.ti, err)
		} else {
			var dev *graph.Device
			if id == 0 {
				dev = graph.NewDevice(0, true, "local")
			} else {
				dev = gpath.GetDevice(id)
			}
			if dev == nil {
				return NewErrorReplyModel(m.ti, errors.New("node info: device not found in graph"))
			}
			return NewNodeInfoModel(m.ti, dev)
		}
	} else {
		return NewErrorReplyModel(m.ti, errors.New("node info: invalid node ID"))
	}
}

func (m model) execute_node_command(tokens []string) Model {
	if len(tokens) == 0 {

	} else {
		token := tokens[0]
		tokens = tokens[1:]
		if token == "info" {
			return m.execute_node_info_command(tokens)
		}
	}
	return NewErrorReplyModel(m.ti, errors.New("node: unknow command"))
}

func (m model) execute_discovery_command(tokens []string) Model {
	var nodeid int64 = 0
	if len(tokens) > 0 {
		var err error
		nodeid, err = graph.ParseDeviceId(tokens[0])
		if err != nil {
			return NewErrorReplyModel(m.ti, err)
		}
	}
	return NewDiscoveryModel(m.ti, nodeid)
}

func (m model) execute_firmware_command(tokens []string) Model {
	var nodeid int64 = 0
	if len(tokens) > 0 {
		var err error
		nodeid, err = graph.ParseDeviceId(tokens[0])
		if err != nil {
			return NewErrorReplyModel(m.ti, err)
		}
	}
	return NewFirmwareModel(m.ti, nodeid)
}

func (m model) execute_command(cmd string) Model {
	tokens := strings.Split(cmd, " ")
	if len(tokens) > 0 {
		token := tokens[0]
		tokens = tokens[1:]
		if token == "help" {
			return NewHelpModel(m.ti)
		} else if token == "coordinator" {
			return NewCoordinatorInfoModel(m.ti)
		} else if token == "graph" {
			return NewGraphShowModel(m.ti)
		} else if token == "node" {
			return m.execute_node_command(tokens)
		} else if token == "discovery" {
			return m.execute_discovery_command(tokens)
		} else if token == "esphome" {
			return NewEspHomeModel(m.ti)
		} else if token == "firmware" {
			return m.execute_firmware_command(tokens)
		}
	}
	return nil
}
