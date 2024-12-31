package tui

import (
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
	"leguru.net/m/v2/graph"
	"leguru.net/m/v2/utils"
)

const (
	colKeyId   = "id"
	colKeyAddr = "addr"
	colKeyTag  = "tag"
	colKeyPath = "path"
	colKeyCost = "cost"
)

func makeColumns() []table.Column {
	return []table.Column{
		table.NewColumn(colKeyId, "Id", 10),
		table.NewColumn(colKeyAddr, "Node Address", 17),
		table.NewColumn(colKeyTag, "Node Tag", 20),
		table.NewColumn(colKeyPath, "Path", 38),
		table.NewColumn(colKeyCost, "Cost", 7),
	}
}

func makeRow(dev *graph.Device, path string, cost float64) table.Row {
	return table.NewRow(
		table.RowData{
			colKeyId:   graph.FmtDeviceId(dev),
			colKeyAddr: graph.FmtDeviceIdHass(dev),
			colKeyTag:  dev.Tag(),
			colKeyPath: path,
			colKeyCost: strconv.FormatFloat(cost, 'f', 2, 32),
		},
	)
}

func graphTableRows() []table.Row {
	rows := []table.Row{}

	devices := gpath.Nodes()
	for devices.Next() {
		dev := devices.Node().(*graph.Device)
		var _path string
		path, weight, err := gpath.GetPath(dev)
		if err == nil {
			for _, p := range path {
				if len(_path) > 0 {
					_path += " > "
				}
				_path += utils.FmtNodeId(p)
			}
		}

		row := makeRow(dev, _path, weight)
		rows = append(rows, row)
	}

	return rows
}

type GraphShowModel struct {
	ti    termInfo
	table table.Model
	//focused int
}

/*func (m *GraphShowModel) blur(i int) {
}

func (m *GraphShowModel) focus(i int) {
}*/

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
