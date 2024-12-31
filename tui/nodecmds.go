package tui

import (
	"bytes"
	"fmt"
	"strconv"

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

func createSelection() *selection.Model[string] {
	_sel := selection.New[string]("node action:", []string{"Node Reboot", "Save Tag", "Save Channel"})
	_sel.Filter = nil
	_sel.Template = selection.DefaultTemplate
	return selection.NewModel(_sel)
}

type NodeInfoModel struct {
	ti      termInfo
	dev     *graph.Device
	rev     string
	cfg     meshmesh.NodeConfigApiReply
	focused int
	txt     []textinput.Model
	sel     *selection.Model[string]
	res     string
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
	_, err := sconn.SendReceiveApiProt(meshmesh.FirmRevApiRequest{}, meshmesh.UnicastProtocol, meshmesh.MeshNodeId(m.dev.ID()))
	if err != nil {
		return err
	}
	return nil
}

func (m *NodeInfoModel) sendSetTag(tag string) error {
	_, err := sconn.SendReceiveApiProt(meshmesh.NodeSetTagApiRequest{Tag: tag}, meshmesh.UnicastProtocol, meshmesh.MeshNodeId(m.dev.ID()))
	if err != nil {
		return err
	}
	return nil
}

func (m *NodeInfoModel) sendSetChannel(Channel uint8) error {
	_, err := sconn.SendReceiveApiProt(meshmesh.NodeSetChannelApiRequest{Channel: Channel}, meshmesh.UnicastProtocol, meshmesh.MeshNodeId(m.dev.ID()))
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
	var cmd = m.sel.Init()
	return cmd
}

func (m *NodeInfoModel) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd = make([]tea.Cmd, len(m.txt))

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
				cho, err := m.sel.ValueAsChoice()
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

	for i := range m.txt {
		m.txt[i], cmds[i] = m.txt[i].Update(msg)
	}
	if m.focused == focusSelection {
		_, _ = m.sel.Update(msg)
	}
	return m, tea.Batch(cmds...)
}

func (m *NodeInfoModel) View() string {
	var buffer bytes.Buffer
	buffer.WriteString("Node ID        : ")
	buffer.WriteString(graph.FmtDeviceId(m.dev))
	buffer.WriteString("\nPath to        : ")
	buffer.WriteString(graph.FmtNodePath(gpath, m.dev))
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
	buffer.WriteString(m.sel.View())
	if len(m.res) > 0 {
		buffer.WriteString(m.res)
	}

	return buffer.String()
}

func (m *NodeInfoModel) Focused() bool {
	return true
}

func NewNodeInfoModel(ti termInfo, dev *graph.Device) Model {
	rep, err := sconn.SendReceiveApiProt(meshmesh.FirmRevApiRequest{}, meshmesh.UnicastProtocol, meshmesh.MeshNodeId(dev.ID()))
	if err != nil {
		return &ErrorReplyModel{err: err}
	}
	rev := rep.(meshmesh.FirmRevApiReply)
	rep, err = sconn.SendReceiveApiProt(meshmesh.NodeConfigApiRequest{}, meshmesh.UnicastProtocol, meshmesh.MeshNodeId(dev.ID()))
	if err != nil {
		return &ErrorReplyModel{err: err}
	}
	cfg := rep.(meshmesh.NodeConfigApiReply)
	txt1 := createTextInput(ti, "<node tag>", string(cfg.Tag), 31, true)
	txt2 := createTextInput(ti, "<node channel>", fmt.Sprintf("%d", cfg.Channel), 6, false)
	txt := []textinput.Model{txt1, txt2}

	return &NodeInfoModel{ti: ti, dev: dev, rev: rev.Revision, cfg: cfg, txt: txt, sel: createSelection(), focused: 0}
}
