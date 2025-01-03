package tui

import (
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

const (
	discoveryChoiceInit = iota
	discoveryChoiceDiscover
	discoveryChoiceSave
	discoveryChoiceClear
)

type DiscoveryModel struct {
	BaseModel
	table     table.Model
	sel       *selection.Model[choiceItem]
	procedure *meshmesh.DiscoveryProcedure
	nodeid    int64
	state     int
	err       error
}

type discoverErrorMsg error
type discoverInitDoneMsg int64
type discoverStepDoneMsg int64
type discoverSaveDoneMsg int64
type discoverClearDoneMsg int64

const (
	discoveryStateInit = iota
	discoveryStateWaitCommand
	discoveryStateInProgress
)

func initDiscoveryCmd(d *DiscoveryModel) tea.Cmd {
	return func() tea.Msg {
		d.procedure = meshmesh.NewDiscoveryProcedure(sconn, gpath, d.nodeid)
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

func saveDiscoveryCmd(d *DiscoveryModel) tea.Cmd {
	return func() tea.Msg {
		err := d.procedure.Save()
		if err != nil {
			return discoverErrorMsg(err)
		}
		return discoverSaveDoneMsg(d.procedure.CurrentNode())
	}
}

func clearDiscoveryCmd(d *DiscoveryModel) tea.Cmd {
	return func() tea.Msg {
		nodeid := d.procedure.CurrentNode()
		d.procedure = nil
		return discoverClearDoneMsg(nodeid)
	}
}

func (m *DiscoveryModel) makeColumns() []table.Column {
	return []table.Column{
		table.NewColumn(colDiscKeyId, "Id", 10),
		table.NewColumn(colDiscKeyAddr, "Node Address", 17),
		table.NewColumn(colDiscKeyCurr, "Prev", 8),
		table.NewColumn(colDiscKeyNext, "Next", 8),
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

func (m *DiscoveryModel) initChoices() {
	m.sel = createSelectionModel("select action:", []choiceItem{
		{ID: discoveryChoiceInit, Name: "[I] Init discovery procedure"},
		{ID: discoveryChoiceDiscover, Name: "[D] Discover from this node"},
		{ID: discoveryChoiceSave, Name: "[S] Save discovery to file"},
		{ID: discoveryChoiceClear, Name: "[C] Clear discovery"},
	})
	m.sel.Init()
}

func (m *DiscoveryModel) Init() tea.Cmd {
	m.state = discoveryStateInit
	return tea.Batch([]tea.Cmd{m.initSpinner(), initDiscoveryCmd(m)}...)
}

func (m *DiscoveryModel) Update(msg tea.Msg) (Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds tea.BatchMsg
	)

	switch msg := msg.(type) {
	case discoverErrorMsg:
		m.state = discoveryStateWaitCommand
		m.err = msg
		m.initChoices()
	case discoverInitDoneMsg:
		m.state = discoveryStateWaitCommand
		m.table = m.table.WithRows(m.tableRows())
		m.initChoices()
	case discoverStepDoneMsg:
		m.state = discoveryStateWaitCommand
		m.table = m.table.WithRows(m.tableRows())
		m.initChoices()
	case discoverClearDoneMsg:
		m.state = discoveryChoiceInit
		cmd = initDiscoveryCmd(m)
		cmds = append(cmds, cmd)
		m.initChoices()
	}

	cmd = m.updateSpinner(msg)
	cmds = append(cmds, cmd)

	switch m.state {
	case discoveryStateWaitCommand:
		m.table, cmd = m.table.Update(msg)
		cmds = append(cmds, cmd)
		_, cmd = m.sel.Update(msg)
		if cmd != nil {
			msg := cmd()
			switch msg.(type) {
			case tea.QuitMsg:
				sel, err := m.sel.ValueAsChoice()
				if err != nil {
					m.err = err
				} else {
					switch sel.Value.ID {
					case discoveryChoiceInit:
						cmd = initDiscoveryCmd(m)
						cmds = append(cmds, cmd)
					case discoveryChoiceDiscover:
						m.state = discoveryStateInProgress
						cmd = stepDiscoveryCmd(m)
						cmds = append(cmds, cmd)
					case discoveryChoiceSave:
						cmd = saveDiscoveryCmd(m)
						cmds = append(cmds, cmd)
					case discoveryChoiceClear:
						cmd = clearDiscoveryCmd(m)
						cmds = append(cmds, cmd)
					}
				}
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *DiscoveryModel) View() string {
	views := []string{}
	if m.state >= discoveryStateWaitCommand {
		views = append(views, "Current Node: "+utils.FmtNodeId(m.procedure.CurrentNode()))
	}

	if m.state >= discoveryStateWaitCommand {
		views = append(views, m.table.View())
		views = append(views, m.sel.View())
	}

	if m.state == discoveryStateInProgress {
		views = append(views, m.ti.progressStyle.Render(m.viewSpinner()+" Discovery in progress from node "+utils.FmtNodeId(m.procedure.CurrentNode())))
	}

	if m.err != nil {
		views = append(views, m.ti.errorStyle.Render(m.err.Error()))
	}
	return lipgloss.JoinVertical(lipgloss.Top, views...)
}

func (m *DiscoveryModel) Focused() bool {
	return true
}

func NewDiscoveryModel(ti termInfo, nodeid int64) *DiscoveryModel {
	model := DiscoveryModel{BaseModel: NewBaseModel(ti), nodeid: nodeid}
	_cols := model.makeColumns()
	_rows := model.tableRows()
	_style := ti.renderer.NewStyle().Align(lipgloss.Left)
	_table := table.New(_cols).WithRows(_rows).BorderRounded().WithBaseStyle(_style).WithPageSize(20).SortByAsc(colKeyId).Focused(true)
	model.table = _table
	model.createSpinner()
	return &model
}
