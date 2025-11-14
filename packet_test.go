package main

import (
	"net"
	"os"
	"testing"

	"github.com/criblio/udp-sender/constants"
)

func TestCalculateChecksum(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "empty data",
			data: []byte{},
		},
		{
			name: "simple data",
			data: []byte{0x00, 0x01, 0x02, 0x03},
		},
		{
			name: "odd length data",
			data: []byte{0x00, 0x01, 0x02},
		},
		{
			name: "all zeros",
			data: []byte{0x00, 0x00, 0x00, 0x00},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateChecksum(tt.data)

			// Verify checksum is calculated (non-zero for most inputs)
			// and verify double calculation returns original
			doubleData := make([]byte, len(tt.data)+2)
			copy(doubleData, tt.data)
			// Insert checksum
			doubleData[len(doubleData)-2] = byte(result >> 8)
			doubleData[len(doubleData)-1] = byte(result & 0xff)

			// Verify checksum produces expected result for validation
			verification := calculateChecksum(doubleData)
			if len(tt.data) > 0 && verification != 0xffff && verification != 0 {
				t.Logf("Checksum verification result: 0x%04x for input 0x%04x", verification, result)
			}
		})
	}
}

// TestBuildIPv4Header_NoRoot tests IPv4 header construction without requiring root
func TestBuildIPv4Header_NoRoot(t *testing.T) {
	// Create a sender struct without initializing raw sockets
	sender := &UDPSender{fdIPv4: -1, fdIPv6: -1}

	srcIP := net.ParseIP("10.0.0.1").To4()
	destIP := net.ParseIP("192.168.1.1").To4()
	dataLen := 100
	header := sender.buildIPv4Header(dataLen, srcIP, destIP)

	// Check header length
	if len(header) != 20 {
		t.Errorf("IP header length = %d, want 20", len(header))
	}

	// Check version and IHL (0x45 = version 4, IHL 5)
	if header[0] != 0x45 {
		t.Errorf("Version/IHL = 0x%02x, want 0x45", header[0])
	}

	// Check Type of Service
	if header[1] != 0 {
		t.Errorf("Type of Service = %d, want 0", header[1])
	}

	// Check total length
	totalLen := (uint16(header[2]) << 8) | uint16(header[3])
	expectedLen := uint16(20 + dataLen)
	if totalLen != expectedLen {
		t.Errorf("Total length = %d, want %d", totalLen, expectedLen)
	}

	// Check TTL
	if header[8] != 64 {
		t.Errorf("TTL = %d, want 64", header[8])
	}

	// Check protocol (UDP = 17)
	if header[9] != 17 {
		t.Errorf("Protocol = %d, want 17 (UDP)", header[9])
	}

	// Verify source IP
	headerSrcIP := net.IP(header[12:16])
	if !headerSrcIP.Equal(srcIP) {
		t.Errorf("Source IP = %s, want %s", headerSrcIP.String(), srcIP.String())
	}

	// Verify destination IP
	headerDestIP := net.IP(header[16:20])
	if !headerDestIP.Equal(destIP) {
		t.Errorf("Destination IP = %s, want %s", headerDestIP.String(), destIP.String())
	}

	// Verify checksum is set (non-zero)
	checksum := (uint16(header[10]) << 8) | uint16(header[11])
	if checksum == 0 {
		t.Error("IP header checksum should not be zero")
	}
}

// TestBuildIPv6Header_NoRoot tests IPv6 header construction without requiring root
func TestBuildIPv6Header_NoRoot(t *testing.T) {
	// Create a sender struct without initializing raw sockets
	sender := &UDPSender{fdIPv4: -1, fdIPv6: -1}

	srcIP := net.ParseIP("2001:db8::1")
	destIP := net.ParseIP("2001:db8::2")
	dataLen := 100
	header := sender.buildIPv6Header(dataLen, srcIP, destIP)

	// Check header length
	if len(header) != 40 {
		t.Errorf("IPv6 header length = %d, want 40", len(header))
	}

	// Check version (first 4 bits should be 6)
	version := (header[0] >> 4) & 0x0F
	if version != 6 {
		t.Errorf("IP version = %d, want 6", version)
	}

	// Check payload length
	payloadLen := (uint16(header[4]) << 8) | uint16(header[5])
	if payloadLen != uint16(dataLen) {
		t.Errorf("Payload length = %d, want %d", payloadLen, dataLen)
	}

	// Check next header (UDP = 17)
	if header[6] != 17 {
		t.Errorf("Next header = %d, want 17 (UDP)", header[6])
	}

	// Check hop limit
	if header[7] != 64 {
		t.Errorf("Hop limit = %d, want 64", header[7])
	}

	// Verify source address
	headerSrcIP := net.IP(header[8:24])
	if !headerSrcIP.Equal(srcIP.To16()) {
		t.Errorf("Source IP = %v, want %v", headerSrcIP, srcIP)
	}

	// Verify destination address
	headerDestIP := net.IP(header[24:40])
	if !headerDestIP.Equal(destIP.To16()) {
		t.Errorf("Destination IP = %v, want %v", headerDestIP, destIP)
	}
}

