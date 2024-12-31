package tui

import (
	"errors"
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/erikgeiser/promptkit/confirmation"
	"leguru.net/m/v2/meshmesh"
	"leguru.net/m/v2/utils"
)

type firmwareErrorMsg error
type firmwareInitDoneMsg string
type firmwareUploadProgressMsg float64
type firmwareUploadDoneMsg bool
type firmwareRebootDoneMsg bool

const (
	firmwareCheckNode = iota
	firmwareStatePickFile
	firmwareStateConfirmUpload
	firmwareStateUploading
	firmwareStateUploadSuccess
	firmwareStateUploadFailed
	firmwareStateRebooting
	firmwareStateRebootSuccess
)

func initFirmwareCmd(m *FirmwareModel) tea.Cmd {
	return func() tea.Msg {
		rep, err := sconn.SendReceiveApiProt(meshmesh.FirmRevApiRequest{}, meshmesh.UnicastProtocol, meshmesh.MeshNodeId(m.nodeid))
		if err != nil {
			return firmwareErrorMsg(errors.Join(
				errors.New("firmware revision request failed"),
				fmt.Errorf("node id: %s", utils.FmtNodeId(m.nodeid)),
				err))
		}
		rev := rep.(meshmesh.FirmRevApiReply)
		return firmwareInitDoneMsg(rev.Revision)
	}
}

func uploadStepFirmwareCmd(m *FirmwareModel) tea.Cmd {
	return func() tea.Msg {
		done, err1, err2 := m.procedure.Step()
		if done {
			return firmwareUploadDoneMsg(true)
		}
		if err1 != nil {
			return firmwareErrorMsg(err1)
		}
		if err2 != nil {
			return firmwareErrorMsg(err2)
		}
		return firmwareUploadProgressMsg(m.procedure.Percent())
	}
}

func rebootNodeCmd(m *FirmwareModel) tea.Cmd {
	return func() tea.Msg {
		_, err := sconn.SendReceiveApiProt(meshmesh.NodeRebootApiRequest{}, meshmesh.UnicastProtocol, meshmesh.MeshNodeId(m.nodeid))
		if err != nil {
			return firmwareErrorMsg(errors.Join(errors.New("reboot request failed"), err))
		}
		return firmwareRebootDoneMsg(true)
	}
}

type FirmwareModel struct {
	ti           termInfo
	fpick        filepicker.Model
	confirm      *confirmation.Model
	progress     progress.Model
	confirm2     *confirmation.Model
	focused      int
	file         string
	err          error
	nodeid       int64
	currentRev   string
	successStyle lipgloss.Style
	errorStyle   lipgloss.Style
	procedure    *meshmesh.FirmwareUploadProcedure
	state        int
}

func (m *FirmwareModel) Init() tea.Cmd {
	m.procedure = meshmesh.NewFirmwareUploadProcedure(sconn, gpath, meshmesh.MeshNodeId(m.nodeid))
	return tea.Batch([]tea.Cmd{m.fpick.Init(), m.confirm.Init(), m.confirm2.Init(), initFirmwareCmd(m)}...)
}

