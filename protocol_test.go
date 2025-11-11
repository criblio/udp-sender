package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"net"
	"strings"
	"testing"
)

// mockSender is a mock implementation of PacketSender for testing
type mockSender struct {
	packets []mockPacket
	sendErr error
}

type mockPacket struct {
	message  string
	srcIP    net.IP
	srcPort  uint16
	destIP   net.IP
	destPort uint16
}

func (m *mockSender) Send(message string, srcIP net.IP, srcPort uint16, destIP net.IP, destPort uint16) (int, error) {
	if m.sendErr != nil {
		return 0, m.sendErr
	}

	// Validate MTU limits like the real sender
	payload := []byte(message)
	srcIPv4 := srcIP.To4()
	isIPv6 := srcIPv4 == nil
	maxPayload := MaxPayloadIPv4
	ipVersion := "IPv4"
	if isIPv6 {
		maxPayload = MaxPayloadIPv6
		ipVersion = "IPv6"
	}

	if len(payload) > maxPayload {
		return 0, errors.New("payload size exceeds MTU limit for " + ipVersion)
	}

	m.packets = append(m.packets, mockPacket{
		message:  message,
		srcIP:    srcIP,
		srcPort:  srcPort,
		destIP:   destIP,
		destPort: destPort,
	})
	return len(message), nil
}

func (m *mockSender) Close() error {
	return nil
}

// buildPacketBytes constructs a binary packet according to the protocol
func buildPacketBytes(version byte, srcIP net.IP, destIP net.IP, srcPort, destPort uint16, payload []byte) []byte {
	buf := &bytes.Buffer{}

	// Magic bytes
	buf.WriteByte(MagicByte1)
	buf.WriteByte(MagicByte2)
	buf.WriteByte(MagicByte3)

	// Flags
	buf.WriteByte(version)

	// Source IP
	isIPv6 := (version & FlagIPv6) != 0
	if !isIPv6 {
		buf.Write(srcIP.To4())
	} else {
		buf.Write(srcIP.To16())
	}

	// Destination IP
	if !isIPv6 {
		buf.Write(destIP.To4())
	} else {
		buf.Write(destIP.To16())
	}

	// Ports
	_ = binary.Write(buf, binary.BigEndian, srcPort)
	_ = binary.Write(buf, binary.BigEndian, destPort)

	// Payload length
	_ = binary.Write(buf, binary.BigEndian, uint16(len(payload)))

	// Payload
	buf.Write(payload)

	return buf.Bytes()
}

func TestProcessInputStream_IPv4_SinglePacket(t *testing.T) {
	srcIP := net.ParseIP("192.168.1.100").To4()
	destIP := net.ParseIP("8.8.8.8").To4()
	payload := []byte("test data")

	packetData := buildPacketBytes(0, srcIP, destIP, 12345, 53, payload)
	reader := bytes.NewReader(packetData)

	var logBuf bytes.Buffer
	logger := &Logger{output: &logBuf, minLevel: LogLevelDebug}
	sender := &mockSender{}

	err := processInputStream(logger, sender, reader)
	if err != nil {
		t.Fatalf("processInputStream failed: %v", err)
	}

	// Verify packet was sent
	if len(sender.packets) != 1 {
		t.Fatalf("Expected 1 packet, got %d", len(sender.packets))
	}

	pkt := sender.packets[0]
	if pkt.message != string(payload) {
		t.Errorf("Payload = %q, want %q", pkt.message, string(payload))
	}
	if !pkt.srcIP.Equal(srcIP) {
		t.Errorf("Source IP = %s, want %s", pkt.srcIP, srcIP)
	}
	if !pkt.destIP.Equal(destIP) {
		t.Errorf("Dest IP = %s, want %s", pkt.destIP, destIP)
	}
	if pkt.srcPort != 12345 {
		t.Errorf("Source port = %d, want 12345", pkt.srcPort)
	}
	if pkt.destPort != 53 {
		t.Errorf("Dest port = %d, want 53", pkt.destPort)
	}

	// Check log output
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "Stream complete") {
		t.Error("Expected 'Stream complete' in log output")
	}
}

