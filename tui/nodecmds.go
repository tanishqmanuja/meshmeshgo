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
	focusSelection
	maxFocusItems
)

const (
	commandReboot = iota
	commandSetTag
)

type NodeInfoModel struct {
	ti      termInfo
	id      int64
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
	}

}

func (m *NodeInfoModel) focus(i int) {
	switch i {
	case focusNodeTag:
		m.txt[focusNodeTag].Focus()
	}

}

func (m *NodeInfoModel) sendReboot() error {
	_, err := sconn.SendReceiveApiProt(meshmesh.FirmRevApiRequest{}, meshmesh.UnicastProtocol, meshmesh.MeshNodeId(m.id))
	if err != nil {
		return err
	}
	return nil
}

func (m *NodeInfoModel) sendSetTag(tag string) error {
	_, err := sconn.SendReceiveApiProt(meshmesh.NodeSetTagApiRequest{Tag: tag}, meshmesh.UnicastProtocol, meshmesh.MeshNodeId(m.id))
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
	buffer.WriteString("Node ID        : 0x")
	buffer.WriteString(strconv.FormatInt(m.id, 16))
	buffer.WriteString("\nPath to        : ")
	buffer.WriteString(graph.FmtNodePath(gpath, m.id))
	buffer.WriteString("\nFirmware rev   : ")
	buffer.WriteString(m.rev)
	buffer.WriteString("\nNode TAG       : ")
	buffer.WriteString(m.txt[0].View())
	buffer.WriteString(fmt.Sprintf("\nNode Log dest. : %d", m.cfg.LogDest))
	buffer.WriteString(fmt.Sprintf("\nNode Channel   : %d", m.cfg.Channel))
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

func NewNodeInfoModel(ti termInfo, id int64) Model {
	rep, err := sconn.SendReceiveApiProt(meshmesh.FirmRevApiRequest{}, meshmesh.UnicastProtocol, meshmesh.MeshNodeId(id))
	if err != nil {
		return &ErrorReplyModel{err: err}
	}
	rev := rep.(meshmesh.FirmRevApiReply)
	rep, err = sconn.SendReceiveApiProt(meshmesh.NodeConfigApiRequest{}, meshmesh.UnicastProtocol, meshmesh.MeshNodeId(id))
	if err != nil {
		return &ErrorReplyModel{err: err}
	}
	cfg := rep.(meshmesh.NodeConfigApiReply)

	txt1 := textinput.New()
	txt1.Placeholder = "<node tag>"
	txt1.CharLimit = 31
	txt1.Width = 35
	txt1.Prompt = ""
	txt1.Focus()
	txt1.SetValue(string(cfg.Tag))

	txt1.PromptStyle = ti.renderer.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(32))
	txt1.PlaceholderStyle = ti.renderer.NewStyle().Foreground(lipgloss.ANSIColor(8))
	txt1.TextStyle = ti.renderer.NewStyle()
	txt1.Cursor.TextStyle = ti.renderer.NewStyle()
	txt1.Cursor.Style = ti.renderer.NewStyle()

	txt := []textinput.Model{txt1}

	_sel := selection.New[string]("node action:", []string{"Node Reboot", "Node Set Tag"})
	_sel.Filter = nil
	_sel.Template = selection.DefaultTemplate
	sel := selection.NewModel(_sel)

	return &NodeInfoModel{ti: ti, id: id, rev: rev.Revision, cfg: cfg, txt: txt, sel: sel, focused: 0}
}