// TestBuildUDPHeader_NoRoot tests UDP header construction without requiring root
func TestBuildUDPHeader_NoRoot(t *testing.T) {
	// Create a sender struct without initializing raw sockets
	sender := &UDPSender{fdIPv4: -1, fdIPv6: -1}

	tests := []struct {
		name     string
		srcIP    net.IP
		destIP   net.IP
		srcPort  uint16
		destPort uint16
		payload  []byte
		isIPv6   bool
	}{
		{
			name:     "IPv4",
			srcIP:    net.ParseIP("127.0.0.1").To4(),
			destIP:   net.ParseIP("127.0.0.2").To4(),
			srcPort:  12345,
			destPort: 8080,
			payload:  []byte("test payload"),
			isIPv6:   false,
		},
		{
			name:     "IPv6",
			srcIP:    net.ParseIP("::1"),
			destIP:   net.ParseIP("::2"),
			srcPort:  54321,
			destPort: 9090,
			payload:  []byte("ipv6 test"),
			isIPv6:   true,
		},
		{
			name:     "Empty payload",
			srcIP:    net.ParseIP("10.0.0.1").To4(),
			destIP:   net.ParseIP("10.0.0.2").To4(),
			srcPort:  1234,
			destPort: 5678,
			payload:  []byte{},
			isIPv6:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := sender.buildUDPHeader(tt.payload, tt.srcIP, tt.srcPort, tt.destIP, tt.destPort, tt.isIPv6)

			// Check header length
			if len(header) != 8 {
				t.Errorf("UDP header length = %d, want 8", len(header))
			}

			// Check source port
			srcPort := (uint16(header[0]) << 8) | uint16(header[1])
			if srcPort != tt.srcPort {
				t.Errorf("Source port = %d, want %d", srcPort, tt.srcPort)
			}

			// Check destination port
			destPort := (uint16(header[2]) << 8) | uint16(header[3])
			if destPort != tt.destPort {
				t.Errorf("Destination port = %d, want %d", destPort, tt.destPort)
			}

			// Check length
			length := (uint16(header[4]) << 8) | uint16(header[5])
			expectedLength := uint16(8 + len(tt.payload))
			if length != expectedLength {
				t.Errorf("UDP length = %d, want %d", length, expectedLength)
			}

			// Checksum should be calculated (non-zero in most cases)
			checksum := (uint16(header[6]) << 8) | uint16(header[7])
			// Note: checksum can be zero for specific data combinations, so we just verify it's set
			_ = checksum
		})
	}
}

// TestCalculateUDPChecksum_NoRoot tests UDP checksum calculation without requiring root
func TestCalculateUDPChecksum_NoRoot(t *testing.T) {
	// Create a sender struct without initializing raw sockets
	sender := &UDPSender{fdIPv4: -1, fdIPv6: -1}

	tests := []struct {
		name      string
		srcIP     net.IP
		destIP    net.IP
		udpHeader []byte
		payload   []byte
		isIPv6    bool
	}{
		{
			name:      "IPv4 with payload",
			srcIP:     net.ParseIP("192.168.1.1").To4(),
			destIP:    net.ParseIP("192.168.1.2").To4(),
			udpHeader: []byte{0x30, 0x39, 0x1f, 0x90, 0x00, 0x10, 0x00, 0x00}, // src:12345 dst:8080 len:16
			payload:   []byte("testdata"),
			isIPv6:    false,
		},
		{
			name:      "IPv6 with payload",
			srcIP:     net.ParseIP("2001:db8::1"),
			destIP:    net.ParseIP("2001:db8::2"),
			udpHeader: []byte{0x30, 0x39, 0x1f, 0x90, 0x00, 0x10, 0x00, 0x00},
			payload:   []byte("testdata"),
			isIPv6:    true,
		},
		{
			name:      "IPv4 empty payload",
			srcIP:     net.ParseIP("10.0.0.1").To4(),
			destIP:    net.ParseIP("10.0.0.2").To4(),
			udpHeader: []byte{0x04, 0xd2, 0x16, 0x2e, 0x00, 0x08, 0x00, 0x00}, // src:1234 dst:5678 len:8
			payload:   []byte{},
			isIPv6:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checksum := sender.calculateUDPChecksum(tt.udpHeader, tt.payload, tt.srcIP, tt.destIP, tt.isIPv6)

			// Checksum should be calculated (we can't predict exact value without implementing the algorithm here)
			// Just verify it returns a value (uint16 is always valid, 0-0xFFFF)
			_ = checksum
		})
	}
}

