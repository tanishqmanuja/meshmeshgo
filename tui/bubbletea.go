package tui

import (
	"bytes"
	"context"
	"errors"
	"net"
	"time"

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

	m := model{
		ti: termInfo{
			term:     pty.Term,
			width:    pty.Window.Width,
			height:   pty.Window.Height,
			renderer: bubbletea.MakeRenderer(s),
		},
		textInput:   textinput.New(),
		headerTable: table.New(),
		submodel:    nil,
	}

	log.Printf("Terminal %s Color profile %s", pty.Term, m.ti.renderer.ColorProfile().Name())

	m.textInput.Prompt = "Command> "
	m.textInput.Placeholder = "<write command here>"
	m.textInput.Focus()
	m.textInput.CharLimit = 128
	m.textInput.Width = 64
	m.textInput.SetSuggestions(get_suggestions(""))
	//m.textInput.ShowSuggestions = true

	m.textInput.PromptStyle = m.ti.renderer.NewStyle().Bold(true).Foreground(lipgloss.ANSIColor(32))
	m.textInput.PlaceholderStyle = m.ti.renderer.NewStyle().Foreground(lipgloss.ANSIColor(8))
	m.textInput.TextStyle = m.ti.renderer.NewStyle()
	m.textInput.Cursor.TextStyle = m.ti.renderer.NewStyle()
	m.textInput.Cursor.Style = m.ti.renderer.NewStyle()

	return m, []tea.ProgramOption{tea.WithAltScreen()}
}

// Just a generic tea.Model to demo terminal information of ssh.
type model struct {
	ti          termInfo
	textInput   textinput.Model
	headerTable table.Model
	submodel    Model
	err         error
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds tea.BatchMsg

	m.textInput, cmd = m.textInput.Update(msg)
	cmds = append(cmds, cmd)

	if m.submodel != nil {
		m.submodel, cmd = m.submodel.Update(msg)
		cmds = append(cmds, cmd)
	}

	if m.submodel != nil {
		// Add or remove focus to command line
		if m.submodel.Focused() && m.textInput.Focused() {
			m.textInput.Blur()
		}
		if !m.submodel.Focused() && !m.textInput.Focused() {
			m.textInput.Focus()
		}
	} else {
		if !m.textInput.Focused() {
			m.textInput.Focus()
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.ti.height = msg.Height
		m.ti.width = msg.Width
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyEsc:
			if m.submodel != nil && m.submodel.Focused() {
				m.submodel = nil
			} else {
				return m, tea.Quit
			}
		case tea.KeySpace:
			if m.submodel == nil || !m.submodel.Focused() {
				m.textInput.SetSuggestions(get_suggestions(m.textInput.Value()))
			}
		case tea.KeyEnter:
			if m.submodel == nil || !m.submodel.Focused() {
				m.submodel = m.execute_command(m.textInput.Value())
				if m.submodel != nil {
					cmd = m.submodel.Init()
					cmds = append(cmds, cmd)
				}
				m.textInput.SetValue("")
				m.textInput.SetSuggestions(get_suggestions(""))
			}
		}
	case error:
		m.err = msg
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	var buffer bytes.Buffer
	buffer.WriteString(m.textInput.View())
	buffer.WriteString("\n\n")
	if m.submodel != nil {
		buffer.WriteString(m.submodel.View())
	}
	return buffer.String()
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
