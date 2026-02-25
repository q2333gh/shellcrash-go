package utils

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

// CheckPort validates a port number and checks for conflicts with existing ports
// and whether it's already in use by another process.
// Returns an error if the port is invalid, conflicts with usedPorts, or is already bound.
func CheckPort(port int, usedPorts []int) error {
	// Validate port range
	if port <= 1 || port > 65535 {
		return fmt.Errorf("invalid port: must be between 1 and 65535")
	}

	// Check for conflicts with already-used ports
	for _, used := range usedPorts {
		if port == used {
			return fmt.Errorf("port %d is already used by another service", port)
		}
	}

	// Check if port is already bound by checking both TCP and UDP
	if isPortInUse(port) {
		return fmt.Errorf("port %d is already in use by another process", port)
	}

	return nil
}

// isPortInUse checks if a port is currently bound on TCP or UDP
func isPortInUse(port int) bool {
	// Check TCP
	tcpAddr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", tcpAddr)
	if err != nil {
		return true // Port is in use
	}
	listener.Close()

	// Check UDP
	udpAddr := fmt.Sprintf(":%d", port)
	conn, err := net.ListenPacket("udp", udpAddr)
	if err != nil {
		return true // Port is in use
	}
	conn.Close()

	return false
}

// ParsePort parses a string as a port number
func ParsePort(s string) (int, error) {
	port, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0, fmt.Errorf("invalid port number: %s", s)
	}
	return port, nil
}