func TestProcessInputStream_IPv6_SinglePacket(t *testing.T) {
	srcIP := net.ParseIP("2001:db8::1")
	destIP := net.ParseIP("2001:db8::2")
	payload := []byte("ipv6 test")

	packetData := buildPacketBytes(FlagIPv6, srcIP, destIP, 54321, 80, payload)
	reader := bytes.NewReader(packetData)

	var logBuf bytes.Buffer
	logger := &Logger{output: &logBuf, minLevel: LogLevelDebug}
	sender := &mockSender{}

	err := processInputStream(logger, sender, reader)
	if err != nil {
		t.Fatalf("processInputStream failed: %v", err)
	}

	// Verify packet was sent
	if len(sender.packets) != 1 {
		t.Fatalf("Expected 1 packet, got %d", len(sender.packets))
	}

	pkt := sender.packets[0]
	if pkt.message != string(payload) {
		t.Errorf("Payload = %q, want %q", pkt.message, string(payload))
	}
	if !pkt.srcIP.Equal(srcIP.To16()) {
		t.Errorf("Source IP = %s, want %s", pkt.srcIP, srcIP)
	}
	if !pkt.destIP.Equal(destIP.To16()) {
		t.Errorf("Dest IP = %s, want %s", pkt.destIP, destIP)
	}
}

func TestProcessInputStream_MultiplePackets(t *testing.T) {
	buf := &bytes.Buffer{}

	// Write 3 packets
	packets := []struct {
		srcIP    net.IP
		destIP   net.IP
		srcPort  uint16
		destPort uint16
		payload  string
	}{
		{net.ParseIP("10.0.0.1").To4(), net.ParseIP("10.0.0.2").To4(), 1000, 2000, "packet1"},
		{net.ParseIP("10.0.0.3").To4(), net.ParseIP("10.0.0.4").To4(), 3000, 4000, "packet2"},
		{net.ParseIP("10.0.0.5").To4(), net.ParseIP("10.0.0.6").To4(), 5000, 6000, "packet3"},
	}

	for _, pkt := range packets {
		buf.Write(buildPacketBytes(0, pkt.srcIP, pkt.destIP, pkt.srcPort, pkt.destPort, []byte(pkt.payload)))
	}

	var logBuf bytes.Buffer
	logger := &Logger{output: &logBuf, minLevel: LogLevelDebug}
	sender := &mockSender{}

	err := processInputStream(logger, sender, buf)
	if err != nil {
		t.Fatalf("processInputStream failed: %v", err)
	}

	// Verify all packets were sent
	if len(sender.packets) != 3 {
		t.Fatalf("Expected 3 packets, got %d", len(sender.packets))
	}

	for i, expected := range packets {
		pkt := sender.packets[i]
		if pkt.message != expected.payload {
			t.Errorf("Packet %d: payload = %q, want %q", i, pkt.message, expected.payload)
		}
	}
}

func TestProcessInputStream_EmptyPayload(t *testing.T) {
	srcIP := net.ParseIP("192.168.1.1").To4()
	destIP := net.ParseIP("192.168.1.2").To4()
	payload := []byte{}

	packetData := buildPacketBytes(0, srcIP, destIP, 1234, 5678, payload)
	reader := bytes.NewReader(packetData)

	var logBuf bytes.Buffer
	logger := &Logger{output: &logBuf, minLevel: LogLevelDebug}
	sender := &mockSender{}

	err := processInputStream(logger, sender, reader)
	if err != nil {
		t.Fatalf("processInputStream failed: %v", err)
	}

	// Verify packet was sent with empty payload
	if len(sender.packets) != 1 {
		t.Fatalf("Expected 1 packet, got %d", len(sender.packets))
	}

	if sender.packets[0].message != "" {
		t.Errorf("Expected empty payload, got %q", sender.packets[0].message)
	}
}

