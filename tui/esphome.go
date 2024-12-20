package tui

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
	"leguru.net/m/v2/logger"
	"leguru.net/m/v2/meshmesh"
	"leguru.net/m/v2/utils"
)

const (
	colConnKeyId       = "id"
	colConnKeyActive   = "ison"
	colConnKeyAddr     = "addr"
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
	ti    termInfo
	table table.Model
}

func (m *EspHomeModel) makeColumns() []table.Column {
	return []table.Column{
		table.NewColumn(colConnKeyId, "Id", 2),
		table.NewColumn(colConnKeyActive, "A", 1),
		table.NewColumn(colConnKeyAddr, "Address", 8),
		table.NewColumn(colConnKeyHandle, "Hndl", 4),
		table.NewColumn(colConnKeySent, "Sent", 12),
		table.NewColumn(colConnKeyReceived, "Recv", 12),
		table.NewColumn(colConnKeyDuration, "Delta", 8),
		table.NewColumn(colConnKeyStart, "Started", 12),
	}
}

func (m *EspHomeModel) makeRow(index int, isactive string, addr meshmesh.MeshNodeId, handle uint16, sent int, recv int, duration time.Duration, start time.Duration) table.Row {
	return table.NewRow(
		table.RowData{
			colConnKeyId:       strconv.Itoa(index),
			colConnKeyActive:   isactive,
			colConnKeyAddr:     utils.FmtNodeId(int64(addr)),
			colConnKeyHandle:   strconv.Itoa(int(handle)),
			colConnKeySent:     strconv.Itoa(int(sent)),
			colConnKeyReceived: strconv.Itoa(int(recv)),
			colConnKeyDuration: duration.String(),
			colConnKeyStart:    start.String(),
		},
	)
}

func (m *EspHomeModel) tableRows() []table.Row {
	num := 0
	rows := []table.Row{}
	stats := esphome.Stats()
	for nodeid, conn := range stats.Connections {
		num += 1
		rows = append(rows, m.makeRow(
			num,
			conn.IsActiveAsText(),
			nodeid,
			conn.GetLastHandle(),
			conn.BytesOut(),
			conn.BytesIn(),
			conn.LastConnectionDuration().Round(time.Second),
			conn.TimeSinceLastConnection().Round(time.Second),
		))
	}
	return rows
}

func (m *EspHomeModel) Init() tea.Cmd {
	logger.Log().Println("EspHomeModel.Init")
	return tickCmd()
}

func (m *EspHomeModel) Update(msg tea.Msg) (Model, tea.Cmd) {
	var (
		cmd tea.Cmd
	)

	switch msg.(type) {
	case tickMsg:
		m.table = m.table.WithRows(m.tableRows())
		return m, tea.Batch(tickCmd(), cmd)
	}

	return m, cmd
}

func (m *EspHomeModel) View() string {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("Active connections: %d\n", esphome.Stats().CountCounnections()))
	buffer.WriteString(m.table.View())
	return buffer.String()
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
