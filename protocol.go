package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strings"
)

// processInputStream reads packets from stdin using the binary protocol.
// See PROTOCOL.md for complete protocol specification.
func processInputStream(logger *Logger, sender PacketSender, input io.Reader) error {
	reader := bufio.NewReader(input)
	packetCount := 0
	packetsDropped := 0
	bytesSent := uint64(0)

	for {
		// Read and validate magic number
		var magicBytes [3]byte
		n, err := io.ReadFull(reader, magicBytes[:])
		if err == io.EOF {
			// End of stream
			logger.Info("Stream complete", map[string]any{
				"packets_sent":    packetCount,
				"packets_dropped": packetsDropped,
				"bytes_sent":      bytesSent,
			})
			return nil
		}
		if err != nil {
			return fmt.Errorf("reading magic bytes: %w (read %d bytes)", err, n)
		}

		// Validate magic number
		if magicBytes[0] != MagicByte1 || magicBytes[1] != MagicByte2 || magicBytes[2] != MagicByte3 {
			return fmt.Errorf("invalid magic number: got [0x%02X 0x%02X 0x%02X], expected [0x%02X 0x%02X 0x%02X] - stream may be misaligned",
				magicBytes[0], magicBytes[1], magicBytes[2], MagicByte1, MagicByte2, MagicByte3)
		}

		// Read flags byte (bitfield)
		var flagsByte [1]byte
		if _, err := io.ReadFull(reader, flagsByte[:]); err != nil {
			return fmt.Errorf("reading flags byte: %w", err)
		}
		flags := flagsByte[0]
		isIPv6 := (flags & FlagIPv6) != 0

		// Read source IP based on version
		var srcIP net.IP
		if !isIPv6 {
			// IPv4: 4 bytes
			var srcIPBytes [4]byte
			if _, err := io.ReadFull(reader, srcIPBytes[:]); err != nil {
				return fmt.Errorf("reading IPv4 source address: %w", err)
			}
			srcIP = net.IP(srcIPBytes[:])
		} else {
			// IPv6: 16 bytes
			var srcIPBytes [16]byte
			if _, err := io.ReadFull(reader, srcIPBytes[:]); err != nil {
				return fmt.Errorf("reading IPv6 source address: %w", err)
			}
			srcIP = net.IP(srcIPBytes[:])
		}

		// Read destination IP based on version (same size as source)
		var destIP net.IP
		if !isIPv6 {
			// IPv4: 4 bytes
			var destIPBytes [4]byte
			if _, err := io.ReadFull(reader, destIPBytes[:]); err != nil {
				return fmt.Errorf("reading IPv4 destination address: %w", err)
			}
			destIP = net.IP(destIPBytes[:])
		} else {
			// IPv6: 16 bytes
			var destIPBytes [16]byte
			if _, err := io.ReadFull(reader, destIPBytes[:]); err != nil {
				return fmt.Errorf("reading IPv6 destination address: %w", err)
			}
			destIP = net.IP(destIPBytes[:])
		}

		// Read source port (2 bytes, big endian)
		var srcPortBytes [2]byte
		if _, err := io.ReadFull(reader, srcPortBytes[:]); err != nil {
			return fmt.Errorf("reading source port: %w", err)
		}
		srcPort := binary.BigEndian.Uint16(srcPortBytes[:])

		// Read destination port (2 bytes, big endian)
		var destPortBytes [2]byte
		if _, err := io.ReadFull(reader, destPortBytes[:]); err != nil {
			return fmt.Errorf("reading destination port: %w", err)
		}
		destPort := binary.BigEndian.Uint16(destPortBytes[:])

		// Read payload length (2 bytes, big endian)
		var lenBytes [2]byte
		if _, err := io.ReadFull(reader, lenBytes[:]); err != nil {
			return fmt.Errorf("reading payload length: %w", err)
		}
		payloadLen := binary.BigEndian.Uint16(lenBytes[:])

		// Read payload
		payload := make([]byte, payloadLen)
		if payloadLen > 0 {
			if _, err := io.ReadFull(reader, payload); err != nil {
				return fmt.Errorf("reading payload (%d bytes): %w", payloadLen, err)
			}
		}

		// Send the packet
		n, err = sender.Send(string(payload), srcIP, srcPort, destIP, destPort)
		if err != nil {
			// Check if this is an MTU error - if so, log and continue
			if strings.Contains(err.Error(), "exceeds MTU limit") {
				logger.Error("Packet dropped due to MTU limit", map[string]any{
					"packet_number": packetCount + packetsDropped + 1,
					"payload_size":  len(payload),
					"source_ip":     srcIP.String(),
					"source_port":   srcPort,
					"dest_ip":       destIP.String(),
					"dest_port":     destPort,
					"error":         err.Error(),
				})
				packetsDropped++
				continue
			}
			// For other errors, terminate processing
			return fmt.Errorf("sending packet %d: %w", packetCount+1, err)
		}

		packetCount++
		bytesSent += uint64(n)

		// Progress feedback every 100 packets
		if packetCount%100 == 0 {
			logger.Debug("Progress update", map[string]any{
				"packets_sent": packetCount,
				"bytes_sent":   bytesSent,
			})
		}
	}
}