func TestProcessInputStream_EmptyStream(t *testing.T) {
	reader := bytes.NewReader([]byte{})

	var logBuf bytes.Buffer
	logger := &Logger{output: &logBuf, minLevel: LogLevelDebug}
	sender := &mockSender{}

	err := processInputStream(logger, sender, reader)
	if err != nil {
		t.Fatalf("processInputStream should handle empty stream, got error: %v", err)
	}

	// Verify no packets were sent
	if len(sender.packets) != 0 {
		t.Errorf("Expected 0 packets, got %d", len(sender.packets))
	}
}

func TestProcessInputStream_InvalidMagicNumber(t *testing.T) {
	buf := &bytes.Buffer{}
	buf.WriteByte(0xFF) // Wrong magic byte
	buf.WriteByte(0xFF)
	buf.WriteByte(0xFF)

	var logBuf bytes.Buffer
	logger := &Logger{output: &logBuf, minLevel: LogLevelDebug}
	sender := &mockSender{}

	err := processInputStream(logger, sender, buf)
	if err == nil {
		t.Fatal("Expected error for invalid magic number")
	}

	if !strings.Contains(err.Error(), "invalid magic number") {
		t.Errorf("Expected 'invalid magic number' error, got: %v", err)
	}
}

func TestProcessInputStream_UnknownFlagsAccepted(t *testing.T) {
	// Flags byte with IPv6 bit set plus an unknown high bit
	flags := byte(FlagIPv6 | 0x80)
	srcIP := net.ParseIP("2001:db8::1")
	destIP := net.ParseIP("2001:db8::2")
	payload := []byte("ipv6 test")

	packetData := buildPacketBytes(flags, srcIP, destIP, 54321, 80, payload)
	reader := bytes.NewReader(packetData)

	var logBuf bytes.Buffer
	logger := &Logger{output: &logBuf, minLevel: LogLevelDebug}
	sender := &mockSender{}

	err := processInputStream(logger, sender, reader)
	if err != nil {
		t.Fatalf("processInputStream failed with unknown flags set: %v", err)
	}
	if len(sender.packets) != 1 {
		t.Fatalf("Expected 1 packet, got %d", len(sender.packets))
	}
}

func TestProcessInputStream_IncompleteStream_MagicBytes(t *testing.T) {
	// Only 2 magic bytes instead of 3
	buf := bytes.NewReader([]byte{MagicByte1, MagicByte2})

	var logBuf bytes.Buffer
	logger := &Logger{output: &logBuf, minLevel: LogLevelDebug}
	sender := &mockSender{}

	err := processInputStream(logger, sender, buf)
	if err == nil {
		t.Fatal("Expected error for incomplete magic bytes")
	}

	if !strings.Contains(err.Error(), "reading magic bytes") {
		t.Errorf("Expected 'reading magic bytes' error, got: %v", err)
	}
}

func TestProcessInputStream_IncompleteStream_Version(t *testing.T) {
	buf := &bytes.Buffer{}
	buf.WriteByte(MagicByte1)
	buf.WriteByte(MagicByte2)
	buf.WriteByte(MagicByte3)
	// Missing flags byte

	var logBuf bytes.Buffer
	logger := &Logger{output: &logBuf, minLevel: LogLevelDebug}
	sender := &mockSender{}

	err := processInputStream(logger, sender, buf)
	if err == nil {
		t.Fatal("Expected error for incomplete stream")
	}

	if !strings.Contains(err.Error(), "reading flags byte") {
		t.Errorf("Expected 'reading flags byte' error, got: %v", err)
	}
}

