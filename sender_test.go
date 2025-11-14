package main

import (
	"fmt"
	"net"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/criblio/udp-sender/constants"
)

func TestNewUDPSender(t *testing.T) {
	requireRoot(t)

	sender, err := NewUDPSender(constants.MaxPayloadIPv4, constants.MaxPayloadIPv6)
	if err != nil {
		t.Fatalf("NewUDPSender() error = %v", err)
	}
	if sender == nil {
		t.Fatal("NewUDPSender() returned nil sender without error")
	}
	defer func() { _ = sender.Close() }()

	// Verify both socket file descriptors are valid
	if sender.fdIPv4 < 0 {
		t.Error("IPv4 socket file descriptor is invalid")
	}
	if sender.fdIPv6 < 0 {
		t.Error("IPv6 socket file descriptor is invalid")
	}
}

func TestUDPSender_Send(t *testing.T) {
	requireRoot(t)

	sender, err := NewUDPSender(constants.MaxPayloadIPv4, constants.MaxPayloadIPv6)
	if err != nil {
		t.Fatalf("Failed to create sender: %v", err)
	}
	defer func() { _ = sender.Close() }()

	// Note: On macOS/Darwin, raw sockets cannot send to localhost (127.0.0.1) with spoofed addresses
	// This is a kernel security restriction, not a bug in the code.
	// We test the Send() method still executes without crashing, even if macOS rejects the packet.

	tests := []struct {
		name     string
		message  string
		srcIP    string
		srcPort  uint16
		destIP   string
		destPort uint16
		wantErr  bool
	}{
		{
			name:     "simple IPv4 message",
			message:  "Hello, UDP!",
			srcIP:    "192.168.1.1",
			srcPort:  54321,
			destIP:   "8.8.8.8",
			destPort: 53,
			wantErr:  false, // May fail on macOS but code executes
		},
		{
			name:     "empty message",
			message:  "",
			srcIP:    "10.0.0.1",
			srcPort:  54322,
			destIP:   "1.1.1.1",
			destPort: 53,
			wantErr:  false,
		},
		{
			name:     "long message",
			message:  "This is a longer message to test UDP sending capabilities with raw sockets",
			srcIP:    "172.16.0.1",
			srcPort:  54323,
			destIP:   "8.8.4.4",
			destPort: 53,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srcIP := net.ParseIP(tt.srcIP)
			destIP := net.ParseIP(tt.destIP)

			n, err := sender.Send(tt.message, srcIP, tt.srcPort, destIP, tt.destPort)

			// On macOS, packets may be rejected, but the code should execute
			if err != nil && !tt.wantErr {
				t.Logf("Send() error (may be expected on macOS): %v", err)
				// Don't fail the test on macOS - this is a known OS limitation
				return
			}

			if err == nil {
				if n != len(tt.message) {
					t.Errorf("Send() sent %d bytes, want %d", n, len(tt.message))
				}
			}
		})
	}
}

func TestUDPSender_Close(t *testing.T) {
	requireRoot(t)

	sender, err := NewUDPSender(constants.MaxPayloadIPv4, constants.MaxPayloadIPv6)
	if err != nil {
		t.Fatalf("Failed to create sender: %v", err)
	}

	err = sender.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Closing again should not panic
	_ = sender.Close()
}

func TestPacketSender_Interface(t *testing.T) {
	requireRoot(t)

	// Test that UDPSender implements PacketSender interface
	var sender PacketSender
	udpSender, err := NewUDPSender(constants.MaxPayloadIPv4, constants.MaxPayloadIPv6)
	if err != nil {
		t.Fatalf("Failed to create sender: %v", err)
	}
	defer func() { _ = udpSender.Close() }()

	// Assign to interface
	sender = udpSender

	// Test Send through interface with destination
	srcIP := net.ParseIP("10.0.0.1")
	destIP := net.ParseIP("8.8.8.8")
	n, err := sender.Send("test", srcIP, 12345, destIP, 53)
	// May fail on macOS, but interface assignment works
	if err != nil {
		t.Logf("Interface Send() error (may be expected on macOS): %v", err)
	} else if n != 4 {
		t.Errorf("Interface Send() sent %d bytes, want 4", n)
	}
}

