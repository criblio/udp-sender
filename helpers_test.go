package main

import (
	"net"
	"os"
	"syscall"
	"testing"

	"github.com/criblio/udp-sender/constants"
)

// requireRoot skips the test if not running as root/admin or if running in short mode
func requireRoot(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test that requires root privileges in short mode")
	}
	if os.Geteuid() != 0 {
		t.Skip("This test requires root privileges (run with sudo)")
	}
}

// requireNonRoot skips the test if running as root
func requireNonRoot(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("This test must not be run as root")
	}
}

// hasIPv6 checks if IPv6 is available and routable on the system
func hasIPv6() bool {
	// Try to create a raw IPv6 socket
	fd, err := syscall.Socket(syscall.AF_INET6, syscall.SOCK_RAW, syscall.IPPROTO_RAW)
	if err != nil {
		return false
	}
	defer func() { _ = syscall.Close(fd) }()

	// Check if we can parse an IPv6 address
	ip := net.ParseIP("::1")
	if ip == nil || ip.To4() != nil {
		return false
	}

	// Try to actually send a packet to see if routing works
	sender, err := NewUDPSender(constants.MaxPayloadIPv4, constants.MaxPayloadIPv6)
	if err != nil {
		return false
	}
	defer func() { _ = sender.Close() }()

	// Try sending a small test packet to ::1 (localhost)
	srcIP := net.ParseIP("::1")
	destIP := net.ParseIP("::1")
	_, err = sender.Send("test", srcIP, 12345, destIP, 54321)

	// If we can send to localhost, IPv6 is functional
	return err == nil
}

// requireIPv6 skips the test if IPv6 is not available
func requireIPv6(t *testing.T) {
	t.Helper()
	requireRoot(t) // IPv6 raw sockets also need root

	if !hasIPv6() {
		t.Skip("IPv6 is not available on this system")
	}
}
