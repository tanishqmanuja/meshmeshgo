package tui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/erikgeiser/promptkit/confirmation"
	"leguru.net/m/v2/config"
	"leguru.net/m/v2/graph"
	"leguru.net/m/v2/meshmesh"
)

const firmwareMaxRetries = 3

type firmwareErrorMsg error
type firmwareInitDoneMsg string
type firmwareUploadProgressMsg float64
type firmwareUploadDoneMsg bool
type firmwareRebootDoneMsg bool
type firmwareCheckRevAfterMsg string

// Firmware state machine
const (
	firmwareGetDevice = iota
	firmwareGetProtocol
	firmwareCheckNode
	firmwareStatePickFile
	firmwareStateConfirmUpload
	firmwareStateUploading
	firmwareStateUploadSuccess
	firmwareStateUploadFailed
	firmwareStateRebooting
	firmwareStateRebootSuccess
	firmwareCheckRevAfter
)

func initFirmwareCmd(m *FirmwareModel) tea.Cmd {
	m.procedure = meshmesh.NewFirmwareUploadProcedure(sconn, gpath, meshmesh.MeshNodeId(m.device.ID()))

	return func() tea.Msg {
		rep, err := sconn.SendReceiveApiProt(meshmesh.FirmRevApiRequest{}, meshmesh.UnicastProtocol, meshmesh.MeshNodeId(m.device.ID()))
		if err != nil {
			return firmwareErrorMsg(errors.Join(
				errors.New("firmware revision request failed"),
				fmt.Errorf("node id: %s", graph.FmtDeviceId(m.device)),
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
			m.retries += 1
			if m.retries > firmwareMaxRetries {
				m.warning = nil
				return firmwareErrorMsg(err1)
			}
			m.warning = err1
			return firmwareUploadProgressMsg(0)
		}
		if err2 != nil {
			return firmwareErrorMsg(err2)
		}
		m.warning = nil
		m.retries = 0
		return firmwareUploadProgressMsg(m.procedure.Percent())
	}
}

func rebootNodeCmd(m *FirmwareModel) tea.Cmd {
	return func() tea.Msg {
		_, err := sconn.SendReceiveApiProt(meshmesh.NodeRebootApiRequest{}, meshmesh.UnicastProtocol, meshmesh.MeshNodeId(m.device.ID()))
		if err != nil {
			return firmwareErrorMsg(errors.Join(errors.New("reboot request failed"), err))
		}
		time.Sleep(10 * time.Second)
		return firmwareRebootDoneMsg(true)
	}
}

func finalizeNodeCmd(m *FirmwareModel) tea.Cmd {
	return func() tea.Msg {
		rep, err := sconn.SendReceiveApiProt(meshmesh.FirmRevApiRequest{}, meshmesh.UnicastProtocol, meshmesh.MeshNodeId(m.device.ID()))
		if err != nil {
			return firmwareErrorMsg(errors.Join(
				errors.New("firmware revision request failed"),
				fmt.Errorf("node id: %s", graph.FmtDeviceId(m.device)),
				err))
		}
		rev := rep.(meshmesh.FirmRevApiReply)
		return firmwareCheckRevAfterMsg(rev.Revision)
	}
}

type FirmwareModel struct {
	BaseModel
	fpick      filepicker.Model
	confirm    *confirmation.Model
	progress   progress.Model
	confirm2   *confirmation.Model
	focused    int
	file       string
	err        error
	warning    error
	currentRev string
	afterRev   string
	procedure  *meshmesh.FirmwareUploadProcedure
	state      int
	retries    int
}

func (m *FirmwareModel) Init() tea.Cmd {
	return tea.Batch([]tea.Cmd{m.initDeviceSelection(), m.initProtocolSelection(), m.initSpinner(), m.fpick.Init(), m.confirm.Init(), m.confirm2.Init()}...)
}

func (m *FirmwareModel) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case deviceItemSelectedMsg:
		config.SetINIValue("firmware", "device", graph.FmtDeviceId(m.device))
		m.state = firmwareGetProtocol
	case protocolSelectedMsg:
		m.protocol = meshmesh.FindBestProtocolOverride(meshmesh.MeshNodeId(m.device.ID()), meshmesh.MeshProtocol(msg))
		cmd = initFirmwareCmd(m)
		cmds = append(cmds, cmd)
	case firmwareInitDoneMsg:
		m.currentRev = string(msg)
		m.fpick.CurrentDirectory = config.GetINIValue("firmware", "path")
		if m.fpick.CurrentDirectory == "" {
			m.fpick.CurrentDirectory, _ = os.Getwd()
		}
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
		cmd = finalizeNodeCmd(m)
		cmds = append(cmds, cmd)
	case firmwareCheckRevAfterMsg:
		m.state = firmwareCheckRevAfter
		m.afterRev = string(msg)
	case firmwareErrorMsg:
		m.err = msg
	}

	if m.err != nil {
		return m, cmd
	}

	cmd = m.updateSpinner(msg)
	cmds = append(cmds, cmd)

	switch m.state {
	case firmwareGetDevice:
		cmd = m.updateDeviceSelection(msg)
		cmds = append(cmds, cmd)
	case firmwareGetProtocol:
		cmd = m.updateProtocolSelection(msg)
		cmds = append(cmds, cmd)
	case firmwareStatePickFile:
		m.fpick, cmd = m.fpick.Update(msg)
		cmds = append(cmds, cmd)
		if didSelect, path := m.fpick.DidSelectFile(msg); didSelect {
			m.file = path
			m.procedure.Init(m.file)
			m.state = firmwareStateConfirmUpload
			config.SetINIValue("firmware", "path", filepath.Dir(m.file))
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

	if m.state >= firmwareGetDevice {
		views = append(views, m.viewDeviceSelection())
	}

	if m.state >= firmwareGetProtocol {
		views = append(views, m.viewProtocolSelection())
	}

	if m.state >= firmwareCheckNode {
		views = append(views, "Firmware upload to node "+graph.FmtDeviceId(m.device))
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
		if m.state == firmwareStateUploading {
			_retries := ""
			if m.retries > 0 {
				_retries = fmt.Sprintf(" (%d retries)", m.retries)
			}
			views = append(views, m.ti.progressStyle.Render(fmt.Sprintf("Uploading firmware in progress: %d/%d bytes %s", m.procedure.BytesSent(), m.procedure.BytesTotal(), _retries)))
		} else {
			views = append(views, m.ti.successStyle.Render(fmt.Sprintf("Uploading firmware successful, sent %d bytes", m.procedure.BytesSent())))
		}
		views = append(views, m.progress.ViewAs(m.procedure.Percent()))
	}

	if m.state >= firmwareStateUploadSuccess {
		views = append(views, m.confirm2.View())
	}

	if m.state == firmwareStateRebooting {
		views = append(views, m.ti.progressStyle.Render(fmt.Sprintf("%s Node rebooting in progress...", m.viewSpinner())))
	}

	if m.state >= firmwareStateRebootSuccess {
		views = append(views, m.ti.successStyle.Render("Node reboot successful, procedure terminated."))
	}

	if m.state >= firmwareCheckRevAfter {
		views = append(views, "After revision: "+m.afterRev)
	}

	if m.warning != nil {
		views = append(views, m.ti.warningStyle.Render(m.warning.Error()))
	}

	if m.err != nil {
		views = append(views, m.ti.errorStyle.Render(m.err.Error()))
	}

	return lipgloss.JoinVertical(lipgloss.Top, views...)
}

func (m *FirmwareModel) Focused() bool {
	return true
}

func NewFirmwareModel(ti termInfo) Model {
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

	confirm := confirmation.New("Are you sure you want to upload firmware to selected node ?", confirmation.Undecided)
	progress := progress.New(progress.WithGradient("#FF0000", "#00FF00"))
	confirm2 := confirmation.New("Upload firmware successful. Reboot node?", confirmation.Undecided)

	model := &FirmwareModel{
		BaseModel: NewBaseModelExtended(ti, gpath),
		fpick:     fpick,
		confirm:   confirmation.NewModel(confirm),
		focused:   0,
		procedure: nil,
		state:     firmwareGetDevice,
		progress:  progress,
		confirm2:  confirmation.NewModel(confirm2),
	}

	model.createSpinner()
	return model
}
