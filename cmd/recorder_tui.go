package cmd

import (
	"fmt"
	"os/exec"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type recorderStatus int

const (
	statusRecording recorderStatus = iota
	statusPaused
	statusStopping
	statusDone
	statusCancelled
	statusError
)

// recorderModel is the bubbletea model for the audio recording TUI.
type recorderModel struct {
	outDir     string
	filename   string
	outputPath string
	cmd        *exec.Cmd
	status     recorderStatus
	startTime  time.Time
	elapsed    time.Duration
	err        error
	canPause   bool
}

// tickMsg is sent every 100ms to update the timer display.
type tickMsg time.Time

// errMsg wraps an error from ffmpeg.
type errMsg error

// runRecorderTUI launches the bubbletea recording interface.
func runRecorderTUI(outDir, filename, outputPath string) error {
	m := &recorderModel{
		outDir:     outDir,
		filename:   filename,
		outputPath: outputPath,
		canPause:   detectPlatform().canPause,
	}
	p := tea.NewProgram(m)
	_, err := p.Run()
	return err
}

func (m *recorderModel) Init() tea.Cmd {
	// Start the ffmpeg subprocess.
	cmd, err := startRecording(m.outputPath)
	if err != nil {
		return func() tea.Msg { return errMsg(err) }
	}
	m.cmd = cmd
	m.startTime = time.Now()
	m.status = statusRecording
	return tickCmd()
}

func (m *recorderModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		if m.status == statusRecording {
			m.elapsed = time.Since(m.startTime)
		}
		if m.status == statusRecording || m.status == statusPaused {
			return m, tickCmd()
		}
		return m, nil

	case errMsg:
		m.status = statusError
		m.err = msg
		return m, tea.Quit

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "enter":
			// Stop recording and save.
			m.status = statusStopping
			go func() { _ = stopRecording(m.cmd) }()
			m.status = statusDone
			return m, tea.Quit

		case " ":
			if !m.canPause {
				return m, nil
			}
			if m.status == statusRecording {
				_ = pauseRecording(m.cmd)
				m.status = statusPaused
			} else if m.status == statusPaused {
				_ = resumeRecording(m.cmd)
				m.status = statusRecording
				m.startTime = time.Now().Add(-m.elapsed)
			}

		case "ctrl+c":
			m.status = statusCancelled
			cancelRecording(m.cmd, m.outputPath)
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m *recorderModel) View() string {
	width := 60

	var statusLine string
	switch m.status {
	case statusRecording:
		indicator := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff5f5f")).Bold(true).Render("⏺")
		controls := "[space] pause  [q] stop  [ctrl+c] cancel"
		statusLine = fmt.Sprintf("%s %s  Recording %s  %s",
			indicator, formatDuration(m.elapsed), m.filename, controls)
	case statusPaused:
		indicator := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffaf00")).Bold(true).Render("⏸")
		controls := "[space] resume  [q] stop  [ctrl+c] cancel"
		statusLine = fmt.Sprintf("%s %s  Paused  %s",
			indicator, formatDuration(m.elapsed), controls)
	case statusDone:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#5fff5f")).Render(
			fmt.Sprintf("✓ Saved %s (%s)", m.filename, formatDuration(m.elapsed)))
	case statusCancelled:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#ffaf00")).Render("Cancelled — partial file discarded.")
	case statusError:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#ff5f5f")).Render(
			fmt.Sprintf("Error: %v", m.err))
	}

	return fmt.Sprintf("Recording to: %s\n%s", m.filename,
		lipgloss.NewStyle().Width(width).Render(statusLine))
}

// tickCmd returns a command that sends a tickMsg after 100ms.
func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// formatDuration formats a duration as MM:SS.
func formatDuration(d time.Duration) string {
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d", m, s)
}
