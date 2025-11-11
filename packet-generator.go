//go:build ignore
// +build ignore

// Packet Generator - Creates binary packets for udp-sender
// Run with: go run packet-generator.go | sudo ./udp-sender -dest-host HOST -dest-port PORT
package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
)

// Protocol magic number for packet synchronization and stream alignment detection
const (
	MagicByte1 = 0xC1
	MagicByte2 = 0x21
	MagicByte3 = 0xB1
)

// Packet represents a UDP packet to be sent
type Packet struct {
	SrcIP    net.IP
	SrcPort  uint16
	DestIP   net.IP
	DestPort uint16
	Payload  []byte
}

// WriteTo writes the packet to the output in binary format
// Format: [Magic(3)][Flags(1)][SrcIP(4/16)][DestIP(4/16)][SrcPort(2)][DestPort(2)][PayloadLen(2)][Payload(N)]
func (p *Packet) WriteTo(w *os.File) error {
	// Write magic number for synchronization
	if _, err := w.Write([]byte{MagicByte1, MagicByte2, MagicByte3}); err != nil {
		return fmt.Errorf("writing magic bytes: %w", err)
	}

	// Determine IP version
	isIPv6 := p.SrcIP.To4() == nil

	// Write flags byte (bitfield)
	var flags byte
	if isIPv6 {
		flags |= FlagIPv6
	}
	if _, err := w.Write([]byte{flags}); err != nil {
		return fmt.Errorf("writing flags byte: %w", err)
	}

	// Write source IP
	if isIPv6 {
		// IPv6: 16 bytes
		srcIP16 := p.SrcIP.To16()
		if srcIP16 == nil {
			return fmt.Errorf("source IP must be valid IPv6")
		}
		if _, err := w.Write(srcIP16); err != nil {
			return fmt.Errorf("writing source IPv6: %w", err)
		}
	} else {
		// IPv4: 4 bytes
		srcIP4 := p.SrcIP.To4()
		if srcIP4 == nil {
			return fmt.Errorf("source IP must be valid IPv4")
		}
		if _, err := w.Write(srcIP4); err != nil {
			return fmt.Errorf("writing source IPv4: %w", err)
		}
	}

	// Write destination IP (same version as source)
	if isIPv6 {
		// IPv6: 16 bytes
		destIP16 := p.DestIP.To16()
		if destIP16 == nil {
			return fmt.Errorf("destination IP must be valid IPv6")
		}
		if _, err := w.Write(destIP16); err != nil {
			return fmt.Errorf("writing destination IPv6: %w", err)
		}
	} else {
		// IPv4: 4 bytes
		destIP4 := p.DestIP.To4()
		if destIP4 == nil {
			return fmt.Errorf("destination IP must be valid IPv4")
		}
		if _, err := w.Write(destIP4); err != nil {
			return fmt.Errorf("writing destination IPv4: %w", err)
		}
	}

	// Write source port (2 bytes, big endian)
	var srcPortBytes [2]byte
	binary.BigEndian.PutUint16(srcPortBytes[:], p.SrcPort)
	if _, err := w.Write(srcPortBytes[:]); err != nil {
		return fmt.Errorf("writing source port: %w", err)
	}

	// Write destination port (2 bytes, big endian)
	var destPortBytes [2]byte
	binary.BigEndian.PutUint16(destPortBytes[:], p.DestPort)
	if _, err := w.Write(destPortBytes[:]); err != nil {
		return fmt.Errorf("writing destination port: %w", err)
	}

	// Write payload length (2 bytes, big endian)
	var lenBytes [2]byte
	binary.BigEndian.PutUint16(lenBytes[:], uint16(len(p.Payload)))
	if _, err := w.Write(lenBytes[:]); err != nil {
		return fmt.Errorf("writing payload length: %w", err)
	}

	// Write payload
	if len(p.Payload) > 0 {
		if _, err := w.Write(p.Payload); err != nil {
			return fmt.Errorf("writing payload: %w", err)
		}
	}

	return nil
}

