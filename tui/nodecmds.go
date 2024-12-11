package tui

import (
	"bytes"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"leguru.net/m/v2/meshmesh"
)

type NodeInfoModel struct {
	id  int64
	cfg meshmesh.NodeConfigApiReply
}

func (m *NodeInfoModel) Init() tea.Cmd {
	var cmd tea.Cmd
	return cmd
}

func (m *NodeInfoModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	return m, cmd
}

func (m *NodeInfoModel) View() string {
	var buffer bytes.Buffer
	buffer.WriteString("Node ID: 0x")
	buffer.WriteString(strconv.FormatInt(m.id, 16))
	buffer.WriteString("\nNode TAG:")
	return buffer.String()
}

func NewNodeInfoModel(id int64) tea.Model {
	rep, err := sconn.SendReceiveApiProt(meshmesh.NodeConfigApiRequest{}, meshmesh.UnicastProtocol, meshmesh.MeshNodeId(id))
	if err != nil {
		return &ErrorReplyModel{err: err}
	} else {
		cfg := rep.(meshmesh.NodeConfigApiReply)
		return &NodeInfoModel{id: id, cfg: cfg}
	}

}
