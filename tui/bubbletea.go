package tui

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
	"leguru.net/m/v2/graph"
	"leguru.net/m/v2/meshmesh"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
)

var sconn *meshmesh.SerialConnection
var esphome *meshmesh.MultiServerApi
var gpath *graph.Network = nil

type Model interface {
	Init() tea.Cmd
	Update(tea.Msg) (Model, tea.Cmd)
	View() string
	Focused() bool
}

func teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	pty, _, _ := s.Pty()

	renderer := bubbletea.MakeRenderer(s)
	m := model{
		ti: termInfo{
			term:          pty.Term,
			width:         pty.Window.Width,
			height:        pty.Window.Height,
			renderer:      renderer,
			errorStyle:    renderer.NewStyle().Foreground(lipgloss.ANSIColor(9)),
			warningStyle:  renderer.NewStyle().Foreground(lipgloss.ANSIColor(13)),
			successStyle:  renderer.NewStyle().Foreground(lipgloss.ANSIColor(10)),
			progressStyle: renderer.NewStyle().Foreground(lipgloss.ANSIColor(11)),
		},
		textInput:   textinput.New(),
		help:        createHelpModel(renderer),
		headerTable: table.New(),
		submodel:    nil,
		state:       bubbleWaitCommandState,
		keymap:      GetKeymap(),
	}

	log.Printf("Terminal %s Color profile %s", pty.Term, m.ti.renderer.ColorProfile().Name())

	m.textInput.Prompt = "Command> "
	m.textInput.Placeholder = "<write command here>"
	m.textInput.Focus()
	m.textInput.CharLimit = 128
	m.textInput.Width = 64
	m.textInput.SetSuggestions(get_suggestions(""))
	m.textInput.ShowSuggestions = true

	m.textInput.PromptStyle = m.ti.renderer.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(32))
	m.textInput.PlaceholderStyle = m.ti.renderer.NewStyle().Foreground(lipgloss.ANSIColor(8))
	m.textInput.TextStyle = m.ti.renderer.NewStyle()
	m.textInput.Cursor.TextStyle = m.ti.renderer.NewStyle()
	m.textInput.Cursor.Style = m.ti.renderer.NewStyle()

	return m, []tea.ProgramOption{tea.WithAltScreen()}
}

type bubbleModuleStartedMsg string
type bubbleModuleDoneMsg bool

func ExecuteModuleCmd(m model, command string) tea.Cmd {
	return func() tea.Msg {
		return bubbleModuleStartedMsg(command)
	}
}

func TerminateModuleCmd(m *model) tea.Cmd {
	return func() tea.Msg {
		m.submodel = nil
		return bubbleModuleDoneMsg(true)
	}
}

const (
	bubbleWaitCommandState = iota
	bubbleWaitProcedureState
)

// Just a generic tea.Model to demo terminal information of ssh.
type model struct {
	ti          termInfo
	textInput   textinput.Model
	help        help.Model
	headerTable table.Model
	submodel    Model
	err         error
	state       int
	keymap      keymap
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds tea.BatchMsg

	switch msg.(type) {
	case bubbleModuleStartedMsg:
		m.textInput.Blur()
		m.err = nil
		m.state = bubbleWaitProcedureState
	case bubbleModuleDoneMsg:
		m.textInput.SetValue("")
		m.textInput.SetSuggestions(get_suggestions(""))
		m.textInput.Focus()
		cmds = append(cmds, textinput.Blink)
		m.err = nil
		m.submodel = nil
		m.state = bubbleWaitCommandState
	}

	switch m.state {
	case bubbleWaitCommandState:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.Type {
			case tea.KeyEnter:
				m.submodel = m.execute_command(m.textInput.Value())
				if m.submodel != nil {
					cmds = append(cmds, m.submodel.Init())
					cmds = append(cmds, ExecuteModuleCmd(m, m.textInput.Value()))
				} else {
					m.err = errors.New("command '" + m.textInput.Value() + "' not found")
				}
			case tea.KeySpace:
				m.textInput.SetSuggestions(get_suggestions(m.textInput.Value()))
			}
		}
	case bubbleWaitProcedureState:
		if m.submodel == nil {
			log.Error("Submodel is nil")
			return m, tea.Quit
		}
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.Type == tea.KeyEsc {
				return m, TerminateModuleCmd(&m)
			}
		}
		m.submodel, cmd = m.submodel.Update(msg)
		cmds = append(cmds, cmd)
	}

	m.textInput, cmd = m.textInput.Update(msg)
	cmds = append(cmds, cmd)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.ti.height = msg.Height
		m.ti.width = msg.Width
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		}
	case error:
		m.err = msg
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	views := []string{}
	views = append(views, m.textInput.View())
	if m.state == bubbleWaitCommandState {
		views = append(views, m.help.View(m.keymap))
	}
	if m.state == bubbleWaitProcedureState {
		views = append(views, m.submodel.View())
	}
	if m.err != nil {
		views = append(views, m.ti.errorStyle.Render(m.err.Error()))
	}
	return lipgloss.JoinVertical(lipgloss.Top, views...)
}

func ShutdownSshServer(s *ssh.Server) {
	log.Info("Stopping SSH server")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() { cancel() }()
	if err := s.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		log.Error("Could not stop server", "error", err)
	}
}

func NewSshServer(host, port string, _gpath *graph.Network, _sconn *meshmesh.SerialConnection, _esphome *meshmesh.MultiServerApi) (*ssh.Server, error) {
	gpath = _gpath
	sconn = _sconn
	esphome = _esphome

	s, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(host, port)),
		wish.WithHostKeyPath(".ssh/id_ed25519"),
		wish.WithMiddleware(
			bubbletea.Middleware(teaHandler),
			activeterm.Middleware(),
			logging.Middleware(),
		),
	)

	if err != nil {
		log.Error("Could not start server", "error", err)
	}

	log.Info("Starting SSH server", "host", host, "port", port)

	go func() {
		if err = s.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			log.Error("Could not start server", "error", err)
		}
	}()

	return s, err
}
