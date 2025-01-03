package tui

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/erikgeiser/promptkit/selection"
	"leguru.net/m/v2/graph"
	"leguru.net/m/v2/meshmesh"
)

const (
	focusNodeTag = iota
	focusNodeChannel
	focusSelection
	maxFocusItems
)

const (
	commandReboot = iota
	commandSetTag
	commandSetChannel
)

const (
	nodeInfoGetDeviceState = iota
	nodeInfoGetProtocolState
	nodeInfoReceivedState
	nodeInfoErrorState
)

type nodeInfoReceivedMsg bool
type nodeInfoErrorMsg error

func getProtocolFromChoice(choice choiceItem) meshmesh.MeshProtocol {
	switch choice.ID {
	case 0:
		return meshmesh.UnicastProtocol
	case 1:
		return meshmesh.MultipathProtocol
	case 2:
		return meshmesh.DirectProtocol
	}
	return meshmesh.UnicastProtocol
}

func nodeInfoGetCmd(m *NodeInfoModel) tea.Cmd {
	return func() tea.Msg {
		rep, err := sconn.SendReceiveApiProt(meshmesh.FirmRevApiRequest{}, m.protocol, meshmesh.MeshNodeId(m.device.ID()))
		if err != nil {
			m.err = err
			m.state = nodeInfoErrorState
			return nodeInfoErrorMsg(err)
		}
		rev := rep.(meshmesh.FirmRevApiReply)

		rep, err = sconn.SendReceiveApiProt(meshmesh.NodeConfigApiRequest{}, m.protocol, meshmesh.MeshNodeId(m.device.ID()))
		if err != nil {
			m.err = err
			m.state = nodeInfoErrorState
			return nodeInfoErrorMsg(err)
		}
		cfg := rep.(meshmesh.NodeConfigApiReply)

		m.rev = rev.Revision
		m.cfg = cfg
		return nodeInfoReceivedMsg(true)
	}
}

func createTextInput(ti termInfo, placeholder string, value string, chars int, focus bool) textinput.Model {
	txt := textinput.New()
	txt.Placeholder = placeholder
	txt.CharLimit = chars
	txt.Width = chars + 2
	txt.Prompt = ""
	if focus {
		txt.Focus()
	}
	txt.SetValue(value)

	txt.PromptStyle = ti.renderer.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(32))
	txt.PlaceholderStyle = ti.renderer.NewStyle().Foreground(lipgloss.ANSIColor(8))
	txt.TextStyle = ti.renderer.NewStyle()
	txt.Cursor.TextStyle = ti.renderer.NewStyle()
	txt.Cursor.Style = ti.renderer.NewStyle()

	return txt
}

type NodeInfoModel struct {
	BaseModel
	rev      string
	cfg      meshmesh.NodeConfigApiReply
	focused  int
	txt      []textinput.Model
	sel0     *selection.Model[choiceItem]
	sel1     *selection.Model[choiceItem]
	res      string
	state    int
	protocol meshmesh.MeshProtocol
	err      error
}

func (m *NodeInfoModel) blur(i int) {
	switch i {
	case focusNodeTag:
		m.txt[focusNodeTag].Blur()
	case focusNodeChannel:
		m.txt[focusNodeChannel].Blur()
	}

}

func (m *NodeInfoModel) focus(i int) {
	switch i {
	case focusNodeTag:
		m.txt[focusNodeTag].Focus()
	case focusNodeChannel:
		m.txt[focusNodeChannel].Focus()
	case focusSelection:
		//m.sel = createSelection()
	}

}

func (m *NodeInfoModel) sendReboot() error {
	_, err := sconn.SendReceiveApiProt(meshmesh.FirmRevApiRequest{}, m.protocol, meshmesh.MeshNodeId(m.device.ID()))
	if err != nil {
		return err
	}
	return nil
}

func (m *NodeInfoModel) sendSetTag(tag string) error {
	_, err := sconn.SendReceiveApiProt(meshmesh.NodeSetTagApiRequest{Tag: tag}, m.protocol, meshmesh.MeshNodeId(m.device.ID()))
	if err != nil {
		return err
	}
	return nil
}

func (m *NodeInfoModel) sendSetChannel(Channel uint8) error {
	_, err := sconn.SendReceiveApiProt(meshmesh.NodeSetChannelApiRequest{Channel: Channel}, m.protocol, meshmesh.MeshNodeId(m.device.ID()))
	if err != nil {
		return err
	}
	return nil
}

func (m *NodeInfoModel) nodeCommand(choice int) error {
	switch choice {
	case commandReboot:
		return m.sendReboot()
	case commandSetTag:
		return m.sendSetTag(m.txt[focusNodeTag].Value())
	case commandSetChannel:
		channel, err := strconv.ParseInt(m.txt[focusNodeChannel].Value(), 10, 8)
		if err != nil {
			return err
		}
		return m.sendSetChannel(uint8(channel))
	}
	return fmt.Errorf("choice %d not found", choice)
}

func (m *NodeInfoModel) Init() tea.Cmd {
	return tea.Batch(m.initDeviceSelection(), m.sel0.Init(), m.sel1.Init())
}