func TestProcessInputStream_IncompleteStream_IPv4SourceIP(t *testing.T) {
	buf := &bytes.Buffer{}
	buf.WriteByte(MagicByte1)
	buf.WriteByte(MagicByte2)
	buf.WriteByte(MagicByte3)
	buf.WriteByte(0)            // IPv4
	buf.Write([]byte{192, 168}) // Only 2 bytes of IP instead of 4

	var logBuf bytes.Buffer
	logger := &Logger{output: &logBuf, minLevel: LogLevelDebug}
	sender := &mockSender{}

	err := processInputStream(logger, sender, buf)
	if err == nil {
		t.Fatal("Expected error for incomplete IPv4 source address")
	}

	if !strings.Contains(err.Error(), "reading IPv4 source address") {
		t.Errorf("Expected 'reading IPv4 source address' error, got: %v", err)
	}
}

func TestProcessInputStream_IncompleteStream_IPv6SourceIP(t *testing.T) {
	buf := &bytes.Buffer{}
	buf.WriteByte(MagicByte1)
	buf.WriteByte(MagicByte2)
	buf.WriteByte(MagicByte3)
	buf.WriteByte(FlagIPv6)    // IPv6
	buf.Write(make([]byte, 8)) // Only 8 bytes instead of 16

	var logBuf bytes.Buffer
	logger := &Logger{output: &logBuf, minLevel: LogLevelDebug}
	sender := &mockSender{}

	err := processInputStream(logger, sender, buf)
	if err == nil {
		t.Fatal("Expected error for incomplete IPv6 source address")
	}

	if !strings.Contains(err.Error(), "reading IPv6 source address") {
		t.Errorf("Expected 'reading IPv6 source address' error, got: %v", err)
	}
}

func TestProcessInputStream_IncompleteStream_DestIP(t *testing.T) {
	buf := &bytes.Buffer{}
	buf.WriteByte(MagicByte1)
	buf.WriteByte(MagicByte2)
	buf.WriteByte(MagicByte3)
	buf.WriteByte(0)                            // IPv4
	buf.Write(net.ParseIP("192.168.1.1").To4()) // Full source IP
	buf.Write([]byte{10, 0})                    // Incomplete dest IP

	var logBuf bytes.Buffer
	logger := &Logger{output: &logBuf, minLevel: LogLevelDebug}
	sender := &mockSender{}

	err := processInputStream(logger, sender, buf)
	if err == nil {
		t.Fatal("Expected error for incomplete destination address")
	}

	if !strings.Contains(err.Error(), "reading IPv4 destination address") {
		t.Errorf("Expected 'reading IPv4 destination address' error, got: %v", err)
	}
}

func TestProcessInputStream_IncompleteStream_SourcePort(t *testing.T) {
	srcIP := net.ParseIP("192.168.1.1").To4()
	destIP := net.ParseIP("192.168.1.2").To4()

	buf := &bytes.Buffer{}
	buf.WriteByte(MagicByte1)
	buf.WriteByte(MagicByte2)
	buf.WriteByte(MagicByte3)
	buf.WriteByte(0)
	buf.Write(srcIP)
	buf.Write(destIP)
	buf.WriteByte(0x12) // Only 1 byte of port

	var logBuf bytes.Buffer
	logger := &Logger{output: &logBuf, minLevel: LogLevelDebug}
	sender := &mockSender{}

	err := processInputStream(logger, sender, buf)
	if err == nil {
		t.Fatal("Expected error for incomplete source port")
	}

	if !strings.Contains(err.Error(), "reading source port") {
		t.Errorf("Expected 'reading source port' error, got: %v", err)
	}
}

