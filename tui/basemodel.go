package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/erikgeiser/promptkit/selection"
	"leguru.net/m/v2/graph"
	"leguru.net/m/v2/utils"
)

type choiceItem struct {
	ID   int
	Name string
}

func (c choiceItem) String() string {
	return c.Name
}

func createSelectionModel(placeholder string, choices []choiceItem) *selection.Model[choiceItem] {
	_sel := selection.New[choiceItem](placeholder, choices)
	_sel.Filter = nil
	_sel.Template = selection.DefaultTemplate
	_sel.ExtendedTemplateFuncs = map[string]interface{}{
		"name": func(c *selection.Choice[choiceItem]) string { return c.Value.Name },
	}
	return selection.NewModel(_sel)
}

type deviceItem struct {
	ID  int64
	Tag string
}

func (d deviceItem) String() string {
	if d.Tag == "" {
		return fmt.Sprintf("ID:%s", utils.FmtNodeId(d.ID))
	} else {
		return fmt.Sprintf("ID:%s TAG:(%s)", utils.FmtNodeId(d.ID), d.Tag)
	}
}

func createDeviceSelectionModel(network *graph.Network) *selection.Model[deviceItem] {
	choices := []deviceItem{}
	nodes := network.Nodes()
	for nodes.Next() {
		device := nodes.Node().(*graph.Device)
		choices = append(choices, deviceItem{ID: device.ID(), Tag: device.Tag()})
	}

	_sel := selection.New[deviceItem]("Pick a device", choices)
	_sel.Filter = func(filterText string, choice *selection.Choice[deviceItem]) bool {
		return strings.Contains(strings.ToLower(choice.Value.Tag), strings.ToLower(filterText))
	}
	_sel.Template = selection.DefaultTemplate
	_sel.ExtendedTemplateFuncs = map[string]interface{}{
		"name": func(c *selection.Choice[deviceItem]) string {
			return fmt.Sprintf("%s (%s)", c.Value.Tag, utils.FmtNodeId(c.Value.ID))
		},
	}
	return selection.NewModel(_sel)
}

type BaseModel struct {
	ti termInfo

	successStyle  lipgloss.Style
	errorStyle    lipgloss.Style
	progressStyle lipgloss.Style

	selDevice *selection.Model[deviceItem]
	spinner   spinner.Model

	network *graph.Network
	device  *graph.Device
}

type deviceItemSelectedMsg *graph.Device

func selectDeviceCmd(m *BaseModel) tea.Cmd {
	return func() tea.Msg {
		choice, err := m.selDevice.Value()
		if err == nil {
			m.device = m.network.GetDevice(choice.ID)
			return deviceItemSelectedMsg(m.device)
		}
		return nil
	}
}

func (m *BaseModel) initDeviceSelection() tea.Cmd {
	return m.selDevice.Init()
}

func (m *BaseModel) updateDeviceSelection(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	_, cmd = m.selDevice.Update(msg)
	if cmd != nil {
		msg := cmd()
		switch msg.(type) {
		case tea.QuitMsg:
			return selectDeviceCmd(m)
		}
	}
	return cmd
}

func (m *BaseModel) viewDeviceSelection() string {
	return m.selDevice.View()
}

func (m *BaseModel) createSpinner() {
	m.spinner = spinner.New()
	m.spinner.Spinner = spinner.Dot
}

func (m *BaseModel) initSpinner() tea.Cmd {
	return m.spinner.Tick
}

func (m *BaseModel) updateSpinner(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
	}
	return cmd
}

func (m *BaseModel) viewSpinner() string {
	return m.spinner.View()
}

func NewBaseModel(ti termInfo) BaseModel {
	return BaseModel{
		ti:            ti,
		successStyle:  ti.renderer.NewStyle().Foreground(lipgloss.ANSIColor(10)),
		errorStyle:    ti.renderer.NewStyle().Foreground(lipgloss.ANSIColor(9)),
		progressStyle: ti.renderer.NewStyle().Foreground(lipgloss.ANSIColor(11)),
	}
}

func NewBaseModelExtended(ti termInfo, network *graph.Network) BaseModel {
	model := BaseModel{
		ti:            ti,
		successStyle:  ti.renderer.NewStyle().Foreground(lipgloss.ANSIColor(10)),
		errorStyle:    ti.renderer.NewStyle().Foreground(lipgloss.ANSIColor(9)),
		progressStyle: ti.renderer.NewStyle().Foreground(lipgloss.ANSIColor(11)),
	}
	model.network = network
	model.selDevice = createDeviceSelectionModel(network)
	return model
}
