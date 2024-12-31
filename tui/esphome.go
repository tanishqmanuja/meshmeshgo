package tui

import (
	"fmt"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
	"leguru.net/m/v2/graph"
	"leguru.net/m/v2/logger"
	"leguru.net/m/v2/meshmesh"
	"leguru.net/m/v2/utils"
)

const (
	colConnKeyId       = "id"
	colConnKeyActive   = "ison"
	colConnKeyTag      = "tag"
	colConnKeyHost     = "addr"
	colConnKeyNodeId   = "node"
	colConnKeyHandle   = "hndl"
	colConnKeySent     = "sent"
	colConnKeyReceived = "recv"
	colConnKeyDuration = "time"
	colConnKeyStart    = "from"
)

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second*1, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

type EspHomeModel struct {
	ti                     termInfo
	table                  table.Model
	lastTableSelectedEvent table.UserEventRowSelectToggled
}

func (m *EspHomeModel) makeColumns() []table.Column {
	return []table.Column{
		table.NewColumn(colConnKeyHost, "Host", 16),
		table.NewColumn(colConnKeyNodeId, "NodeId", 8),
		table.NewColumn(colConnKeyTag, "Tag", 24),
		table.NewColumn(colConnKeyActive, "A", 1),
		table.NewColumn(colConnKeyHandle, "Hndl", 4),
		table.NewColumn(colConnKeySent, "Sent", 8),
		table.NewColumn(colConnKeyReceived, "Recv", 8),
		table.NewColumn(colConnKeyDuration, "Delta", 18),
		table.NewColumn(colConnKeyStart, "Started", 18),
	}
}

func (m *EspHomeModel) makeRow(nodeid meshmesh.MeshNodeId, client *meshmesh.ApiConnection) table.Row {
	device := gpath.Node(int64(nodeid)).(*graph.Device)
	return table.NewRow(
		table.RowData{
			colConnKeyHost:     utils.FmtNodeIdHass(int64(nodeid)),
			colConnKeyNodeId:   utils.FmtNodeId(int64(nodeid)),
			colConnKeyTag:      device.Tag(),
			colConnKeyActive:   client.Stats.IsActiveAsText(),
			colConnKeyHandle:   client.Stats.GetLastHandle(),
			colConnKeySent:     client.Stats.BytesOut(),
			colConnKeyReceived: client.Stats.BytesIn(),
			colConnKeyDuration: client.Stats.TimeSinceLastConnection().String(),
			colConnKeyStart:    client.Stats.LastConnectionDuration().String(),
		},
	)
}

func (m *EspHomeModel) tableRows() []table.Row {
	rows := []table.Row{}
	for _, server := range esphome.Servers {
		for _, client := range server.Clients {
			rows = append(rows, m.makeRow(server.Address, client))
		}
	}
	return rows
}

func (m *EspHomeModel) Init() tea.Cmd {
	logger.Log().Println("EspHomeModel.Init")
	return tickCmd()
}

func (m *EspHomeModel) Update(msg tea.Msg) (Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	m.table, cmd = m.table.Update(msg)
	cmds = append(cmds, cmd)

	for _, e := range m.table.GetLastUpdateUserEvents() {
		switch e := e.(type) {
		case table.UserEventRowSelectToggled:
			m.lastTableSelectedEvent = e
		}
	}

	switch msg := msg.(type) {
	case tickMsg:
		m.table = m.table.WithRows(m.tableRows())
		cmds = append(cmds, tickCmd())
	case tea.KeyMsg:
		switch msg.String() {
		case "c":
			if m.table.HighlightedRow().Data[colConnKeyNodeId] != nil {
				addr, err := strconv.ParseInt(m.table.HighlightedRow().Data[colConnKeyNodeId].(string), 0, 32)
				if err == nil {
					esphome.CloseConnection(meshmesh.MeshNodeId(addr))
				}
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *EspHomeModel) View() string {
	//var buffer bytes.Buffer
	view := lipgloss.JoinVertical(
		lipgloss.Left,
		fmt.Sprintf("Last selected: %d", m.lastTableSelectedEvent.RowIndex),
		fmt.Sprintf("Highlighted: %s", m.table.HighlightedRow().Data[colKeyAddr]),
		fmt.Sprintf("Active connections: %d\n", esphome.Stats().CountCounnections()),
		m.table.View(),
	)
	return m.ti.renderer.NewStyle().MarginLeft(1).Render(view)
	//buffer.WriteString(fmt.Sprintf("Active connections: %d\n", esphome.Stats().CountCounnections()))
	//buffer.WriteString(m.table.View())
	//return buffer.String()
}

func (m *EspHomeModel) Focused() bool {
	return true
}

func NewEspHomeModel(ti termInfo) *EspHomeModel {
	model := EspHomeModel{ti: ti}
	_cols := model.makeColumns()
	_rows := model.tableRows()
	_style := ti.renderer.NewStyle().Align(lipgloss.Left)
	_table := table.New(_cols).WithRows(_rows).BorderRounded().WithBaseStyle(_style).WithPageSize(20).SortByAsc(colKeyId).Focused(true)

	model.table = _table
	return &model
}