func TestProcessInputStream_IncompleteStream_DestPort(t *testing.T) {
	srcIP := net.ParseIP("192.168.1.1").To4()
	destIP := net.ParseIP("192.168.1.2").To4()

	buf := &bytes.Buffer{}
	buf.WriteByte(MagicByte1)
	buf.WriteByte(MagicByte2)
	buf.WriteByte(MagicByte3)
	buf.WriteByte(0)
	buf.Write(srcIP)
	buf.Write(destIP)
	_ = binary.Write(buf, binary.BigEndian, uint16(12345)) // Source port
	buf.WriteByte(0x12)                                    // Only 1 byte of dest port

	var logBuf bytes.Buffer
	logger := &Logger{output: &logBuf, minLevel: LogLevelDebug}
	sender := &mockSender{}

	err := processInputStream(logger, sender, buf)
	if err == nil {
		t.Fatal("Expected error for incomplete dest port")
	}

	if !strings.Contains(err.Error(), "reading destination port") {
		t.Errorf("Expected 'reading destination port' error, got: %v", err)
	}
}

func TestProcessInputStream_IncompleteStream_PayloadLength(t *testing.T) {
	srcIP := net.ParseIP("192.168.1.1").To4()
	destIP := net.ParseIP("192.168.1.2").To4()

	buf := &bytes.Buffer{}
	buf.WriteByte(MagicByte1)
	buf.WriteByte(MagicByte2)
	buf.WriteByte(MagicByte3)
	buf.WriteByte(0)
	buf.Write(srcIP)
	buf.Write(destIP)
	_ = binary.Write(buf, binary.BigEndian, uint16(12345))
	_ = binary.Write(buf, binary.BigEndian, uint16(8080))
	buf.WriteByte(0x00) // Only 1 byte of payload length

	var logBuf bytes.Buffer
	logger := &Logger{output: &logBuf, minLevel: LogLevelDebug}
	sender := &mockSender{}

	err := processInputStream(logger, sender, buf)
	if err == nil {
		t.Fatal("Expected error for incomplete payload length")
	}

	if !strings.Contains(err.Error(), "reading payload length") {
		t.Errorf("Expected 'reading payload length' error, got: %v", err)
	}
}

func TestProcessInputStream_IncompleteStream_Payload(t *testing.T) {
	srcIP := net.ParseIP("192.168.1.1").To4()
	destIP := net.ParseIP("192.168.1.2").To4()

	buf := &bytes.Buffer{}
	buf.WriteByte(MagicByte1)
	buf.WriteByte(MagicByte2)
	buf.WriteByte(MagicByte3)
	buf.WriteByte(4)
	buf.Write(srcIP)
	buf.Write(destIP)
	_ = binary.Write(buf, binary.BigEndian, uint16(12345))
	_ = binary.Write(buf, binary.BigEndian, uint16(8080))
	_ = binary.Write(buf, binary.BigEndian, uint16(100)) // Payload length = 100
	buf.Write([]byte("only 10 bytes"))                   // Less than declared

	var logBuf bytes.Buffer
	logger := &Logger{output: &logBuf, minLevel: LogLevelDebug}
	sender := &mockSender{}

	err := processInputStream(logger, sender, buf)
	if err == nil {
		t.Fatal("Expected error for incomplete payload")
	}

	if !strings.Contains(err.Error(), "reading payload") {
		t.Errorf("Expected 'reading payload' error, got: %v", err)
	}
}

func TestProcessInputStream_SendError(t *testing.T) {
	srcIP := net.ParseIP("192.168.1.1").To4()
	destIP := net.ParseIP("192.168.1.2").To4()
	payload := []byte("test")

	packetData := buildPacketBytes(4, srcIP, destIP, 12345, 8080, payload)
	reader := bytes.NewReader(packetData)

	var logBuf bytes.Buffer
	logger := &Logger{output: &logBuf, minLevel: LogLevelDebug}
	sender := &mockSender{sendErr: errors.New("send failed")}

	err := processInputStream(logger, sender, reader)
	if err == nil {
		t.Fatal("Expected error when sender.Send fails")
	}

	if !strings.Contains(err.Error(), "sending packet") {
		t.Errorf("Expected 'sending packet' error, got: %v", err)
	}
}

