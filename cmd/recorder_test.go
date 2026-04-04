package cmd

import (
	"runtime"
	"testing"
)

func TestDetectPlatform(t *testing.T) {
	pc := detectPlatform()

	// Every platform must set these
	if pc.inputFormat == "" {
		t.Error("inputFormat must not be empty")
	}
	if pc.inputDevice == "" {
		t.Error("inputDevice must not be empty")
	}
	if pc.canPause && runtime.GOOS == "windows" {
		t.Error("windows should not report canPause=true")
	}
}

func TestDetectPlatformKnownOS(t *testing.T) {
	tests := []struct {
		goos         string
		wantFormat   string
		wantDevice   string
		wantCanPause bool
	}{
		{"darwin", "avfoundation", ":0", true},
		{"linux", "pulse", "default", true},
		{"windows", "dshow", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.goos, func(t *testing.T) {
			pc := detectPlatformFor(tt.goos)
			if pc.inputFormat != tt.wantFormat {
				t.Errorf("inputFormat = %q, want %q", pc.inputFormat, tt.wantFormat)
			}
			if tt.goos != "windows" && pc.inputDevice != tt.wantDevice {
				t.Errorf("inputDevice = %q, want %q", pc.inputDevice, tt.wantDevice)
			}
			if pc.canPause != tt.wantCanPause {
				t.Errorf("canPause = %v, want %v", pc.canPause, tt.wantCanPause)
			}
		})
	}
}

func TestBuildFFmpegArgs(t *testing.T) {
	pc := platformConfig{inputFormat: "avfoundation", inputDevice: ":0"}
	args := pc.buildArgs("/tmp/attempt-1.mp3")

	want := []string{
		"-f", "avfoundation",
		"-i", ":0",
		"-c:a", "libmp3lame",
		"-q:a", "2",
		"-y",
		"/tmp/attempt-1.mp3",
	}
	if len(args) != len(want) {
		t.Fatalf("args length = %d, want %d\ngot:  %v\nwant: %v", len(args), len(want), args, want)
	}
	for i := range args {
		if args[i] != want[i] {
			t.Errorf("args[%d] = %q, want %q", i, args[i], want[i])
		}
	}
}

func TestBuildFFmpegArgsWindows(t *testing.T) {
	pc := platformConfig{inputFormat: "dshow", inputDevice: "", canPause: false}
	args := pc.buildArgs("/tmp/attempt-1.mp3")

	// Windows dshow uses -i audio=Microphone (device auto-detected at runtime)
	// Without a specific device, it should just use -i with no device name
	found := false
	for _, a := range args {
		if a == "-f" {
			found = true
		}
	}
	if !found {
		t.Error("args should contain -f flag")
	}
}