func (m *FirmwareModel) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case firmwareInitDoneMsg:
		m.currentRev = string(msg)
		cmd = m.fpick.Init()
		cmds = append(cmds, cmd)
		m.state = firmwareStatePickFile
	case firmwareUploadProgressMsg:
		m.progress.SetPercent(float64(msg))
		cmd = uploadStepFirmwareCmd(m)
		cmds = append(cmds, cmd)
	case firmwareUploadDoneMsg:
		if bool(msg) {
			m.state = firmwareStateUploadSuccess
		}
	case firmwareRebootDoneMsg:
		m.state = firmwareStateRebootSuccess
	case firmwareErrorMsg:
		m.err = msg
	}

	if m.err != nil {
		return m, cmd
	}

	switch m.state {
	case firmwareStatePickFile:
		m.fpick, cmd = m.fpick.Update(msg)
		cmds = append(cmds, cmd)
		if didSelect, path := m.fpick.DidSelectFile(msg); didSelect {
			m.file = path
			m.procedure.Init(m.file)
			m.state = firmwareStateConfirmUpload
		}

		if didSelect, path := m.fpick.DidSelectDisabledFile(msg); didSelect {
			m.err = errors.New(path + " is not valid.")
		}
	case firmwareStateConfirmUpload:
		_, cmd = m.confirm.Update(msg)
		if cmd != nil {
			msg := cmd()
			switch msg.(type) {
			case tea.QuitMsg:
				confirm, err := m.confirm.Value()
				if err != nil {
					m.err = err
				} else {
					if confirm {
						m.state = firmwareStateUploading
						cmd = uploadStepFirmwareCmd(m)
						cmds = append(cmds, cmd)
					} else {
						m.state = firmwareStateUploadFailed
					}
				}
			}
		}
	case firmwareStateUploadSuccess:
		_, cmd = m.confirm2.Update(msg)
		if cmd != nil {
			msg := cmd()
			switch msg.(type) {
			case tea.QuitMsg:
				confirm, err := m.confirm2.Value()
				if err != nil {
					m.err = err
				} else {
					if confirm {
						m.state = firmwareStateRebooting
						cmd = rebootNodeCmd(m)
						cmds = append(cmds, cmd)
					}
				}
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *FirmwareModel) View() string {
	views := []string{}
	views = append(views, "Firmware upload to node "+utils.FmtNodeId(m.nodeid))
	if m.state >= firmwareCheckNode {
		views = append(views, "Current revision: "+m.currentRev)
	}

	if m.state == firmwareStatePickFile {
		views = append(views, "Select a firmware file to upload")
		views = append(views, m.fpick.View())
	} else if m.state >= firmwareStateConfirmUpload {
		views = append(views, "Selected file: "+m.file)
		views = append(views, m.confirm.View())
	}

	if m.state >= firmwareStateUploading {
		views = append(views, fmt.Sprintf("Uploading firmware: %d/%d bytes", m.procedure.BytesSent(), m.procedure.BytesTotal()))
		views = append(views, m.progress.ViewAs(m.procedure.Percent()))
	}

	if m.state >= firmwareStateUploadSuccess {
		views = append(views, m.confirm2.View())
	}

	if m.state >= firmwareStateRebootSuccess {
		views = append(views, m.successStyle.Render("Node reboot successful, procedure terminated."))
	}

	if m.err != nil {
		views = append(views, m.errorStyle.Render(m.err.Error()))
	}

	return lipgloss.JoinVertical(lipgloss.Top, views...)
}

func (m *FirmwareModel) Focused() bool {
	return true
}

func NewFirmwareModel(ti termInfo, nodeid int64) Model {
	var err error
	fpick := filepicker.New()
	fpick.AllowedTypes = []string{".bin"}
	fpick.CurrentDirectory, err = os.Getwd()
	fpick.AutoHeight = false
	fpick.Height = 15
	fpick.ShowHidden = true
	if err != nil {
		fpick.CurrentDirectory = "/"
	}
	fpick.Styles = filepicker.DefaultStylesWithRenderer(ti.renderer)

	confirm := confirmation.New("Are you sure you want to upload firmware to node "+utils.FmtNodeId(nodeid)+"?", confirmation.Undecided)
	progress := progress.New(progress.WithGradient("#FF0000", "#00FF00"))
	confirm2 := confirmation.New("Upload firmware successful. Reboot node?", confirmation.Undecided)

	return &FirmwareModel{
		ti:           ti,
		fpick:        fpick,
		confirm:      confirmation.NewModel(confirm),
		focused:      0,
		nodeid:       nodeid,
		errorStyle:   ti.renderer.NewStyle().Foreground(lipgloss.ANSIColor(9)),
		successStyle: ti.renderer.NewStyle().Foreground(lipgloss.ANSIColor(10)),
		procedure:    nil,
		state:        firmwareCheckNode,
		progress:     progress,
		confirm2:     confirmation.NewModel(confirm2),
	}
}
