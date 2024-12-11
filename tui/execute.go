package tui

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
	"leguru.net/m/v2/utils"
)

type ErrorReplyModel struct {
	err error
}

func (m *ErrorReplyModel) Init() tea.Cmd {
	var cmd tea.Cmd
	return cmd
}

func (m *ErrorReplyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	return m, cmd
}

func (m *ErrorReplyModel) View() string {
	return fmt.Sprintf("Error: %s\n", m.err.Error())
}

func NewErrorReplyModel(err error) *ErrorReplyModel {
	return &ErrorReplyModel{err: err}
}

type CoordinatorInfoModel struct {
}

func (m *CoordinatorInfoModel) Init() tea.Cmd {
	var cmd tea.Cmd
	return cmd
}

func (m *CoordinatorInfoModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	return m, cmd
}

func (m *CoordinatorInfoModel) View() string {
	return fmt.Sprintf("Coordinator ID: %s\n", utils.FmtNodeId(uint32(gpath.SourceNode)))
}

func NewCoordinatorInfoModel() *CoordinatorInfoModel {
	return &CoordinatorInfoModel{}
}

type GraphShowModel struct {
	table table.Model
}

func (m *GraphShowModel) Init() tea.Cmd {
	var cmd tea.Cmd
	return cmd
}

func (m *GraphShowModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func NewGraphShowModel() *GraphShowModel {
	_style := lipgloss.NewStyle().Align(lipgloss.Left)
	_table := table.New([]table.Column{
		table.NewColumn("id", "Node Id", 10),
		table.NewColumn("addr", "Node Address", 17),
		table.NewColumn("tag", "Node Tag", 20),
		table.NewColumn("port", "Port", 6),
		table.NewFlexColumn("path", "Path", 38),
		table.NewColumn("cost", "Cost", 7).WithFormatString("%1.2f"),
	}).WithRows(graphTableRows()).WithBaseStyle(_style).BorderRounded().Focused(true)
	return &GraphShowModel{table: _table}

}

func tokenize(cmd string) []string {
	return strings.Fields(cmd)
}

func get_node_suggestions(tokens []string) []string {
	sugg := []string{}
	if len(tokens) == 0 {
		sugg = []string{"info"}
	}
	return sugg
}

func get_suggestions(cmd string) []string {
	sugg := []string{}
	tokens := tokenize(cmd)
	if len(tokens) == 0 {
		sugg = []string{"coordinator", "graph", "node"}
	} else {
		token := tokens[0]
		switch token {
		case "node":
			sugg = get_node_suggestions(tokens[1:])
		}
	}
	return sugg
}

func execute_node_info_command(tokens []string) tea.Model {
	if len(tokens) == 1 {
		id, err := strconv.ParseInt(tokens[0], 0, 32)
		if err != nil {
			return NewErrorReplyModel(err)
		} else {
			return NewNodeInfoModel(id)
		}
	}
	return NewErrorReplyModel(errors.New("node info: missing node ID"))
}

func execute_node_command(tokens []string) tea.Model {
	if len(tokens) == 0 {

	} else {
		token := tokens[0]
		tokens = tokens[1:]
		if token == "info" {
			return execute_node_info_command(tokens)
		}
	}
	return NewErrorReplyModel(errors.New("node: unknow command"))
}

func execute_command(cmd string) tea.Model {
	tokens := strings.Split(cmd, " ")
	if len(tokens) > 0 {
		token := tokens[0]
		tokens = tokens[1:]
		if token == "coordinator" {
			return NewCoordinatorInfoModel()
		} else if token == "graph" {
			return NewGraphShowModel()
		} else if token == "node" {
			return execute_node_command(tokens)
		}
	}
	return nil
}
