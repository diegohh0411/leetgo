package cmd

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os/exec"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	sampleRate   = 44100
	pcmChunkSize = 2048 // samples per read (matches FFT input size)
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

// Unicode block characters for EQ bars, from lowest to highest.
var blockChars = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// recorderModel is the bubbletea model for the audio recording TUI.
type recorderModel struct {
	outDir     string
	filename   string
	outputPath string
	cmd        *exec.Cmd
	pipe       io.Reader
	status     recorderStatus
	startTime  time.Time
	elapsed    time.Duration
	err        error
	canPause   bool
	bands      []float64 // frequency band levels, 0.0–1.0
	width      int       // terminal width in columns
}

// tickMsg is sent every 100ms to update the timer display.
type tickMsg time.Time

// errMsg wraps an error from ffmpeg.
type errMsg error

// bandsMsg carries frequency band levels from the PCM reader.
type bandsMsg []float64

// numBandsForWidth returns how many EQ bands fit the given terminal width.
// Each band takes 2 chars (block + space), minus 1 for the last band.
func numBandsForWidth(w int) int {
	if w <= 0 {
		return 16
	}
	n := (w + 1) / 2
	if n < 4 {
		n = 4
	}
	return n
}

// runRecorderTUI launches the bubbletea recording interface.
func runRecorderTUI(outDir, filename, outputPath string) error {
	m := &recorderModel{
		outDir:     outDir,
		filename:   filename,
		outputPath: outputPath,
		canPause:   detectPlatform().canPause,
		width:      80, // default before first WindowSizeMsg
	}
	m.bands = make([]float64, numBandsForWidth(m.width))
	p := tea.NewProgram(m)
	_, err := p.Run()
	return err
}

func (m *recorderModel) Init() tea.Cmd {
	// Start the ffmpeg subprocess (writes MP3 + pipes raw PCM).
	cmd, pipe, err := startRecording(m.outputPath)
	if err != nil {
		return func() tea.Msg { return errMsg(err) }
	}
	m.cmd = cmd
	m.pipe = pipe
	m.startTime = time.Now()
	m.status = statusRecording
	return tea.Batch(tickCmd(), readPCMCmd(m.pipe, numBandsForWidth(m.width)))
}

func (m *recorderModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		n := numBandsForWidth(m.width)
		if len(m.bands) != n {
			m.bands = make([]float64, n)
		}
		return m, nil

	case tickMsg:
		if m.status == statusRecording {
			m.elapsed = time.Since(m.startTime)
		}
		if m.status == statusRecording || m.status == statusPaused {
			return m, tickCmd()
		}
		return m, nil

	case bandsMsg:
		m.bands = msg
		// Keep reading while recording or paused (pipe stays open).
		if m.status == statusRecording || m.status == statusPaused {
			return m, readPCMCmd(m.pipe, numBandsForWidth(m.width))
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
	switch m.status {
	case statusDone:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#5fff5f")).Render(
			fmt.Sprintf("✓ Saved %s (%s)", m.filename, formatDuration(m.elapsed)))
	case statusCancelled:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#ffaf00")).Render("Cancelled — partial file discarded.")
	case statusError:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#ff5f5f")).Render(
			fmt.Sprintf("Error: %v", m.err))
	}

	// Status indicator
	var indicator string
	var controls string
	switch m.status {
	case statusRecording:
		indicator = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff5f5f")).Bold(true).Render("⏺")
		controls = "[space] pause  [q] stop  [ctrl+c] cancel"
	case statusPaused:
		indicator = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffaf00")).Bold(true).Render("⏸")
		controls = "[space] resume  [q] stop  [ctrl+c] cancel"
	}

	// EQ bars — black & white, full terminal width
	eqLine := renderEQ(m.bands)

	return fmt.Sprintf("%s %s  Recording %s\n%s\n%s",
		indicator, formatDuration(m.elapsed), m.filename,
		eqLine,
		lipgloss.NewStyle().Faint(true).Render(controls))
}

// renderEQ renders frequency bands as plain block characters, full width.
func renderEQ(bands []float64) string {
	result := make([]byte, 0, len(bands)*2)
	for i, level := range bands {
		idx := int(level * float64(len(blockChars)-1))
		if idx < 0 {
			idx = 0
		}
		if idx >= len(blockChars) {
			idx = len(blockChars) - 1
		}
		result = append(result, string(blockChars[idx])...)
		if i < len(bands)-1 {
			result = append(result, ' ')
		}
	}
	return string(result)
}

// readPCMCmd reads a chunk of raw PCM from the pipe, runs FFT,
// and returns frequency band levels as a bandsMsg.
func readPCMCmd(pipe io.Reader, numBands int) tea.Cmd {
	return func() tea.Msg {
		buf := make([]byte, pcmChunkSize*2) // s16le = 2 bytes per sample
		_, err := io.ReadFull(pipe, buf)
		if err != nil {
			return nil // pipe closed, stop reading
		}

		// Convert s16le bytes to float64 samples
		samples := make([]float64, pcmChunkSize)
		for i := range samples {
			sample := int16(binary.LittleEndian.Uint16(buf[i*2 : i*2+2]))
			samples[i] = float64(sample) / 32768.0
		}

		return bandsMsg(analyzeBands(samples, numBands))
	}
}

// analyzeBands runs FFT on samples and groups frequency bins into n bands.
func analyzeBands(samples []float64, n int) []float64 {
	mags := magnitudeSpectrum(samples)

	bands := make([]float64, n)
	binHz := float64(sampleRate) / float64(len(samples))

	// Logarithmic band grouping over voice range (80 Hz – 8000 Hz)
	minFreq := 80.0
	maxFreq := 8000.0

	for i := 0; i < n; i++ {
		lo := minFreq * math.Pow(maxFreq/minFreq, float64(i)/float64(n))
		hi := minFreq * math.Pow(maxFreq/minFreq, float64(i+1)/float64(n))
		loBin := int(lo / binHz)
		hiBin := int(hi / binHz)
		if loBin < 0 {
			loBin = 0
		}
		if hiBin >= len(mags) {
			hiBin = len(mags) - 1
		}

		var sum float64
		count := 0
		for b := loBin; b <= hiBin; b++ {
			sum += mags[b]
			count++
		}
		if count > 0 {
			bands[i] = sum / float64(count)
		}
	}

	// Normalize relative to current frame max
	maxBand := 0.001
	for _, b := range bands {
		if b > maxBand {
			maxBand = b
		}
	}
	for i := range bands {
		bands[i] = math.Min(bands[i]/maxBand, 1.0)
	}

	return bands
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
