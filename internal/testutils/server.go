package testutils

import (
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

// Protocol represents the network protocol type.
type Protocol uint

const (
	// TCP protocol.
	TCP Protocol = iota
	// UDP protocol.
	UDP
)

// GetFreePort returns a free port on the specified host for the given protocol (TCP or UDP).
func GetFreePort(t *testing.T, host string, protocol Protocol) int {
	t.Helper()

	switch protocol {
	case TCP:
		ln, err := net.Listen("tcp", net.JoinHostPort(host, "0"))
		require.NoError(t, err, "Setup: failed to listen on tcp")
		defer ln.Close()
		addr, ok := ln.Addr().(*net.TCPAddr)
		require.True(t, ok, "Setup: expected TCPAddr")
		return addr.Port

	case UDP:
		pc, err := net.ListenPacket("udp", net.JoinHostPort(host, "0"))
		require.NoError(t, err, "Setup: failed to listen on udp")
		defer pc.Close()
		addr, ok := pc.LocalAddr().(*net.UDPAddr)
		require.True(t, ok, "Setup: expected UDPAddr")
		return addr.Port

	default:
		t.Fatalf("unsupported protocol: %v", protocol)
		return 0
	}
}

// PortOpen checks if a port is open on the specified TCP host.
func PortOpen(t *testing.T, host string, port int) bool {
	t.Helper()

	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, fmt.Sprint(port)), 0)
	if err != nil {
		return false
	}
	defer conn.Close()
	return true
}