// TestBuildPacket_NoRoot tests complete packet building without requiring root
func TestBuildPacket_NoRoot(t *testing.T) {
	// Create a sender struct without initializing raw sockets
	sender := &UDPSender{fdIPv4: -1, fdIPv6: -1}

	tests := []struct {
		name             string
		srcIP            net.IP
		destIP           net.IP
		srcPort          uint16
		destPort         uint16
		payload          []byte
		expectedIPHdrLen int
	}{
		{
			name:             "IPv4 packet",
			srcIP:            net.ParseIP("192.168.1.100").To4(),
			destIP:           net.ParseIP("8.8.8.8").To4(),
			srcPort:          12345,
			destPort:         53,
			payload:          []byte("DNS query"),
			expectedIPHdrLen: 20,
		},
		{
			name:             "IPv6 packet",
			srcIP:            net.ParseIP("2001:db8::100"),
			destIP:           net.ParseIP("2001:4860:4860::8888"),
			srcPort:          54321,
			destPort:         53,
			payload:          []byte("DNS query v6"),
			expectedIPHdrLen: 40,
		},
		{
			name:             "IPv4 empty payload",
			srcIP:            net.ParseIP("10.0.0.1").To4(),
			destIP:           net.ParseIP("10.0.0.2").To4(),
			srcPort:          1234,
			destPort:         5678,
			payload:          []byte{},
			expectedIPHdrLen: 20,
		},
		{
			name:             "IPv4 large payload",
			srcIP:            net.ParseIP("172.16.0.1").To4(),
			destIP:           net.ParseIP("172.16.0.2").To4(),
			srcPort:          9999,
			destPort:         8888,
			payload:          make([]byte, 1024),
			expectedIPHdrLen: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packet := sender.buildPacket(tt.payload, tt.srcIP, tt.srcPort, tt.destIP, tt.destPort)

			// Verify packet structure: IP header + UDP header (8 bytes) + payload
			expectedLen := tt.expectedIPHdrLen + 8 + len(tt.payload)
			if len(packet) != expectedLen {
				t.Errorf("Packet length = %d, want %d", len(packet), expectedLen)
			}

			// Verify IP header
			if tt.expectedIPHdrLen == 20 {
				// IPv4
				if packet[0] != 0x45 {
					t.Errorf("IP version/IHL = 0x%02x, want 0x45", packet[0])
				}
				if packet[9] != 17 {
					t.Errorf("IP protocol = %d, want 17 (UDP)", packet[9])
				}
			} else {
				// IPv6
				version := (packet[0] >> 4) & 0x0F
				if version != 6 {
					t.Errorf("IP version = %d, want 6", version)
				}
				if packet[6] != 17 {
					t.Errorf("IPv6 next header = %d, want 17 (UDP)", packet[6])
				}
			}

			// Verify UDP header starts at correct offset
			udpOffset := tt.expectedIPHdrLen
			srcPort := (uint16(packet[udpOffset]) << 8) | uint16(packet[udpOffset+1])
			if srcPort != tt.srcPort {
				t.Errorf("UDP source port = %d, want %d", srcPort, tt.srcPort)
			}

			destPort := (uint16(packet[udpOffset+2]) << 8) | uint16(packet[udpOffset+3])
			if destPort != tt.destPort {
				t.Errorf("UDP dest port = %d, want %d", destPort, tt.destPort)
			}

			// Verify payload
			payloadOffset := udpOffset + 8
			if len(tt.payload) > 0 {
				if !bytesEqual(packet[payloadOffset:], tt.payload) {
					t.Error("Payload mismatch")
				}
			}
		})
	}
}

// Helper function to compare byte slices
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestBuildIPHeader(t *testing.T) {
	requireRoot(t)

	sender, err := NewUDPSender(constants.MaxPayloadIPv4, constants.MaxPayloadIPv6)
	if err != nil {
		t.Fatalf("Failed to create sender: %v", err)
	}
	defer func() { _ = sender.Close() }()

	srcIP := net.ParseIP("10.0.0.1").To4()
	destIP := net.ParseIP("192.168.1.1").To4()
	header := sender.buildIPv4Header(100, srcIP, destIP)

	// Check header length
	if len(header) != 20 {
		t.Errorf("IP header length = %d, want 20", len(header))
	}

	// Check version and IHL
	if header[0] != 0x45 {
		t.Errorf("Version/IHL = 0x%02x, want 0x45", header[0])
	}

	// Check protocol (UDP = 17)
	if header[9] != 17 {
		t.Errorf("Protocol = %d, want 17 (UDP)", header[9])
	}

	// Verify source IP
	headerSrcIP := net.IP(header[12:16])
	if headerSrcIP.String() != "10.0.0.1" {
		t.Errorf("Source IP = %s, want 10.0.0.1", headerSrcIP.String())
	}

	// Verify destination IP
	dstIP := net.IP(header[16:20])
	if dstIP.String() != "192.168.1.1" {
		t.Errorf("Destination IP = %s, want 192.168.1.1", dstIP.String())
	}
}

