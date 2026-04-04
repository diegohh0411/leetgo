package cmd

import (
	"fmt"
	"io"
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

// buildVizArgs constructs ffmpeg args that write MP3 to file AND pipe raw PCM to stdout.
func (pc platformConfig) buildVizArgs(outputPath string) []string {
	args := []string{"-f", pc.inputFormat}
	if pc.inputDevice != "" {
		args = append(args, "-i", pc.inputDevice)
	} else {
		args = append(args, "-i", "audio")
	}
	// Split audio: stream [a] → MP3 file, stream [b] → raw PCM to stdout
	args = append(args,
		"-filter_complex", "[0:a]asplit=2[a][b]",
		"-map", "[a]", "-c:a", "libmp3lame", "-q:a", "2", "-y", outputPath,
		"-map", "[b]", "-f", "s16le", "-ac", "1", "-ar", "44100", "pipe:1",
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

// startRecording spawns ffmpeg that writes MP3 to file and pipes raw PCM to stdout.
func startRecording(outputPath string) (*exec.Cmd, io.Reader, error) {
	pc := detectPlatform()
	args := pc.buildVizArgs(outputPath)

	cmd := exec.Command("ffmpeg", args...)
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}
	return cmd, pipe, nil
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