func main() {
	numPackets := flag.Int("count", 10, "Number of packets to generate")
	baseIP := flag.String("base-ip", "10.0.0.1", "Base source IP address (will increment)")
	basePort := flag.Int("base-port", 5000, "Base source port number (will increment)")
	destIP := flag.String("dest-ip", "192.168.1.100", "Destination IP address")
	destPort := flag.Int("dest-port", 514, "Destination port number")
	message := flag.String("message", "Test packet", "Message template (will append packet number)")
	ipv6 := flag.Bool("ipv6", false, "Generate IPv6 packets instead of IPv4")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Packet Generator - Creates binary packet stream for udp-sender\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # Generate 100 IPv4 packets\n")
		fmt.Fprintf(os.Stderr, "  go run packet-generator.go -count 100 -dest-ip 192.168.1.100 -dest-port 514 | sudo ./udp-sender\n\n")
		fmt.Fprintf(os.Stderr, "  # Generate 50 IPv6 packets\n")
		fmt.Fprintf(os.Stderr, "  go run packet-generator.go -ipv6 -base-ip 2001:db8::1 -dest-ip 2001:db8::100 -count 50 | sudo ./udp-sender\n\n")
		fmt.Fprintf(os.Stderr, "  # Save to file\n")
		fmt.Fprintf(os.Stderr, "  go run packet-generator.go -count 1000 > packets.bin\n")
		fmt.Fprintf(os.Stderr, "  cat packets.bin | sudo ./udp-sender\n")
	}

	flag.Parse()

	// Parse base IP
	baseIPAddr := net.ParseIP(*baseIP)
	if baseIPAddr == nil {
		log.Fatalf("Invalid base IP address: %s", *baseIP)
	}

	// Parse destination IP
	destIPAddr := net.ParseIP(*destIP)
	if destIPAddr == nil {
		log.Fatalf("Invalid destination IP address: %s", *destIP)
	}

	// Validate IP version matches flag
	isIPv6 := baseIPAddr.To4() == nil
	if *ipv6 && !isIPv6 {
		log.Fatalf("IPv6 flag set but base IP is IPv4: %s", *baseIP)
	}
	if !*ipv6 && isIPv6 {
		log.Fatalf("IPv4 mode but base IP is IPv6: %s (use -ipv6 flag)", *baseIP)
	}

	// Validate destination IP matches source IP version
	destIsIPv6 := destIPAddr.To4() == nil
	if isIPv6 != destIsIPv6 {
		log.Fatalf("Source and destination IP versions must match (source: %s, dest: %s)", *baseIP, *destIP)
	}

	fmt.Fprintf(os.Stderr, "Generating %d %s packets: %s:%d -> %s:%d...\n",
		*numPackets, map[bool]string{true: "IPv6", false: "IPv4"}[*ipv6],
		*baseIP, *basePort, *destIP, *destPort)

	for i := 0; i < *numPackets; i++ {
		var srcIP net.IP

		if *ipv6 {
			// IPv6: increment last byte
			srcIP = make(net.IP, net.IPv6len)
			copy(srcIP, baseIPAddr.To16())
			srcIP[15] = byte((int(baseIPAddr.To16()[15]) + i) % 256)
		} else {
			// IPv4: increment last octet
			srcIP = make(net.IP, net.IPv4len)
			copy(srcIP, baseIPAddr.To4())
			srcIP[3] = byte((int(baseIPAddr.To4()[3]) + i) % 256)
		}

		// Calculate source port
		srcPort := uint16(*basePort + i)

		// Create payload
		payload := fmt.Sprintf("%s #%d", *message, i+1)

		// Create and write packet
		packet := Packet{
			SrcIP:    srcIP,
			SrcPort:  srcPort,
			DestIP:   destIPAddr,
			DestPort: uint16(*destPort),
			Payload:  []byte(payload),
		}

		if err := packet.WriteTo(os.Stdout); err != nil {
			log.Fatalf("Error writing packet %d: %v", i+1, err)
		}

		// Progress to stderr
		if (i+1)%100 == 0 {
			fmt.Fprintf(os.Stderr, "Generated %d packets...\n", i+1)
		}
	}

	fmt.Fprintf(os.Stderr, "Complete: generated %d packets\n", *numPackets)
}
