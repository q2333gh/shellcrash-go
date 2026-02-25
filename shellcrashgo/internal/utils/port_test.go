package utils

import (
	"net"
	"testing"
)

func TestCheckPort(t *testing.T) {
	tests := []struct {
		name      string
		port      int
		usedPorts []int
		wantErr   bool
	}{
		{
			name:      "valid port",
			port:      8080,
			usedPorts: []int{9090, 9091},
			wantErr:   false,
		},
		{
			name:      "port too low",
			port:      0,
			usedPorts: []int{},
			wantErr:   true,
		},
		{
			name:      "port too high",
			port:      65536,
			usedPorts: []int{},
			wantErr:   true,
		},
		{
			name:      "port conflicts with used ports",
			port:      8080,
			usedPorts: []int{8080, 9090},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckPort(tt.port, tt.usedPorts)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckPort() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCheckPortInUse(t *testing.T) {
	// Bind a port to test detection
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to bind test port: %v", err)
	}
	defer listener.Close()

	boundPort := listener.Addr().(*net.TCPAddr).Port

	// Test that the bound port is detected as in use
	err = CheckPort(boundPort, []int{})
	if err == nil {
		t.Errorf("CheckPort() should detect port %d as in use", boundPort)
	}
}

func TestParsePort(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		{
			name:    "valid port",
			input:   "8080",
			want:    8080,
			wantErr: false,
		},
		{
			name:    "valid port with whitespace",
			input:   "  9090  ",
			want:    9090,
			wantErr: false,
		},
		{
			name:    "invalid port - not a number",
			input:   "abc",
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid port - empty",
			input:   "",
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePort(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParsePort() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParsePort() = %v, want %v", got, tt.want)
			}
		})
	}
}
