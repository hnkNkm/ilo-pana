package commands

import (
	"os"
	"strings"
	"testing"
)

func TestConfirmClear(t *testing.T) {
	tests := []struct {
		name      string
		force     bool
		input     string
		wantClear bool
	}{
		{
			name:      "force_skips_prompt",
			force:     true,
			wantClear: true,
		},
		{
			name:      "lowercase_y_confirms",
			force:     false,
			input:     "y\n",
			wantClear: true,
		},
		{
			name:      "uppercase_Y_confirms",
			force:     false,
			input:     "Y\n",
			wantClear: true,
		},
		{
			name:      "n_cancels",
			force:     false,
			input:     "n\n",
			wantClear: false,
		},
		{
			name:      "empty_input_cancels",
			force:     false,
			input:     "\n",
			wantClear: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.input != "" {
				oldStdin := os.Stdin
				r, w, _ := os.Pipe()
				w.WriteString(tt.input)
				w.Close()
				os.Stdin = r
				defer func() { os.Stdin = oldStdin }()
			}

			// Capture stdout to avoid noise
			oldStdout := os.Stdout
			rOut, wOut, _ := os.Pipe()
			os.Stdout = wOut

			got := confirmClear("test-session", tt.force)

			wOut.Close()
			os.Stdout = oldStdout

			// Drain prompt output
			var sb strings.Builder
			buf := make([]byte, 1024)
			for {
				n, err := rOut.Read(buf)
				sb.Write(buf[:n])
				if err != nil {
					break
				}
			}

			if got != tt.wantClear {
				t.Errorf("confirmClear(%v) = %v, want %v", tt.force, got, tt.wantClear)
			}
		})
	}
}
