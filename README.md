# UDP Sender with IP Spoofing

A Go application for sending UDP packets with raw socket support, allowing IP and port spoofing.

[![CI](https://github.com/criblio/udp-sender/actions/workflows/ci.yml/badge.svg)](https://github.com/criblio/udp-sender/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/criblio/udp-sender)](https://goreportcard.com/report/github.com/criblio/udp-sender)

## Features

- **Dynamic IP Spoofing**: Specify both source and destination IP address per packet (IPv4 and IPv6)
- **Dynamic Port Spoofing**: Specify both source and destination port per packet  
- **IPv6 Support**: Full support for both IPv4 and IPv6 addresses
- **Binary Protocol**: Efficient wire format for high-volume packet streams
- **Raw Socket Implementation**: Manual IP and UDP header construction
- **Configurable MTU**: Adjustable Maximum Transmission Unit via command line (default 1500, range 576-9000 bytes)
- **MTU Validation**: Automatic validation and rejection of packets exceeding MTU limits based on configured value
- **Flexible API**: Source and destination addresses can change with each packet
- **Command Line Interface**: Simple command-line interface with no required arguments
- **Structured Logging**: Newline-delimited JSON (ND-JSON) logs for easy parsing and analysis (see [LOGGING.md](LOGGING.md))
- **Comprehensive Tests**: Full test coverage with proper privilege handling
- **CI/CD**: Automated testing with GitHub Actions

## Requirements

### System Requirements

- Go 1.21 or later
- **Root/Administrator privileges** or **`CAP_NET_RAW`** (required for raw socket creation)
- IPv4/6 network support

### Platform Support

- **Linux**: Full support (with `sudo` or `CAP_NET_RAW`)
- **macOS**: Full support (requires `sudo`)
- **Windows**: Not supported (use containers instead)

## Installation

### Linux Package Installation (Recommended)

Install using DEB or RPM packages for easier system integration and management:

```bash
# Debian/Ubuntu
VERSION=v1.0.0  # Replace with latest version
wget https://github.com/criblio/udp-sender/releases/download/${VERSION}/udp-sender-${VERSION#v}-x64.deb
wget https://github.com/criblio/udp-sender/releases/download/${VERSION}/udp-sender-${VERSION#v}-x64.deb.sha256

# Verify checksum
sha256sum -c udp-sender-${VERSION#v}-x64.deb.sha256

# Install package
sudo dpkg -i udp-sender-${VERSION#v}-x64.deb
```

or

```bash
# RHEL/CentOS/Fedora
wget https://github.com/criblio/udp-sender/releases/download/${VERSION}/udp-sender-${VERSION#v}-x64.rpm
wget https://github.com/criblio/udp-sender/releases/download/${VERSION}/udp-sender-${VERSION#v}-x64.rpm.sha256

# Verify checksum
sha256sum -c udp-sender-${VERSION#v}-x64.rpm.sha256

# Install package
sudo rpm -i udp-sender-${VERSION#v}-x64.rpm
```

then

```bash
# Add yourself to the udp-senders group
sudo usermod -a -G udp-senders $USER
newgrp udp-senders  # Or log out and back in
```

**Benefits (over using pre-built binaries):**

- Automatic capability configuration (`CAP_NET_RAW`)
- Group-based access control via `udp-senders` group
- Easier updates and removal through package manager
- System integration (installs to `/usr/bin/udp-sender`)

**Available packages:**

- Debian/Ubuntu: `.deb` packages (x64, ARM64)
- RHEL/CentOS/Fedora: `.rpm` packages (x64, ARM64)

### Download Pre-built Binaries

Download standalone binaries for any supported platform from [GitHub Releases](https://github.com/criblio/udp-sender/releases):

```bash
# Example: Linux x64
VERSION=v1.0.0  # Replace with latest version
wget https://github.com/criblio/udp-sender/releases/download/${VERSION}/udp-sender-${VERSION}-linux-x64.tar.gz
wget https://github.com/criblio/udp-sender/releases/download/${VERSION}/udp-sender-${VERSION}-linux-x64.tar.gz.sha256

# Verify checksum before extracting
sha256sum -c udp-sender-${VERSION}-linux-x64.tar.gz.sha256

# Extract
tar -xzf udp-sender-${VERSION}-linux-x64.tar.gz

# Make executable and move to PATH
chmod +x udp-sender-linux-x64
sudo mv udp-sender-linux-x64 /usr/bin/udp-sender

# For Linux: Grant CAP_NET_RAW capability (more secure than sudo)
sudo setcap cap_net_raw+ep /usr/bin/udp-sender
```

**Available Platforms:**

- Linux: x64, ARM64
- macOS: x64 (Intel), ARM64 (Apple Silicon)

**Security Note:** Always verify checksums to ensure file integrity and authenticity.

### Docker Container

Run using the official container image from GitHub Container Registry:

```bash
# Pull the latest image
docker pull ghcr.io/criblio/udp-sender:latest

# Run with a packet generator
go run packet-generator.go -count 10 | \
  docker run --rm -i --cap-add=NET_RAW \
  ghcr.io/criblio/udp-sender:latest

# Using a specific version
cat packets.bin | docker run --rm -i --cap-add=NET_RAW \
  ghcr.io/criblio/udp-sender:1.0.0
```

**Important**: The container requires `--cap-add=NET_RAW` capability to create raw sockets.

**Available tags:**

- `latest` - Latest stable release
- `1.0.0`, `1.0`, `1` - Semantic version tags
- Multi-architecture support: `linux/amd64`, `linux/arm64`

### Build from Source

```bash
git clone https://github.com/criblio/udp-sender.git
cd udp-sender

# Using Make (recommended)
make build

# Or using Go directly
go build
```

## Usage

⚠️ **Important**: This application requires root privileges or the `CAP_NET_RAW` capability to create raw sockets.

### Running Without Root (Linux)

Instead of running as root, you can grant the `CAP_NET_RAW` capability to the binary:

```bash
# Build the application
make build
# Or: go build -o udp-sender .

# Grant CAP_NET_RAW capability
sudo setcap cap_net_raw+ep ./udp-sender

# Now you can run without sudo
cat packets.bin | ./udp-sender
```

**Benefits of using capabilities**:

- ✅ More secure than running as root
- ✅ Follows principle of least privilege
- ✅ Only grants raw socket access, not full system privileges

**Note**: This only works on Linux. macOS does not support Linux capabilities.

### Command Line Arguments

```bash
Usage: udp-sender [OPTIONS]

Options:
  -h, --help        Show this help message
  -V, --version     Print version and exit
  -v, --verbose     Enable verbose logging (debug level)
  -m, --mtu <bytes> Maximum Transmission Unit (default: 1500)
```

The application reads packets from stdin using the binary protocol format. Each packet specifies its own source and destination IP address and port.

**MTU Configuration**:

- Default: 1500 bytes (standard Ethernet)
- Range: 576-9000 bytes
- Affects maximum payload sizes for IPv4 and IPv6 packets
- IPv4 max payload = MTU - 20 (IP header) - 8 (UDP header)
- IPv6 max payload = MTU - 40 (IP header) - 8 (UDP header)

### Version Information

Check the installed version:

```bash
./udp-sender -V
# Or: ./udp-sender -version
# Output: udp-sender version v1.0.0
```

## Examples

### Sending Multiple Packets

The application reads packets from stdin using the binary protocol:

```bash
# Generate and send 100 IPv4 packets to 192.168.1.100:514 (using sudo)
go run packet-generator.go -count 100 -dest-ip 192.168.1.100 -dest-port 514 | \
  sudo ./udp-sender

# Or with CAP_NET_RAW capability (Linux only, no sudo needed)
sudo setcap cap_net_raw+ep ./udp-sender
go run packet-generator.go -count 100 -dest-ip 192.168.1.100 -dest-port 514 | \
  ./udp-sender

# Generate and send 50 IPv6 packets
go run packet-generator.go -ipv6 -base-ip "2001:db8::1" -dest-ip "2001:db8::100" \
  -dest-port 8080 -count 50 | sudo ./udp-sender

# Custom base IP and port (IPv4)
go run packet-generator.go -base-ip 192.168.1.10 -base-port 1000 \
  -dest-ip 192.168.1.100 -dest-port 514 -count 50 | sudo ./udp-sender

# Use jumbo frames (9000 byte MTU)
go run packet-generator.go -count 100 -dest-ip 192.168.1.100 -dest-port 514 | \
  sudo ./udp-sender -m 9000

# Custom MTU for specific network
go run packet-generator.go -count 100 -dest-ip 192.168.1.100 -dest-port 514 | \
  sudo ./udp-sender -m 1400

# Save packets to file for later
go run packet-generator.go -count 1000 -dest-ip 192.168.1.100 -dest-port 514 > packets.bin
cat packets.bin | sudo ./udp-sender

# IPv6 packets to file
go run packet-generator.go -ipv6 -base-ip "fe80::1" -dest-ip "::1" \
  -dest-port 8080 -count 1000 > ipv6-packets.bin
cat ipv6-packets.bin | sudo ./udp-sender
```

See [PROTOCOL.md](PROTOCOL.md) for complete protocol specification, field details, error handling, and examples in Python, Node.js, and Go.

## Development

### Running Tests

**Using Make (Recommended)**:

```bash
# Run all tests (non-root tests only, ~81% coverage)
make test

# Run all tests with root privileges (~91% coverage)
sudo make test-root

# Generate coverage report (requires root for full coverage)
sudo make coverage-root

# View coverage report
open coverage.html  # macOS
# Or: xdg-open coverage.html  # Linux
```

**Using Go directly**:

```bash
# Run all tests (non-root tests only)
go test -v ./...

# Run all tests with root (requires -count=1 to bypass cache)
sudo go test -v -count=1 ./...

# Run tests with coverage
sudo go test -v -race -count=1 -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

**Note**:

- Tests that require root privileges will be skipped automatically when not running with sudo
- Non-root tests achieve ~81% coverage; full coverage (~91%) requires root privileges
- The Makefile automatically includes `-count=1` for root tests to bypass Go's test cache

### Running Linter

```bash
make lint
```

### Makefile Targets

The Makefile provides convenient targets for common tasks:

| Target | Description |
|--------|-------------|
| `make build` | Build the application (use `VERSION=v1.0.0` to set version) |
| `make test` | Run tests without root (~81% coverage) |
| `make test-root` | Run all tests with root privileges (~91% coverage) |
| `make coverage` | Generate coverage report (without root) |
| `make coverage-root` | Generate coverage report with root (recommended) |
| `make lint` | Run golangci-lint |
| `make clean` | Clean build artifacts |
| `make run` | Build and run (requires root) |
| `make deps` | Download and verify dependencies |
| `make help` | Show all available targets |

**Examples**:

```bash
# Build with a specific version
make build VERSION=v1.0.0

# Run full test suite with coverage
sudo make coverage-root

# Clean and rebuild
make clean build
```

## How It Works

See [DESIGN.md](./DESIGN.md) for details of the design.

## Security Considerations

⚠️ **Important Security Notes**:

1. **Privilege Requirement**: Raw sockets require root/admin privileges for security reasons
   - **Linux**: Use `CAP_NET_RAW` capability instead of root (recommended)
   - **macOS**: Must use `sudo` (capabilities not supported)
2. **Network Abuse**: IP spoofing can be used for network attacks. Use responsibly.
3. **Legal Implications**: Spoofing IP addresses may be illegal in some jurisdictions
4. **Firewall Rules**: Many networks block or filter spoofed packets
5. **Testing Only**: This tool is intended for testing and educational purposes

### Why Use CAP_NET_RAW Instead of Root?

Running as root grants full system access, which is a security risk. Using Linux capabilities provides:

- **Principle of Least Privilege**: Only grants raw socket access
- **Reduced Attack Surface**: Compromised process can't modify system files
- **Better Security Posture**: Limits damage if the application is exploited

```bash
# Set capability once after building
sudo setcap cap_net_raw+ep ./udp-sender

# Remove capability if needed
sudo setcap -r ./udp-sender
```

### Legitimate Use Cases

- Network testing and debugging
- Load testing with simulated sources
- Security research (with permission)
- Protocol development
- Educational purposes

## Troubleshooting

### "operation not permitted" error

**Problem**: `Failed to create raw socket: operation not permitted`

**Solutions**:

1. **Run with sudo** (works on Linux and macOS):

   ```bash
   sudo ./udp-sender
   ```

2. **Use CAP_NET_RAW capability** (Linux only, more secure):

   ```bash
   sudo setcap cap_net_raw+ep ./udp-sender
   ./udp-sender  # Now works without sudo
   ```

To verify capabilities are set:

```bash
getcap ./udp-sender
# Should output: ./udp-sender = cap_net_raw+ep
```

### Packets not received

**Possible causes**:

1. **Firewall**: Check firewall rules on both sender and receiver
2. **Network filtering**: ISPs and routers may drop spoofed packets
3. **Routing**: Spoofed source IPs may not have valid routes
4. **Checksum issues**: Verify packet construction is correct
5. **MTU exceeded**: Packets larger than MTU limits are automatically dropped (see logs)

**Debug with tcpdump**:

```bash
# On receiver
sudo tcpdump -i any -n udp port 8080 -v
```

### Packets being dropped (MTU errors)

**Problem**: Logs show "Packet dropped due to MTU limit" errors

**Cause**: Packet payloads exceed the Maximum Transmission Unit.

With the default MTU of 1500 bytes:

- IPv4: Maximum 1472 bytes (1500 - 20 - 8)
- IPv6: Maximum 1452 bytes (1500 - 40 - 8)

**Solutions**:

1. **Increase MTU** if your network supports it:

   ```bash
   # Use jumbo frames for larger payloads
   cat packets.bin | sudo ./udp-sender -m 9000
   ```

2. **Reduce payload size** in your packet generator:

   ```bash
   # Ensure payloads are within limits
   go run packet-generator.go -count 100 -payload-size 1400
   ```

3. **Check final statistics** for dropped packet count:

   ```json
   {
     "level": "info",
     "message": "Stream complete",
     "packets_sent": 98,
     "packets_dropped": 2,
     "bytes_sent": 142856
   }
   ```

4. **Monitor error logs** for specific packets that were dropped:

   ```bash
   # Filter for MTU errors
   ./udp-sender | grep "MTU limit"
   ```

**Note**: The application automatically validates and drops oversized packets to prevent fragmentation issues with IP spoofing.

### Tests skipped

**Problem**: Most tests show "SKIP" status

**Solution**: Tests requiring root are automatically skipped. Run with sudo:

```bash
sudo go test -v ./...
```

## CI/CD

The project uses GitHub Actions for continuous integration:

### Workflows

- **Test Job**: Runs on Go 1.21 and 1.22
  - Note: Root-required tests are skipped in CI
- **Build Job**: Verifies compilation
- **Lint Job**: Runs golangci-lint

### GitHub Actions Note

Raw socket tests are skipped in GitHub Actions CI because:

1. GitHub runners don't provide root access
2. Network isolation in containers prevents raw socket usage

For full testing, run locally with sudo.

## Project Structure

```text
.
├── .github/
│   └── workflows/
│       ├── ci.yml               # CI workflow (test, build, lint, benchmark)
│       └── release.yml          # Release automation workflow
├── .gitignore                   # Git ignore rules
├── .golangci.yml                # Linter configuration
├── AUTHORS                      # Project authors and contributors
├── constants.go                 # Shared protocol constants (magic bytes, MTU limits)
├── DESIGN.md                    # Class design documentation
├── Dockerfile                   # Container image definition
├── go.mod                       # Go module definition
├── helpers_test.go              # Common test helpers (requireRoot, requireNonRoot)
├── LICENSE                      # MIT License
├── logger.go                    # Structured ND-JSON logger
├── logger_test.go               # Logger tests (100% coverage)
├── LOGGING.md                   # Logging documentation
├── main.go                      # Application entry point and CLI
├── main_test.go                 # CLI and application tests
├── Makefile                     # Build automation (test, build, lint, coverage)
├── packet.go                    # Packet construction (IP/UDP headers, checksums)
├── packet_test.go               # Packet construction tests (100% coverage)
├── packet-generator.go          # Utility to generate binary packet streams
├── protocol.go                  # Stream protocol processing and validation
├── protocol_test.go             # Protocol parsing tests (98.4% coverage)
├── PROTOCOL.md                  # Binary stream protocol specification
├── README.md                    # This file
├── RELEASING.md                 # Release process documentation
├── scripts/
│   ├── postinstall.sh           # Post-install script for packages
│   └── preremove.sh             # Pre-removal script for packages
├── sender.go                    # UDPSender class and PacketSender interface
├── sender_test.go               # UDPSender tests (91.7% coverage)
└── TESTING.md                   # Testing strategy and guidelines
```

### Code Organization

The codebase is organized into focused modules:

**Core Application**:

- **main.go** - Command-line interface and application entry point
- **sender.go** - Core UDPSender class with PacketSender interface and MTU validation
- **packet.go** - Low-level packet construction (IPv4/IPv6 headers, UDP headers, checksums)
- **protocol.go** - Stream protocol processing, validation, and MTU error handling
- **constants.go** - Shared protocol constants (magic bytes, MTU limits)
- **logger.go** - Structured ND-JSON logging implementation

**Testing**:

- **helpers_test.go** - Common test helpers (privilege checking, mocks)
- **main_test.go** - CLI and application logic tests
- **sender_test.go** - UDPSender integration and unit tests
- **packet_test.go** - Packet construction and checksum tests
- **protocol_test.go** - Protocol parsing and validation tests
- **logger_test.go** - Logging functionality tests

**Utilities & Documentation**:

- **packet-generator.go** - CLI tool to generate test packet streams
- **Makefile** - Build automation and common development tasks
- **Dockerfile** - Container image for isolated execution
- **DESIGN.md**, **PROTOCOL.md**, **TESTING.md**, **LOGGING.md**, **RELEASING.md** - Documentation

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Guidelines

- Maintain test coverage
- Follow Go best practices
- Update documentation for new features
- Test with and without root privileges

### Commit Message Format

This project uses [Conventional Commits](https://www.conventionalcommits.org/) for automatic changelog generation. When you create a release, commit messages are automatically parsed and categorized in the release notes.

**Format**: `<type>(<scope>): <description>`

#### Supported Types

| Type | Emoji | Description | Example |
|------|-------|-------------|---------|
| `feat` | ✨ | New features | `feat: add IPv6 support` |
| `fix` | 🐛 | Bug fixes | `fix: correct checksum calculation` |
| `docs` | 📚 | Documentation changes | `docs: update README with examples` |
| `perf` | ⚡ | Performance improvements | `perf: optimize packet sending` |
| `refactor` | ♻️ | Code refactoring | `refactor: simplify header building` |
| `test` | ✅ | Test additions/changes | `test: add benchmark tests` |
| `build` | 🔧 | Build system changes | `build: update Go version to 1.22` |
| `ci` | 🔧 | CI/CD changes | `ci: add performance benchmarks` |
| `chore` | 🔧 | Other changes | `chore: update dependencies` |

The scope is optional but encouraged (e.g., `feat(ipv6): add support for IPv6 packets`).

**Examples**:

```bash
git commit -m "feat: add support for variable payload sizes"
git commit -m "fix(checksum): handle odd-length packets correctly"
git commit -m "docs: add contribution guidelines"
git commit -m "perf: reduce memory allocations in packet building"
```

**Non-conventional commits** are also supported and will appear in the changelog without an emoji prefix.

### Testing Your Changes

**Using Make (Recommended)**:

```bash
# Run all tests with full coverage
sudo make test-root

# Generate and view coverage report
sudo make coverage-root
open coverage.html
```

**Using Go directly**:

```bash
# Run all tests (requires -count=1 to bypass cache)
sudo go test -v -count=1 ./...

# Run tests with coverage
sudo go test -v -race -count=1 -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Building for All Platforms

```bash
# Linux x64
GOOS=linux GOARCH=amd64 go build -o udp-sender-linux-x64

# Linux ARM64
GOOS=linux GOARCH=arm64 go build -o udp-sender-linux-arm64

# macOS x64 (Intel)
GOOS=darwin GOARCH=amd64 go build -o udp-sender-darwin-x64

# macOS ARM64 (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o udp-sender-darwin-arm64
```

### Creating a Release

See [RELEASING.md](RELEASING.md) for detailed release instructions.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

Copyright (c) 2025 Cribl, Inc.

## Disclaimer

This software is provided for educational and testing purposes only. Users are responsible for ensuring their use complies with applicable laws and regulations. The authors assume no liability for misuse of this software.
