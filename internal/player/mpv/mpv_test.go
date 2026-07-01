package mpv

import (
	"testing"

	"github.com/gustmrg/lofi/internal/player"
)

func TestClassifyEndFile(t *testing.T) {
	tests := []struct {
		name       string
		reason     string
		stderrTail string
		want       player.ErrorCategory
	}{
		{
			name:       "generic error stays unknown",
			reason:     "error",
			stderrTail: "",
			want:       player.ErrUnknown,
		},
		{
			name:       "audio output evidence",
			reason:     "error",
			stderrTail: "Could not open/initialize audio device",
			want:       player.ErrAudioOutput,
		},
		{
			name:       "network evidence",
			reason:     "error",
			stderrTail: "connection reset by peer",
			want:       player.ErrNetwork,
		},
		{
			name:       "decode evidence",
			reason:     "error",
			stderrTail: "ffmpeg decoder failed with invalid data",
			want:       player.ErrDecode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyEndFile(tt.reason, tt.stderrTail); got != tt.want {
				t.Fatalf("classifyEndFile() = %v, want %v", got, tt.want)
			}
		})
	}
}