func TestProcessInputStream_ProgressUpdates(t *testing.T) {
	buf := &bytes.Buffer{}

	// Write 150 packets to trigger progress updates
	srcIP := net.ParseIP("10.0.0.1").To4()
	destIP := net.ParseIP("10.0.0.2").To4()
	for i := 0; i < 150; i++ {
		buf.Write(buildPacketBytes(4, srcIP, destIP, 1000, 2000, []byte("data")))
	}

	var logBuf bytes.Buffer
	logger := &Logger{output: &logBuf, minLevel: LogLevelDebug}
	sender := &mockSender{}

	err := processInputStream(logger, sender, buf)
	if err != nil {
		t.Fatalf("processInputStream failed: %v", err)
	}

	// Verify all packets were sent
	if len(sender.packets) != 150 {
		t.Fatalf("Expected 150 packets, got %d", len(sender.packets))
	}

	// Check for progress updates in log (should occur at packet 100)
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "Progress update") {
		t.Error("Expected 'Progress update' in log output")
	}
}

func TestProcessInputStream_LargePayload(t *testing.T) {
	srcIP := net.ParseIP("192.168.1.1").To4()
	destIP := net.ParseIP("192.168.1.2").To4()
	// Create a large payload at the MTU limit (1472 bytes for IPv4)
	payload := make([]byte, MaxPayloadIPv4)
	for i := range payload {
		payload[i] = byte(i % 256)
	}

	packetData := buildPacketBytes(4, srcIP, destIP, 12345, 8080, payload)
	reader := bytes.NewReader(packetData)

	var logBuf bytes.Buffer
	logger := &Logger{output: &logBuf, minLevel: LogLevelDebug}
	sender := &mockSender{}

	err := processInputStream(logger, sender, reader)
	if err != nil {
		t.Fatalf("processInputStream failed: %v", err)
	}

	// Verify packet was sent
	if len(sender.packets) != 1 {
		t.Fatalf("Expected 1 packet, got %d", len(sender.packets))
	}

	if len(sender.packets[0].message) != len(payload) {
		t.Errorf("Payload length = %d, want %d", len(sender.packets[0].message), len(payload))
	}
}

// TestProcessInputStream_MTUExceeded tests that packets exceeding MTU are logged and skipped
func TestProcessInputStream_MTUExceeded(t *testing.T) {
	tests := []struct {
		name            string
		version         byte
		srcIP           net.IP
		destIP          net.IP
		payloadSize     int
		expectedDropped int
		expectedSent    int
	}{
		{
			name:            "IPv4 packet exceeds MTU",
			version:         0,
			srcIP:           net.ParseIP("192.168.1.1"),
			destIP:          net.ParseIP("192.168.1.2"),
			payloadSize:     2000, // Exceeds 1472 byte limit
			expectedDropped: 1,
			expectedSent:    0,
		},
		{
			name:            "IPv6 packet exceeds MTU",
			version:         FlagIPv6,
			srcIP:           net.ParseIP("2001:db8::1"),
			destIP:          net.ParseIP("2001:db8::2"),
			payloadSize:     2000, // Exceeds 1452 byte limit
			expectedDropped: 1,
			expectedSent:    0,
		},
		{
			name:            "IPv4 packet within MTU",
			version:         0,
			srcIP:           net.ParseIP("192.168.1.1"),
			destIP:          net.ParseIP("192.168.1.2"),
			payloadSize:     1000, // Within 1472 byte limit
			expectedDropped: 0,
			expectedSent:    1,
		},
		{
			name:            "IPv6 packet within MTU",
			version:         FlagIPv6,
			srcIP:           net.ParseIP("2001:db8::1"),
			destIP:          net.ParseIP("2001:db8::2"),
			payloadSize:     1000, // Within 1452 byte limit
			expectedDropped: 0,
			expectedSent:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logBuf bytes.Buffer
			logger := &Logger{output: &logBuf, minLevel: LogLevelDebug}

			sender := &mockSender{}

			// Build packet with specified payload size
			payload := bytes.Repeat([]byte("X"), tt.payloadSize)
			packet := buildPacketBytes(tt.version, tt.srcIP, tt.destIP, 12345, 54321, payload)

			reader := bytes.NewReader(packet)

			err := processInputStream(logger, sender, reader)
			if err != nil {
				t.Fatalf("processInputStream failed: %v", err)
			}

			// Check number of packets sent
			if len(sender.packets) != tt.expectedSent {
				t.Errorf("Expected %d packets sent, got %d", tt.expectedSent, len(sender.packets))
			}

			// Check log output for dropped packets
			logOutput := logBuf.String()
			if tt.expectedDropped > 0 {
				if !strings.Contains(logOutput, "Packet dropped due to MTU limit") {
					t.Errorf("Expected MTU drop log message, but didn't find it in: %s", logOutput)
				}
				if !strings.Contains(logOutput, "exceeds MTU limit") {
					t.Errorf("Expected MTU limit error in log, but didn't find it in: %s", logOutput)
				}
				// Verify packets_dropped field is in the completion message
				if !strings.Contains(logOutput, "packets_dropped") {
					t.Errorf("Expected packets_dropped field in log output, but didn't find it in: %s", logOutput)
				}
			} else {
				if strings.Contains(logOutput, "Packet dropped") {
					t.Errorf("Unexpected packet drop in log: %s", logOutput)
				}
			}
		})
	}
}

