package main

import (
	"fmt"
	"net"
	"syscall"
)

// PacketSender is an interface for sending UDP packets
type PacketSender interface {
	Send(message string, srcIP net.IP, srcPort uint16, destIP net.IP, destPort uint16) (int, error)
	Close() error
}

// UDPSender handles UDP packet sending with raw sockets for IP spoofing.
// This class implements the PacketSender interface. Both source and destination
// IP and port are specified per-packet in the Send() method.
// Supports both IPv4 and IPv6.
type UDPSender struct {
	fdIPv4         int
	fdIPv6         int
	maxPayloadIPv4 int
	maxPayloadIPv6 int
}

// Ensure UDPSender implements PacketSender interface at compile time
var _ PacketSender = (*UDPSender)(nil)

// NewUDPSender creates a new UDP sender with raw socket support
// Both source and destination IP and port are specified per-packet in the Send() method
// Requires root/admin privileges to create raw sockets
// Supports both IPv4 and IPv6
// maxPayloadIPv4 and maxPayloadIPv6 specify the maximum payload sizes based on MTU
func NewUDPSender(maxPayloadIPv4, maxPayloadIPv6 int) (*UDPSender, error) {
	sender := &UDPSender{
		fdIPv4:         -1,
		fdIPv6:         -1,
		maxPayloadIPv4: maxPayloadIPv4,
		maxPayloadIPv6: maxPayloadIPv6,
	}

	// Create IPv4 socket
	fd4, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_RAW)
	if err != nil {
		return nil, fmt.Errorf("failed to create raw IPv4 socket (requires root): %w", err)
	}

	// Set IP_HDRINCL option to tell the kernel we're providing the IP header
	err = syscall.SetsockoptInt(fd4, syscall.IPPROTO_IP, syscall.IP_HDRINCL, 1)
	if err != nil {
		_ = syscall.Close(fd4)
		return nil, fmt.Errorf("failed to set IP_HDRINCL: %w", err)
	}

	sender.fdIPv4 = fd4

	// Create IPv6 socket
	fd6, err := syscall.Socket(syscall.AF_INET6, syscall.SOCK_RAW, syscall.IPPROTO_RAW)
	if err != nil {
		// Clean up IPv4 socket
		_ = syscall.Close(sender.fdIPv4)
		return nil, fmt.Errorf("failed to create raw IPv6 socket (requires root): %w", err)
	}
	// Note: IPv6 raw sockets don't require IPV6_HDRINCL on most systems
	// The kernel automatically sets the next header field

	sender.fdIPv6 = fd6

	return sender, nil
}

// Send sends a message via raw UDP socket with spoofed source and destination
// Implements PacketSender.Send
// srcIP and srcPort specify the spoofed source address for this packet
// destIP and destPort specify the destination address for this packet
func (s *UDPSender) Send(message string, srcIP net.IP, srcPort uint16, destIP net.IP, destPort uint16) (int, error) {
	payload := []byte(message)

	// Validate IPs
	if srcIP == nil {
		return 0, fmt.Errorf("source IP is nil")
	}
	if destIP == nil {
		return 0, fmt.Errorf("destination IP is nil")
	}

	// Determine IP versions
	srcIPv4 := srcIP.To4()
	destIPv4 := destIP.To4()
	isSrcIPv4 := srcIPv4 != nil
	isDestIPv4 := destIPv4 != nil

	// Validate payload size against MTU limits
	maxPayload := s.maxPayloadIPv4
	ipVersion := "IPv4"
	if !isSrcIPv4 {
		maxPayload = s.maxPayloadIPv6
		ipVersion = "IPv6"
	}

	if len(payload) > maxPayload {
		return 0, fmt.Errorf("payload size %d exceeds MTU limit for %s (%d bytes)", len(payload), ipVersion, maxPayload)
	}

	// Check if both are IPv4
	if isSrcIPv4 && isDestIPv4 {
		// Use IPv4
		packet := s.buildPacket(payload, srcIPv4, srcPort, destIPv4, destPort)

		addr4 := &syscall.SockaddrInet4{
			Port: int(destPort),
		}
		copy(addr4.Addr[:], destIPv4)

		err := syscall.Sendto(s.fdIPv4, packet, 0, addr4)
		if err != nil {
			return 0, fmt.Errorf("failed to send packet to %s: %w", destIPv4, err)
		}

		return len(payload), nil
	}

	// Check if both are IPv6
	if !isSrcIPv4 && !isDestIPv4 {
		// Use IPv6
		packet := s.buildPacket(payload, srcIP, srcPort, destIP, destPort)

		addr6 := &syscall.SockaddrInet6{
			Port: int(0), // Destination port is not used for IPv6
		}
		copy(addr6.Addr[:], destIP.To16())

		err := syscall.Sendto(s.fdIPv6, packet, 0, addr6)
		if err != nil {
			return 0, fmt.Errorf("failed to send packet to %s: %w", destIP, err)
		}

		return len(payload), nil
	}

	// Mismatched IP versions
	return 0, fmt.Errorf("source and destination IP versions must match (src: %s, dest: %s)", srcIP, destIP)
}

// Close closes the raw socket(s)
// Implements PacketSender.Close
func (s *UDPSender) Close() error {
	var err4, err6 error

	if s.fdIPv4 >= 0 {
		err4 = syscall.Close(s.fdIPv4)
	}
	if s.fdIPv6 >= 0 {
		err6 = syscall.Close(s.fdIPv6)
	}

	// Return first error encountered
	if err4 != nil {
		return err4
	}
	return err6
}
