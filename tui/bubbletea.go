package tui

import (
	"bytes"
	"context"
	"errors"
	"net"
	"os"
	"os/signal"
	"syscall"
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
var gpath *graph.GraphPath = nil

type termInfo struct {
	term    string
	profile string
	width   int
	height  int
	bg      string
}

func teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	pty, _, _ := s.Pty()
	renderer := bubbletea.MakeRenderer(s)
	txtStyle := renderer.NewStyle().Foreground(lipgloss.Color("10"))
	quitStyle := renderer.NewStyle().Foreground(lipgloss.Color("8"))

	bg := "light"
	if renderer.HasDarkBackground() {
		bg = "dark"
	}

	m := model{
		termInfo: termInfo{
			term:    pty.Term,
			profile: renderer.ColorProfile().Name(),
			width:   pty.Window.Width,
			height:  pty.Window.Height,
			bg:      bg,
		},
		txtStyle:    txtStyle,
		quitStyle:   quitStyle,
		textInput:   textinput.New(),
		headerTable: table.New(),
		submodel:    nil,
	}

	m.textInput.Placeholder = "Command"
	m.textInput.Focus()
	m.textInput.CharLimit = 128
	m.textInput.Width = 64
	m.textInput.TextStyle = txtStyle
	m.textInput.SetSuggestions(get_suggestions(""))
	m.textInput.ShowSuggestions = true

	return m, []tea.ProgramOption{tea.WithAltScreen()}
}

// Just a generic tea.Model to demo terminal information of ssh.
type model struct {
	termInfo    termInfo
	txtStyle    lipgloss.Style
	quitStyle   lipgloss.Style
	textInput   textinput.Model
	headerTable table.Model
	submodel    tea.Model
	err         error
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.termInfo.height = msg.Height
		m.termInfo.width = msg.Width
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeySpace:
			m.textInput.SetSuggestions(get_suggestions(m.textInput.Value()))
		case tea.KeyEnter:
			m.submodel = execute_command(m.textInput.Value())
			m.textInput.SetValue("")
			m.textInput.SetSuggestions(get_suggestions(""))
		}
	case error:
		m.err = msg
	}

	var cmds tea.BatchMsg
	m.textInput, cmd = m.textInput.Update(msg)
	cmds = append(cmds, cmd)
	if m.submodel != nil {
		m.submodel, cmd = m.submodel.Update(msg)
	}
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	var buffer bytes.Buffer
	buffer.WriteString("Command prompt: ")
	buffer.WriteString(m.textInput.Value())
	buffer.WriteString("\n\n")
	if m.submodel != nil {
		buffer.WriteString(m.submodel.View())
	}
	return buffer.String()
}

func TuiStart(host, port string, _gpath *graph.GraphPath, _sconn *meshmesh.SerialConnection) {
	gpath = _gpath
	sconn = _sconn

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

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	log.Info("Starting SSH server", "host", host, "port", port)
	go func() {
		if err = s.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			log.Error("Could not start server", "error", err)
			done <- nil
		}
	}()

	<-done
	log.Info("Stopping SSH server")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer func() { cancel() }()
	if err := s.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		log.Error("Could not stop server", "error", err)
	}
}