// TestProcessInputStream_MixedMTU tests processing a stream with both valid and oversized packets
func TestProcessInputStream_MixedMTU(t *testing.T) {
	var logBuf bytes.Buffer
	logger := &Logger{output: &logBuf, minLevel: LogLevelDebug}

	sender := &mockSender{}

	// Build a stream with 3 packets: valid, oversized, valid
	buf := &bytes.Buffer{}

	// Packet 1: Valid IPv4 packet (small payload)
	payload1 := []byte("Small packet")
	packet1 := buildPacketBytes(4, net.ParseIP("192.168.1.1"), net.ParseIP("192.168.1.2"), 12345, 54321, payload1)
	buf.Write(packet1)

	// Packet 2: Oversized IPv4 packet
	payload2 := bytes.Repeat([]byte("X"), 2000)
	packet2 := buildPacketBytes(4, net.ParseIP("192.168.1.3"), net.ParseIP("192.168.1.4"), 12346, 54322, payload2)
	buf.Write(packet2)

	// Packet 3: Valid IPv4 packet (small payload)
	payload3 := []byte("Another small packet")
	packet3 := buildPacketBytes(4, net.ParseIP("192.168.1.5"), net.ParseIP("192.168.1.6"), 12347, 54323, payload3)
	buf.Write(packet3)

	reader := bytes.NewReader(buf.Bytes())

	err := processInputStream(logger, sender, reader)
	if err != nil {
		t.Fatalf("processInputStream failed: %v", err)
	}

	// Should have sent 2 packets (packet 1 and packet 3)
	if len(sender.packets) != 2 {
		t.Errorf("Expected 2 packets sent, got %d", len(sender.packets))
	}

	// Verify the correct packets were sent
	if len(sender.packets) >= 1 && sender.packets[0].message != string(payload1) {
		t.Errorf("First packet payload mismatch")
	}
	if len(sender.packets) >= 2 && sender.packets[1].message != string(payload3) {
		t.Errorf("Second packet payload mismatch")
	}

	// Check log output
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "Packet dropped due to MTU limit") {
		t.Errorf("Expected MTU drop log message, but didn't find it in: %s", logOutput)
	}

	// Verify final stats show 2 sent, 1 dropped
	if !strings.Contains(logOutput, "packets_sent") {
		t.Errorf("Expected packets_sent in log output")
	}
	if !strings.Contains(logOutput, "packets_dropped") {
		t.Errorf("Expected packets_dropped in log output")
	}
}
