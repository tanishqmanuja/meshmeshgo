package tui

import (
	"bytes"
	"log"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/erikgeiser/promptkit/selection"
	"github.com/evertras/bubble-table/table"
	"leguru.net/m/v2/meshmesh"
	"leguru.net/m/v2/utils"
)

const (
	colDiscKeyId    = "id"
	colDiscKeyAddr  = "addr"
	colDiscKeyCurr  = "prev"
	colDiscKeyNext  = "curr"
	colDiscKeyDelta = "delta"
)

type DiscoveryModel struct {
	ti        termInfo
	table     table.Model
	sel       *selection.Model[string]
	procedure *meshmesh.DiscoveryProcedure
}

type discoverErrorMsg error
type discoverInitDoneMsg int64
type discoverStepDoneMsg int64

func initDiscoveryCmd(d *DiscoveryModel) tea.Cmd {
	return func() tea.Msg {
		d.procedure = meshmesh.NewDiscoveryProcedure(sconn)
		err := d.procedure.Init()
		if err != nil {
			return discoverErrorMsg(err)
		}
		return discoverInitDoneMsg(d.procedure.CurrentNode())
	}
}

func stepDiscoveryCmd(d *DiscoveryModel) tea.Cmd {
	return func() tea.Msg {
		err := d.procedure.Step()
		if err != nil {
			return discoverErrorMsg(err)
		}
		return discoverStepDoneMsg(d.procedure.CurrentNode())
	}
}

func (m *DiscoveryModel) makeColumns() []table.Column {
	return []table.Column{
		table.NewColumn(colDiscKeyId, "Id", 10),
		table.NewColumn(colDiscKeyAddr, "Node Address", 17),
		table.NewColumn(colDiscKeyCurr, "Prev", 8),
		table.NewColumn(colDiscKeyNext, "Curr", 8),
		table.NewColumn(colDiscKeyDelta, "Delta", 8),
	}
}

func (m *DiscoveryModel) makeRow(nid int64, curr float64, next float64) table.Row {
	return table.NewRow(
		table.RowData{
			colKeyId:        utils.FmtNodeId(nid),
			colKeyAddr:      utils.FmtNodeIdHass(nid),
			colDiscKeyCurr:  strconv.FormatFloat(curr, 'f', 2, 32),
			colDiscKeyNext:  strconv.FormatFloat(next, 'f', 2, 32),
			colDiscKeyDelta: strconv.FormatFloat(next-curr, 'f', 2, 32),
		},
	)
}

func (m *DiscoveryModel) tableRows() []table.Row {
	rows := []table.Row{}
	if m.procedure != nil {
		for i, d := range m.procedure.Neighbors {
			rows = append(rows, m.makeRow(i, d.Current, d.Next))
		}
	}
	return rows
}

func (m *DiscoveryModel) blur(i int) {
}

func (m *DiscoveryModel) focus(i int) {
}

func (m *DiscoveryModel) changeFocus() {
}

func (m *DiscoveryModel) Init() tea.Cmd {
	return tea.Batch([]tea.Cmd{m.sel.Init(), initDiscoveryCmd(m)}...)
}

func (m *DiscoveryModel) Update(msg tea.Msg) (Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds tea.BatchMsg
	)

	switch msg := msg.(type) {
	case discoverErrorMsg:
		log.Println("Discovery error")
	case discoverInitDoneMsg:
		log.Println("Discovery done")
	case discoverStepDoneMsg:
		log.Println("Discovery Step done")
		m.table = m.table.WithRows(m.tableRows())
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeySpace:
			var cmd tea.Cmd
			if m.procedure == nil {
				cmd = initDiscoveryCmd(m)
			} else {
				cmd = stepDiscoveryCmd(m)

			}
			cmds = append(cmds, cmd)
		case tea.KeyTab:
			m.changeFocus()
		}
	}

	_, cmd = m.sel.Update(msg)
	cmds = append(cmds, cmd)
	m.table, cmd = m.table.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *DiscoveryModel) View() string {
	var buffer bytes.Buffer
	if m.procedure == nil {
		buffer.WriteString("Discovery inactive")
	} else {
		buffer.WriteString("Current Node: 0x")
		buffer.WriteString(utils.FmtNodeId(m.procedure.CurrentNode()))
	}
	buffer.WriteString("\n")

	buffer.WriteString(m.table.View())
	buffer.WriteString("\n\n")
	buffer.WriteString(m.sel.View())
	return buffer.String()
}

func (m *DiscoveryModel) Focused() bool {
	return true
}

func NewDiscoveryModel(ti termInfo) *DiscoveryModel {
	model := DiscoveryModel{ti: ti}
	_cols := model.makeColumns()
	_rows := model.tableRows()
	_style := ti.renderer.NewStyle().Align(lipgloss.Left)
	_table := table.New(_cols).WithRows(_rows).BorderRounded().WithBaseStyle(_style).WithPageSize(20).SortByAsc(colKeyId).Focused(true)
	model.table = _table
	_sel := selection.New[string]("discoery action:", []string{"[I] Init discovery procedure", "[D] Discover from this node"})
	_sel.Filter = nil
	_sel.Template = selection.DefaultTemplate
	model.sel = selection.NewModel(_sel)

	return &model

}
