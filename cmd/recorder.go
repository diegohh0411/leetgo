package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"syscall"
)

// platformConfig holds platform-specific ffmpeg input flags.
type platformConfig struct {
	inputFormat string
	inputDevice string
	canPause    bool
}

// detectPlatform returns the ffmpeg input config for the current OS.
func detectPlatform() platformConfig {
	return detectPlatformFor(runtime.GOOS)
}

// detectPlatformFor returns the ffmpeg input config for a given GOOS.
func detectPlatformFor(goos string) platformConfig {
	switch goos {
	case "darwin":
		return platformConfig{inputFormat: "avfoundation", inputDevice: ":0", canPause: true}
	case "windows":
		return platformConfig{inputFormat: "dshow", inputDevice: "", canPause: false}
	default: // linux, freebsd, etc.
		return platformConfig{inputFormat: "pulse", inputDevice: "default", canPause: true}
	}
}

// buildArgs constructs the ffmpeg argument list for recording to outputPath.
func (pc platformConfig) buildArgs(outputPath string) []string {
	args := []string{"-f", pc.inputFormat}
	if pc.inputDevice != "" {
		args = append(args, "-i", pc.inputDevice)
	} else {
		// Windows dshow: let ffmpeg auto-detect the default audio device
		args = append(args, "-i", "audio")
	}
	args = append(args,
		"-c:a", "libmp3lame",
		"-q:a", "2",
		"-y",
		outputPath,
	)
	return args
}

// checkFFmpeg verifies that ffmpeg is available on PATH.
func checkFFmpeg() error {
	_, err := exec.LookPath("ffmpeg")
	if err != nil {
		return fmt.Errorf("ffmpeg is not installed or not on PATH.\n\nInstall it with:\n  macOS:   brew install ffmpeg\n  Linux:   sudo apt install ffmpeg\n  Windows: winget install ffmpeg\n  See: https://ffmpeg.org/download.html")
	}
	return nil
}

// startRecording spawns an ffmpeg subprocess that records audio to outputPath.
func startRecording(outputPath string) (*exec.Cmd, error) {
	pc := detectPlatform()
	args := pc.buildArgs(outputPath)

	cmd := exec.Command("ffmpeg", args...)
	cmd.Stderr = nil // suppress ffmpeg's verbose stderr; TUI handles errors
	return cmd, cmd.Start()
}

// stopRecording sends SIGINT to ffmpeg so it flushes and exits cleanly.
func stopRecording(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	if runtime.GOOS == "windows" {
		return cmd.Process.Kill()
	}
	_ = cmd.Process.Signal(syscall.SIGINT)
	return cmd.Wait()
}

// pauseRecording suspends the ffmpeg process (Unix only).
func pauseRecording(cmd *exec.Cmd) error {
	if runtime.GOOS == "windows" {
		return fmt.Errorf("pause is not supported on Windows")
	}
	return cmd.Process.Signal(syscall.SIGSTOP)
}

// resumeRecording resumes a paused ffmpeg process (Unix only).
func resumeRecording(cmd *exec.Cmd) error {
	if runtime.GOOS == "windows" {
		return fmt.Errorf("resume is not supported on Windows")
	}
	return cmd.Process.Signal(syscall.SIGCONT)
}

// cancelRecording kills ffmpeg and removes the partial output file.
func cancelRecording(cmd *exec.Cmd, outputPath string) {
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}
	_ = os.Remove(outputPath)
}