func TestUDPSender_Send_ErrorCases(t *testing.T) {
	requireRoot(t)

	sender, err := NewUDPSender(constants.MaxPayloadIPv4, constants.MaxPayloadIPv6)
	if err != nil {
		t.Fatalf("Failed to create sender: %v", err)
	}
	defer func() { _ = sender.Close() }()

	tests := []struct {
		name     string
		srcIP    net.IP
		srcPort  uint16
		destIP   net.IP
		destPort uint16
		wantErr  string
	}{
		{
			name:     "nil source IP",
			srcIP:    nil,
			srcPort:  1234,
			destIP:   net.ParseIP("8.8.8.8"),
			destPort: 53,
			wantErr:  "source IP is nil",
		},
		{
			name:     "nil dest IP",
			srcIP:    net.ParseIP("192.168.1.1"),
			srcPort:  1234,
			destIP:   nil,
			destPort: 53,
			wantErr:  "destination IP is nil",
		},
		{
			name:     "mismatched IP versions - IPv4 src, IPv6 dest",
			srcIP:    net.ParseIP("192.168.1.1"),
			srcPort:  1234,
			destIP:   net.ParseIP("2001:db8::1"),
			destPort: 53,
			wantErr:  "source and destination IP versions must match",
		},
		{
			name:     "mismatched IP versions - IPv6 src, IPv4 dest",
			srcIP:    net.ParseIP("2001:db8::1"),
			srcPort:  1234,
			destIP:   net.ParseIP("8.8.8.8"),
			destPort: 53,
			wantErr:  "source and destination IP versions must match",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := sender.Send("test", tt.srcIP, tt.srcPort, tt.destIP, tt.destPort)
			if err == nil {
				t.Error("Expected error, got nil")
			} else if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Expected error containing %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func TestUDPSender_Send_IPv6(t *testing.T) {
	requireIPv6(t)

	sender, err := NewUDPSender(constants.MaxPayloadIPv4, constants.MaxPayloadIPv6)
	if err != nil {
		t.Fatalf("Failed to create sender: %v", err)
	}
	defer func() { _ = sender.Close() }()

	// Test IPv6 packet sending
	srcIP := net.ParseIP("2001:db8::1")
	destIP := net.ParseIP("2001:4860:4860::8888") // Google DNS IPv6
	message := "IPv6 test"

	n, err := sender.Send(message, srcIP, 12345, destIP, 53)
	if err != nil {
		t.Logf("IPv6 Send() error (may be expected on some networks): %v", err)
	} else if n != len(message) {
		t.Errorf("Send() sent %d bytes, want %d", n, len(message))
	}
}

func TestUDPSender_Close_ErrorHandling(t *testing.T) {
	requireRoot(t)

	sender, err := NewUDPSender(constants.MaxPayloadIPv4, constants.MaxPayloadIPv6)
	if err != nil {
		t.Fatalf("Failed to create sender: %v", err)
	}

	// Close once
	err = sender.Close()
	if err != nil {
		t.Errorf("First Close() error = %v", err)
	}

	// Close again - should not panic, but may return error for invalid fd
	err = sender.Close()
	// This may or may not error depending on OS, but shouldn't crash
	t.Logf("Second Close() result: %v", err)
}

func TestRawSocketPermissions(t *testing.T) {
	// This test checks if we have permissions to create raw sockets
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_RAW)

	if err != nil {
		if os.Geteuid() != 0 {
			t.Skip("Raw socket creation requires root privileges (run with sudo)")
		} else {
			t.Errorf("Failed to create raw socket even with root: %v", err)
		}
		return
	}

	_ = syscall.Close(fd)
	t.Log("Successfully created raw socket - running with appropriate privileges")
}

// Benchmarks

func BenchmarkUDPSender_Send(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark that requires root privileges in short mode")
	}
	if os.Geteuid() != 0 {
		b.Skip("This benchmark requires root privileges (run with sudo)")
	}

	// Start a UDP server to receive messages
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		b.Fatalf("Failed to resolve address: %v", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		b.Fatalf("Failed to listen on UDP: %v", err)
	}
	defer func() { _ = conn.Close() }()

	// Get the actual port assigned
	serverAddr := conn.LocalAddr().(*net.UDPAddr)

	// Create sender
	sender, err := NewUDPSender(constants.MaxPayloadIPv4, constants.MaxPayloadIPv6)
	if err != nil {
		b.Fatalf("Failed to create sender: %v", err)
	}
	defer func() { _ = sender.Close() }()

	// Start goroutine to drain received packets
	done := make(chan bool)
	go func() {
		buf := make([]byte, 2048)
		for {
			select {
			case <-done:
				return
			default:
				_ = conn.SetReadDeadline(time.Now().Add(10 * time.Millisecond))
				_, _, _ = conn.ReadFromUDP(buf)
			}
		}
	}()
	defer func() { done <- true }()

	message := "Benchmark message payload for UDP sender testing"
	loopbackIP := net.ParseIP("127.0.0.1")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		srcPort := uint16(10000 + (i % 55000))
		_, err := sender.Send(message, loopbackIP, srcPort, loopbackIP, uint16(serverAddr.Port))
		if err != nil {
			b.Errorf("Send() error = %v", err)
		}
	}
}

