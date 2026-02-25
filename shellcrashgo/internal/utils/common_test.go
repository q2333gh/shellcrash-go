package utils

import (
	"testing"
)

func TestURLEncode(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple string",
			input: "hello world",
			want:  "hello+world",
		},
		{
			name:  "special characters",
			input: "hello@world.com",
			want:  "hello%40world.com",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := URLEncode(tt.input)
			if got != tt.want {
				t.Errorf("URLEncode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestURLDecode(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "simple encoded string",
			input:   "hello+world",
			want:    "hello world",
			wantErr: false,
		},
		{
			name:    "special characters",
			input:   "hello%40world.com",
			want:    "hello@world.com",
			wantErr: false,
		},
		{
			name:    "empty string",
			input:   "",
			want:    "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := URLDecode(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("URLDecode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("URLDecode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCommandExists(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
		want bool
	}{
		{
			name: "existing command - sh",
			cmd:  "sh",
			want: true,
		},
		{
			name: "non-existing command",
			cmd:  "this_command_definitely_does_not_exist_12345",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CommandExists(tt.cmd)
			if got != tt.want {
				t.Errorf("CommandExists() = %v, want %v", got, tt.want)
			}
		})
	}
}