func (m *NodeInfoModel) updateNodeInfo(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.ti.height = msg.Height
		m.ti.width = msg.Width
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyTab:
			m.blur(m.focused)
			m.focused += 1
			m.focused = m.focused % maxFocusItems
			m.focus(m.focused)
		case tea.KeyEnter:
			if m.focused == focusSelection {
				cho, err := m.sel1.ValueAsChoice()
				if err == nil {
					err = m.nodeCommand(cho.Index())
					if err != nil {
						m.res = m.ti.renderer.NewStyle().Foreground(lipgloss.ANSIColor(9)).Render(err.Error())
					} else {
						m.res = m.ti.renderer.NewStyle().Foreground(lipgloss.ANSIColor(10)).Render("Command succesful")
					}
				}
			}
		}
	}

	var cmd tea.Cmd
	var cmds []tea.Cmd
	for i := range m.txt {
		m.txt[i], cmd = m.txt[i].Update(msg)
		cmds = append(cmds, cmd)
	}
	if m.focused == focusSelection {
		_, _ = m.sel1.Update(msg)
	}

	return m, tea.Batch(cmds...)
}

func (m *NodeInfoModel) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch m.state {
	case nodeInfoGetDeviceState:
		cmd = m.updateDeviceSelection(msg)
		cmds = append(cmds, cmd)
	case nodeInfoGetProtocolState:
		_, cmd = m.sel0.Update(msg)
		if cmd != nil {
			msg := cmd()
			switch msg.(type) {
			case tea.QuitMsg:
				choice, err := m.sel0.Value()
				if err != nil {
					m.err = err
				} else {
					m.protocol = getProtocolFromChoice(choice)
					cmd = nodeInfoGetCmd(m)
					cmds = append(cmds, cmd)
				}
			}
		}
	case nodeInfoReceivedState:
		_, cmd = m.updateNodeInfo(msg)
		cmds = append(cmds, cmd)
	}

	switch msg.(type) {
	case deviceItemSelectedMsg:
		m.state = nodeInfoGetProtocolState
	case nodeInfoReceivedMsg:
		m.txt[focusNodeTag].SetValue(string(m.cfg.Tag))
		m.txt[focusNodeChannel].SetValue(fmt.Sprintf("%d", m.cfg.Channel))
		m.state = nodeInfoReceivedState
	}

	return m, tea.Batch(cmds...)
}

func (m *NodeInfoModel) View() string {
	views := []string{}

	if m.state >= nodeInfoGetDeviceState {
		views = append(views, m.selDevice.View())
	}

	if m.state >= nodeInfoGetProtocolState {
		views = append(views, m.sel0.View())
	}

	if m.state == nodeInfoReceivedState {
		var buffer bytes.Buffer
		buffer.WriteString("Node ID        : ")
		buffer.WriteString(graph.FmtDeviceId(m.device))
		buffer.WriteString("\nPath to        : ")
		buffer.WriteString(graph.FmtNodePath(gpath, m.device))
		buffer.WriteString("\nFirmware rev   : ")
		buffer.WriteString(m.rev)
		buffer.WriteString("\nNode TAG       : ")
		buffer.WriteString(m.txt[focusNodeTag].View())
		buffer.WriteString(fmt.Sprintf("\nNode Log dest. : %d", m.cfg.LogDest))
		buffer.WriteString("\nNode Channel   : ")
		buffer.WriteString(m.txt[focusNodeChannel].View())
		buffer.WriteString(fmt.Sprintf("\nNode TxPower   : %d", m.cfg.TxPower))
		buffer.WriteString(fmt.Sprintf("\nNode Groups    : %d", m.cfg.Groups))
		buffer.WriteString(fmt.Sprintf("\nNode Binded    : 0x%06X", m.cfg.BindedServer))
		buffer.WriteString("\n\n")
		buffer.WriteString(m.sel1.View())
		if len(m.res) > 0 {
			buffer.WriteString(m.res)
		}

		views = append(views, buffer.String())
	}

	if m.err != nil {
		views = append(views, m.ti.errorStyle.Render(m.err.Error()))
	}

	return strings.Join(views, "\n")
}

func (m *NodeInfoModel) Focused() bool {
	return true
}

func NewNodeInfoModel(ti termInfo, dev *graph.Device) Model {
	txt1 := createTextInput(ti, "<node tag>", "", 31, true)
	txt2 := createTextInput(ti, "<node channel>", "", 6, false)
	txt := []textinput.Model{txt1, txt2}

	model := &NodeInfoModel{
		BaseModel: NewBaseModelExtended(ti, gpath),
		txt:       txt,
		sel0: createSelectionModel("Select protocol:", []choiceItem{
			{ID: 0, Name: "Unicast protocol"},
			{ID: 1, Name: "Multipath protocol"},
			{ID: 2, Name: "Direct protocol"}}),
		sel1: createSelectionModel("node action:", []choiceItem{
			{ID: 0, Name: "Node Reboot"},
			{ID: 1, Name: "Save Tag"},
			{ID: 2, Name: "Save Channel"}}),
		focused: 0,
		state:   nodeInfoGetDeviceState,
	}

	if dev != nil {
		model.device = dev
		model.state = nodeInfoGetProtocolState
	}

	return model
}