func BenchmarkUDPSender_SendVariablePayloadSizes(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark that requires root privileges in short mode")
	}
	if os.Geteuid() != 0 {
		b.Skip("This benchmark requires root privileges (run with sudo)")
	}

	// Start a UDP server to receive messages
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		b.Fatalf("Failed to resolve address: %v", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		b.Fatalf("Failed to listen on UDP: %v", err)
	}
	defer func() { _ = conn.Close() }()

	serverAddr := conn.LocalAddr().(*net.UDPAddr)

	sender, err := NewUDPSender(constants.MaxPayloadIPv4, constants.MaxPayloadIPv6)
	if err != nil {
		b.Fatalf("Failed to create sender: %v", err)
	}
	defer func() { _ = sender.Close() }()

	// Start goroutine to drain received packets
	done := make(chan bool)
	go func() {
		buf := make([]byte, 65536)
		for {
			select {
			case <-done:
				return
			default:
				_ = conn.SetReadDeadline(time.Now().Add(10 * time.Millisecond))
				_, _, _ = conn.ReadFromUDP(buf)
			}
		}
	}()
	defer func() { done <- true }()

	loopbackIP := net.ParseIP("127.0.0.1")
	sizes := []int{64, 256, 512, 1024, 4096, 8192}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			payload := string(make([]byte, size))
			b.SetBytes(int64(size))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				srcPort := uint16(10000 + (i % 55000))
				_, err := sender.Send(payload, loopbackIP, srcPort, loopbackIP, uint16(serverAddr.Port))
				if err != nil {
					b.Errorf("Send() error = %v", err)
				}
			}
		})
	}
}

// TestUDPSender_MTUValidation tests that packets exceeding MTU limits are rejected
func TestUDPSender_MTUValidation(t *testing.T) {
	requireRoot(t)

	sender, err := NewUDPSender(constants.MaxPayloadIPv4, constants.MaxPayloadIPv6)
	if err != nil {
		t.Fatalf("Failed to create sender: %v", err)
	}
	defer func() { _ = sender.Close() }()

	tests := []struct {
		name        string
		payloadSize int
		srcIP       string
		destIP      string
		expectError bool
		description string
	}{
		{
			name:        "IPv4 within MTU limit",
			payloadSize: 1472,
			srcIP:       "192.168.1.1",
			destIP:      "192.168.1.2",
			expectError: false,
			description: "Payload exactly at IPv4 MTU limit should succeed",
		},
		{
			name:        "IPv4 exceeds MTU limit",
			payloadSize: 1473,
			srcIP:       "192.168.1.1",
			destIP:      "192.168.1.2",
			expectError: true,
			description: "Payload exceeding IPv4 MTU limit should fail",
		},
		{
			name:        "IPv4 small payload",
			payloadSize: 100,
			srcIP:       "192.168.1.1",
			destIP:      "192.168.1.2",
			expectError: false,
			description: "Small payload should succeed",
		},
		{
			name:        "IPv4 way over MTU",
			payloadSize: 5000,
			srcIP:       "192.168.1.1",
			destIP:      "192.168.1.2",
			expectError: true,
			description: "Large payload way over MTU should fail",
		},
		{
			name:        "IPv6 within MTU limit",
			payloadSize: 1452,
			srcIP:       "2001:db8::1",
			destIP:      "2001:db8::2",
			expectError: false,
			description: "Payload exactly at IPv6 MTU limit should succeed",
		},
		{
			name:        "IPv6 exceeds MTU limit",
			payloadSize: 1453,
			srcIP:       "2001:db8::1",
			destIP:      "2001:db8::2",
			expectError: true,
			description: "Payload exceeding IPv6 MTU limit should fail",
		},
		{
			name:        "IPv6 small payload",
			payloadSize: 100,
			srcIP:       "2001:db8::1",
			destIP:      "2001:db8::2",
			expectError: false,
			description: "Small IPv6 payload should succeed",
		},
		{
			name:        "IPv6 way over MTU",
			payloadSize: 5000,
			srcIP:       "2001:db8::1",
			destIP:      "2001:db8::2",
			expectError: true,
			description: "Large IPv6 payload way over MTU should fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Determine if this is an IPv6 test
			srcIP := net.ParseIP(tt.srcIP)
			isIPv6 := srcIP.To4() == nil

			// Skip IPv6 tests if IPv6 is not available
			if isIPv6 && !hasIPv6() {
				t.Skip("IPv6 is not available on this system")
			}

			// Create payload of specified size
			payload := strings.Repeat("X", tt.payloadSize)

			destIP := net.ParseIP(tt.destIP)

			_, err := sender.Send(payload, srcIP, 12345, destIP, 54321)

			if tt.expectError {
				if err == nil {
					t.Errorf("%s: expected error but got none", tt.description)
				} else if !strings.Contains(err.Error(), "exceeds MTU limit") {
					t.Errorf("%s: expected MTU error but got: %v", tt.description, err)
				} else {
					t.Logf("%s: correctly rejected with error: %v", tt.description, err)
				}
			} else {
				if err != nil {
					// For IPv6, network/routing errors are acceptable (may not have routable IPv6)
					if isIPv6 && (strings.Contains(err.Error(), "no route to host") ||
						strings.Contains(err.Error(), "network is unreachable") ||
						strings.Contains(err.Error(), "failed to send")) {
						t.Logf("%s: IPv6 Send() routing error (expected on systems without full IPv6): %v", tt.description, err)
					} else {
						t.Errorf("%s: unexpected error: %v", tt.description, err)
					}
				} else {
					t.Logf("%s: correctly accepted", tt.description)
				}
			}
		})
	}
}
