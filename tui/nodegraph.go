package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
	"leguru.net/m/v2/utils"
)

type GraphShowModel struct {
	ti    termInfo
	table table.Model
}

func (m *GraphShowModel) Init() tea.Cmd {
	var cmd tea.Cmd
	return cmd
}

func (m *GraphShowModel) Update(msg tea.Msg) (Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds tea.BatchMsg
	)
	m.table, cmd = m.table.Update(msg)
	cmds = append(cmds, cmd)
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.table.WithTargetWidth(msg.Width)
	}
	return m, tea.Batch(cmds...)
}

func (m *GraphShowModel) View() string {
	return m.table.View()
}

func (m *GraphShowModel) Focused() bool {
	return false
}

func graphTableRows() []table.Row {
	inuse := gpath.GetAllInUse()
	rows := []table.Row{}

	for _, d := range inuse {
		nid := uint32(d)

		var _path string
		path, weight, err := gpath.GetPath(d)
		if err == nil {
			for _, p := range path {
				if len(_path) > 0 {
					_path += " > "
				}
				_path += utils.FmtNodeId(uint32(p))
			}
		}

		row := table.NewRow(table.RowData{
			"id":   utils.FmtNodeId(nid),
			"addr": utils.FmtNodeIdHass(nid),
			"tag":  gpath.NodeTag(int64(nid)),
			"port": "6053",
			"path": _path,
			"cost": weight,
		})
		rows = append(rows, row)
	}

	return rows
}

func NewGraphShowModel(ti termInfo) *GraphShowModel {
	_style := ti.renderer.NewStyle().Align(lipgloss.Left)
	_table := table.New([]table.Column{
		table.NewColumn("id", "Node Id", 10),
		table.NewColumn("addr", "Node Address", 17),
		table.NewColumn("tag", "Node Tag", 20),
		table.NewColumn("port", "Port", 6),
		table.NewColumn("path", "Path", 38),
		table.NewColumn("cost", "Cost", 7).WithFormatString("%1.2f"),
	}).WithRows(graphTableRows()).WithBaseStyle(_style).BorderRounded().Focused(true)
	return &GraphShowModel{ti: ti, table: _table}

}