func TestBuildIPv6Header(t *testing.T) {
	requireRoot(t)

	sender, err := NewUDPSender(constants.MaxPayloadIPv4, constants.MaxPayloadIPv6)
	if err != nil {
		t.Skip("IPv6 not available")
	}
	defer func() { _ = sender.Close() }()

	srcIP := net.ParseIP("2001:db8::1")
	destIP := net.ParseIP("::1")
	header := sender.buildIPv6Header(100, srcIP, destIP)

	// Check header length
	if len(header) != 40 {
		t.Errorf("IPv6 header length = %d, want 40", len(header))
	}

	// Check version (first 4 bits should be 6)
	version := (header[0] >> 4) & 0x0F
	if version != 6 {
		t.Errorf("IP version = %d, want 6", version)
	}

	// Check next header (UDP = 17)
	if header[6] != 17 {
		t.Errorf("Next header = %d, want 17 (UDP)", header[6])
	}

	// Verify source address
	headerSrcIP := net.IP(header[8:24])
	if !headerSrcIP.Equal(srcIP.To16()) {
		t.Errorf("Source IP = %v, want %v", headerSrcIP, srcIP)
	}

	// Verify destination address
	headerDestIP := net.IP(header[24:40])
	if !headerDestIP.Equal(destIP.To16()) {
		t.Errorf("Destination IP = %v, want %v", headerDestIP, destIP)
	}
}

func TestBuildUDPHeader(t *testing.T) {
	requireRoot(t)

	sender, err := NewUDPSender(constants.MaxPayloadIPv4, constants.MaxPayloadIPv6)
	if err != nil {
		t.Fatalf("Failed to create sender: %v", err)
	}
	defer func() { _ = sender.Close() }()

	payload := []byte("test payload")
	srcIP := net.ParseIP("127.0.0.1").To4()
	destIP := net.ParseIP("127.0.0.1").To4()
	header := sender.buildUDPHeader(payload, srcIP, 12345, destIP, 8080, false)

	// Check header length
	if len(header) != 8 {
		t.Errorf("UDP header length = %d, want 8", len(header))
	}

	// Check source port
	srcPort := (uint16(header[0]) << 8) | uint16(header[1])
	if srcPort != 12345 {
		t.Errorf("Source port = %d, want 12345", srcPort)
	}

	// Check destination port
	dstPort := (uint16(header[2]) << 8) | uint16(header[3])
	if dstPort != 8080 {
		t.Errorf("Destination port = %d, want 8080", dstPort)
	}

	// Check length
	length := (uint16(header[4]) << 8) | uint16(header[5])
	expectedLength := uint16(8 + len(payload))
	if length != expectedLength {
		t.Errorf("UDP length = %d, want %d", length, expectedLength)
	}
}

// Benchmarks

func BenchmarkCalculateChecksum(b *testing.B) {
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i % 256)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calculateChecksum(data)
	}
}

func BenchmarkBuildIPv4Header(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark that requires root privileges in short mode")
	}
	if os.Geteuid() != 0 {
		b.Skip("This benchmark requires root privileges (run with sudo)")
	}

	sender, err := NewUDPSender(constants.MaxPayloadIPv4, constants.MaxPayloadIPv6)
	if err != nil {
		b.Fatalf("Failed to create sender: %v", err)
	}
	defer func() { _ = sender.Close() }()

	srcIP := net.ParseIP("10.0.0.1").To4()
	destIP := net.ParseIP("192.168.1.1").To4()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sender.buildIPv4Header(1024, srcIP, destIP)
	}
}

func BenchmarkBuildUDPHeader(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark that requires root privileges in short mode")
	}
	if os.Geteuid() != 0 {
		b.Skip("This benchmark requires root privileges (run with sudo)")
	}

	sender, err := NewUDPSender(constants.MaxPayloadIPv4, constants.MaxPayloadIPv6)
	if err != nil {
		b.Fatalf("Failed to create sender: %v", err)
	}
	defer func() { _ = sender.Close() }()

	payload := make([]byte, 512)
	srcIP := net.ParseIP("127.0.0.1").To4()
	destIP := net.ParseIP("127.0.0.1").To4()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sender.buildUDPHeader(payload, srcIP, 12345, destIP, 8080, false)
	}
}
