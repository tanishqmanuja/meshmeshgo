package tui

import (
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
	"leguru.net/m/v2/utils"
)

const (
	colKeyId   = "id"
	colKeyAddr = "addr"
	colKeyTag  = "tag"
	colKeyPort = "port"
	colKeyPath = "path"
	colKeyCost = "cost"
)

func makeColumns() []table.Column {
	return []table.Column{
		table.NewColumn(colKeyId, "Id", 10),
		table.NewColumn(colKeyAddr, "Node Address", 17),
		table.NewColumn(colKeyTag, "Node Tag", 20),
		table.NewColumn(colKeyPort, "Port", 6),
		table.NewColumn(colKeyPath, "Path", 38),
		table.NewColumn(colKeyCost, "Cost", 7),
	}
}

func makeRow(nid int64, tag string, port uint64, path string, cost float64) table.Row {
	return table.NewRow(
		table.RowData{
			colKeyId:   utils.FmtNodeId(nid),
			colKeyAddr: utils.FmtNodeId(nid),
			colKeyTag:  tag,
			colKeyPort: strconv.FormatUint(port, 10),
			colKeyPath: path,
			colKeyCost: strconv.FormatFloat(cost, 'f', 2, 32),
		},
	)
}

func graphTableRows() []table.Row {
	inuse := gpath.GetAllInUse()
	rows := []table.Row{}

	for _, d := range inuse {
		nid := d

		var _path string
		path, weight, err := gpath.GetPath(d)
		if err == nil {
			for _, p := range path {
				if len(_path) > 0 {
					_path += " > "
				}
				_path += utils.FmtNodeId(p)
			}
		}

		row := makeRow(nid, gpath.NodeTag(int64(nid)), 6053, _path, weight)
		rows = append(rows, row)
	}

	return rows
}

type GraphShowModel struct {
	ti      termInfo
	table   table.Model
	focused int
}

func (m *GraphShowModel) blur(i int) {
}

func (m *GraphShowModel) focus(i int) {
}

func (m *GraphShowModel) changeFocus() {
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

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		//m.table = m.table.WithTargetWidth(msg.Width)
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyTab:
			m.changeFocus()
		}
	}

	m.table, cmd = m.table.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *GraphShowModel) View() string {
	return m.table.View()
}

func (m *GraphShowModel) Focused() bool {
	return true
}

func NewGraphShowModel(ti termInfo) *GraphShowModel {
	_cols := makeColumns()
	_rows := graphTableRows()
	_style := ti.renderer.NewStyle().Align(lipgloss.Left)
	_table := table.New(_cols).WithRows(_rows).BorderRounded().WithBaseStyle(_style).WithPageSize(20).SortByAsc(colKeyId).Focused(true)
	return &GraphShowModel{ti: ti, table: _table}

}
