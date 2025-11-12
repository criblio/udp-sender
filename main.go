package main

import (
	"flag"
	"fmt"
	"io"
	"os"
)

// Version is the current version of udp-sender
// This can be overridden at build time using: go build -ldflags "-X main.Version=v1.0.0"
var Version = "dev"

// run contains the core application logic and is testable
// It returns an error instead of calling os.Exit, making it easier to test
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	// Create a new flag set for testing purposes
	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	fs.SetOutput(stderr)

	// Define flags
	var showVersion bool
	var verbose bool
	var mtu int

	fs.BoolVar(&showVersion, "version", false, "Print version and exit")
	fs.BoolVar(&showVersion, "V", false, "Print version and exit (short)")
	fs.BoolVar(&verbose, "verbose", false, "Enable verbose logging (debug level)")
	fs.BoolVar(&verbose, "v", false, "Enable verbose logging (debug level, short)")
	fs.IntVar(&mtu, "mtu", DefaultMTU, "Maximum Transmission Unit in bytes (default: 1500)")
	fs.IntVar(&mtu, "m", DefaultMTU, "Maximum Transmission Unit in bytes (default: 1500, short)")

	fs.Usage = func() {
		_, _ = fmt.Fprintf(stderr, "Usage: %s [OPTIONS]\n\n", args[0])
		_, _ = fmt.Fprintf(stderr, "Send UDP packets with IP/port spoofing using raw sockets.\n")
		_, _ = fmt.Fprintf(stderr, "Requires root/administrator privileges (or the CAP_NET_RAW capability).\n\n")
		_, _ = fmt.Fprintf(stderr, "Options:\n")
		_, _ = fmt.Fprintf(stderr, "  -h, --help       Show this help message\n")
		_, _ = fmt.Fprintf(stderr, "  -V, --version    Print version and exit\n")
		_, _ = fmt.Fprintf(stderr, "  -v, --verbose    Enable verbose logging (debug level)\n")
		_, _ = fmt.Fprintf(stderr, "  -m, --mtu <bytes> Maximum Transmission Unit (default: 1500)\n")
		_, _ = fmt.Fprintf(stderr, "\n")
		_, _ = fmt.Fprintf(stderr, "Reads packets from stdin using a binary protocol.\n")
		_, _ = fmt.Fprintf(stderr, "See PROTOCOL.md for complete protocol specification.\n\n")
		_, _ = fmt.Fprintf(stderr, "Examples:\n")
		_, _ = fmt.Fprintf(stderr, "  cat packets.bin | sudo %s\n", args[0])
		_, _ = fmt.Fprintf(stderr, "  ./packet-generator | sudo %s\n", args[0])
		_, _ = fmt.Fprintf(stderr, "  ./packet-generator | sudo %s -m 9000\n", args[0])
	}

	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	// Handle version flag
	if showVersion {
		_, _ = fmt.Fprintf(stdout, "udp-sender version %s\n", Version)
		return nil
	}

	// Validate MTU
	if mtu < MinMTU || mtu > MaxMTU {
		return fmt.Errorf("MTU must be between %d and %d bytes (got %d)", MinMTU, MaxMTU, mtu)
	}

	// Create logger with appropriate level
	var logger *Logger
	if verbose {
		logger = NewLoggerWithLevel(LogLevelDebug)
	} else {
		logger = NewLogger()
	}

	// Report version on startup
	logger.Info("udp-sender starting", map[string]any{
		"version": Version,
	})

	// Calculate max payload sizes based on MTU
	// IPv4: MTU - 20 (IP header) - 8 (UDP header)
	// IPv6: MTU - 40 (IPv6 header) - 8 (UDP header)
	maxPayloadIPv4 := mtu - IPv4HeaderSize - UDPHeaderSize
	maxPayloadIPv6 := mtu - IPv6HeaderSize - UDPHeaderSize

	logger.Debug("MTU configuration", map[string]any{
		"mtu":              mtu,
		"max_payload_ipv4": maxPayloadIPv4,
		"max_payload_ipv6": maxPayloadIPv6,
	})

	// Create sender (no destination needed - comes from packets)
	sender, err := NewUDPSender(maxPayloadIPv4, maxPayloadIPv6)
	if err != nil {
		return fmt.Errorf("error creating UDP sender: %w", err)
	}
	defer func() {
		if err := sender.Close(); err != nil {
			logger.Error("Error closing sender", map[string]any{
				"error": err.Error(),
			})
		}
	}()

	// Process stream from stdin
	err = processInputStream(logger, sender, stdin)
	if err != nil {
		return fmt.Errorf("error processing stream: %w", err)
	}

	return nil
}

func main() {
	if err := run(os.Args, os.Stdin, os.Stdout, os.Stderr); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
