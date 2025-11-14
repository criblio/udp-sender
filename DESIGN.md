# UDPSender Class Design

## Overview

The `UDPSender` is implemented as a Go class using idiomatic Go patterns: structs with methods, interfaces, and proper encapsulation. The key design feature is **per-packet source and destination address specification**, allowing complete dynamic control of both source and destination IP and port for each transmitted packet. The implementation supports both IPv4 and IPv6 through separate raw sockets.

```mermaid
graph TD
    A[Client Code] -->|Creates| B[UDPSender Instance]
    B -->|Implements| C[PacketSender Interface]
    B -->|Manages| D[Raw Socket]
    A -->|Calls Send with<br/>different sources| B
    B -->|Builds Packet| E[IP Header]
    B -->|Builds Packet| F[UDP Header]
    B -->|Builds Packet| G[Payload]
    E -->|Combines| H[Complete Packet]
    F -->|Combines| H
    G -->|Combines| H
    H -->|Sends via| D
    D -->|Transmits to| I[Network]
    
    style B fill:#e1f5ff
    style C fill:#fff4e1
    style D fill:#ffe1e1
```

## Class Structure

### Interface Definition

```go
type PacketSender interface {
    Send(message string, srcIP net.IP, srcPort uint16, destIP net.IP, destPort uint16) (int, error)
    Close() error
}
```

**Design Decision**: Both source and destination IP and port are specified per packet in the `Send()` method for maximum flexibility. Uses proper types (`net.IP` and `uint16`) instead of strings for type safety and efficiency.

```mermaid
classDiagram
    class PacketSender {
        <<interface>>
        +Send(message string, srcIP net.IP, srcPort uint16, destIP net.IP, destPort uint16) (int, error)
        +Close() error
    }
    
    class UDPSender {
        -int fdIPv4
        -int fdIPv6
        -int maxPayloadIPv4
        -int maxPayloadIPv6
        +Send(message string, srcIP net.IP, srcPort uint16, destIP net.IP, destPort uint16) (int, error)
        +Close() error
        -buildPacket(payload []byte, srcIP net.IP, srcPort uint16, destIP net.IP, destPort uint16) []byte
        -buildIPv4Header(dataLen int, srcIP net.IP, destIP net.IP) []byte
        -buildIPv6Header(dataLen int, srcIP net.IP, destIP net.IP) []byte
        -buildUDPHeader(payload []byte, srcIP net.IP, srcPort uint16, destIP net.IP, destPort uint16, isIPv6 bool) []byte
        -calculateUDPChecksum(udpHeader []byte, payload []byte, srcIP net.IP, destIP net.IP, isIPv6 bool) uint16
    }
    
    PacketSender <|.. UDPSender : implements
```

### Class Implementation

```go
type UDPSender struct {
    fdIPv4         int  // Private: raw socket file descriptor for IPv4
    fdIPv6         int  // Private: raw socket file descriptor for IPv6
    maxPayloadIPv4 int  // Private: maximum payload size for IPv4 based on MTU
    maxPayloadIPv6 int  // Private: maximum payload size for IPv6 based on MTU
}
```

**Key Design Philosophy**: The struct stores only what's absolutely necessary (socket file descriptors and MTU-based payload limits). Neither source nor destination IP and port are stored. All addressing information is passed as parameters to the `Send()` method, enabling complete dynamic control of both source and destination per packet.

## Object-Oriented Principles

### 1. Encapsulation

**Private Fields**: All struct fields use lowercase names, making them package-private:

```go
fdIPv4         int  // Not accessible outside the package - IPv4 raw socket
fdIPv6         int  // Not accessible outside the package - IPv6 raw socket
maxPayloadIPv4 int  // Not accessible outside the package - MTU-based IPv4 payload limit
maxPayloadIPv6 int  // Not accessible outside the package - MTU-based IPv6 payload limit
```

**No Getters**: Neither source nor destination IP/port are stored in the struct, so there are no getter methods. Both source and destination are specified per call to `Send()` for complete per-packet control.

### 2. Constructor Pattern

Factory function to create and initialize instances:

```go
func NewUDPSender(maxPayloadIPv4, maxPayloadIPv6 int) (*UDPSender, error)
```

**Parameters**:
- `maxPayloadIPv4`: Maximum payload size for IPv4 packets (based on MTU - 20 - 8)
- `maxPayloadIPv6`: Maximum payload size for IPv6 packets (based on MTU - 40 - 8)

Benefits:

- Validation before object creation
- Complex initialization logic
- Returns errors instead of invalid objects
- Resource acquisition (creates both IPv4 and IPv6 raw sockets)
- MTU configuration for payload validation
- No destination needed (both source and destination specified per packet)

### 3. Interface Implementation

The class implements the `PacketSender` interface, enabling:

- **Polymorphism**: Multiple implementations can satisfy the interface
- **Dependency Injection**: Functions can accept `PacketSender` instead of concrete type
- **Testing**: Easy to create mock implementations
- **Future Extensions**: New sender types (TCP, mock, etc.)

Compile-time verification:

```go
var _ PacketSender = (*UDPSender)(nil)
```

### 4. Method Types

**Public Methods** (exported):

- `Send(message string, srcIP net.IP, srcPort uint16, destIP net.IP, destPort uint16) (int, error)` - Send UDP packet with specified source and destination
- `Close() error` - Release resources (closes both IPv4 and IPv6 sockets)

**Private Methods** (package-private):

- `buildPacket(payload []byte, srcIP net.IP, srcPort uint16, destIP net.IP, destPort uint16) []byte`
- `buildIPv4Header(dataLen int, srcIP net.IP, destIP net.IP) []byte`
- `buildIPv6Header(dataLen int, srcIP net.IP, destIP net.IP) []byte`
- `buildUDPHeader(payload []byte, srcIP net.IP, srcPort uint16, destIP net.IP, destPort uint16, isIPv6 bool) []byte`
- `calculateUDPChecksum(udpHeader []byte, payload []byte, srcIP net.IP, destIP net.IP, isIPv6 bool) uint16`

**Key Design**: All packet-building methods accept both source and destination as parameters. Nothing is stored in the struct except the socket file descriptors.

```mermaid
sequenceDiagram
    participant Client
    participant UDPSender
    participant buildPacket
    participant buildIPv4Header
    participant buildUDPHeader
    participant RawSocket
    participant Network
    
    Client->>UDPSender: Send(payload, srcIP, srcPort, destIP, destPort)
    UDPSender->>buildPacket: buildPacket(payload, srcIP, srcPort, destIP, destPort)
    buildPacket->>buildIPv4Header: buildIPv4Header(len, srcIP, destIP)
    buildIPv4Header-->>buildPacket: IP header bytes
    buildPacket->>buildUDPHeader: buildUDPHeader(payload, srcIP, srcPort, destIP, destPort, isIPv6)
    Note over buildUDPHeader: Calculate checksum
    buildUDPHeader-->>buildPacket: UDP header bytes
    buildPacket-->>UDPSender: Complete packet
    UDPSender->>RawSocket: syscall.Sendto(packet, destAddr)
    RawSocket->>Network: Transmit packet
    Network-->>Client: Bytes sent / error
```

### 5. SOLID Principles

#### Single Responsibility

- UDPSender: Manages UDP packet sending with raw sockets
- Separate methods for each concern (IP header, UDP header, checksums)

#### Open/Closed

- Open for extension: Interface allows new implementations
- Closed for modification: Core functionality is stable

#### Liskov Substitution

- Any `PacketSender` implementation can be used interchangeably
- UDPSender maintains the interface contract

#### Interface Segregation

- Minimal interface with only essential methods
- Clients depend only on what they need

#### Dependency Inversion

- Code depends on `PacketSender` interface, not concrete class
- High-level code doesn't depend on low-level implementation

```mermaid
graph TB
    subgraph "Single Responsibility"
        A[UDPSender] -->|Manages| B[Raw Socket]
        A -->|Delegates| C[buildIPHeader]
        A -->|Delegates| D[buildUDPHeader]
        A -->|Delegates| E[calculateChecksum]
    end
    
    subgraph "Open/Closed & Liskov Substitution"
        F[PacketSender Interface] -.->|implements| G[UDPSender]
        F -.->|implements| H[MockSender]
        F -.->|implements| I[TCPSender]
        J[Client Code] -->|depends on| F
    end
    
    subgraph "Interface Segregation"
        K[Minimal Interface] -->|Send| L[Core Method]
        K -->|Close| M[Cleanup Method]
        K -->|Getters| N[Read-only Access]
    end
    
    style F fill:#fff4e1
    style G fill:#e1f5ff
    style A fill:#e1f5ff
```

## Usage Patterns

### Direct Instantiation

```go
import "github.com/criblio/udp-sender/constants"

// Option 1: Use constants from the constants package
sender, err := NewUDPSender(constants.MaxPayloadIPv4, constants.MaxPayloadIPv6)

// Option 2: Calculate max payload sizes based on custom MTU (e.g., 1500 bytes)
maxPayloadIPv4 := 1500 - constants.IPv4HeaderSize - constants.UDPHeaderSize  // 1472 bytes
maxPayloadIPv6 := 1500 - constants.IPv6HeaderSize - constants.UDPHeaderSize  // 1452 bytes

// Create sender with MTU-based payload limits (creates both IPv4 and IPv6 sockets)
sender, err := NewUDPSender(maxPayloadIPv4, maxPayloadIPv6)
if err != nil {
    log.Fatal(err)
}
defer sender.Close()

// Specify source and destination per packet
srcIP := net.ParseIP("10.0.0.1")
destIP := net.ParseIP("192.168.1.100")
n, err := sender.Send("Hello, World!", srcIP, 12345, destIP, 514)
```

### Dynamic Spoofing

```go
sender, _ := NewUDPSender(1472, 1452)  // Standard MTU limits
defer sender.Close()

// Send from different sources to different destinations
sender.Send("Packet 1", net.ParseIP("10.0.0.1"), 5001, net.ParseIP("192.168.1.100"), 514)
sender.Send("Packet 2", net.ParseIP("10.0.0.2"), 5002, net.ParseIP("192.168.1.101"), 514)
sender.Send("Packet 3", net.ParseIP("192.168.1.5"), 6000, net.ParseIP("10.0.0.1"), 8080)

// IPv6 works too
sender.Send("IPv6 Packet", net.ParseIP("2001:db8::1"), 5000, net.ParseIP("2001:db8::100"), 8080)
```

### Interface-Based

```go
func sendPacket(ps PacketSender, message string, srcIP net.IP, srcPort uint16, destIP net.IP, destPort uint16) error {
    _, err := ps.Send(message, srcIP, srcPort, destIP, destPort)
    return err
}

sender, _ := NewUDPSender(1472, 1452)  // Standard MTU limits
srcIP := net.ParseIP("10.0.0.1")
destIP := net.ParseIP("192.168.1.100")
sendPacket(sender, "Using interface", srcIP, 12345, destIP, 514)
```

### Polymorphic Collections

```go
var senders []PacketSender
senders = append(senders, udpSender1)
senders = append(senders, udpSender2)

// Each packet can have different source and destination
destIP := net.ParseIP("192.168.1.100")
for i, sender := range senders {
    srcIP := net.ParseIP(fmt.Sprintf("10.0.0.%d", i+1))
    srcPort := uint16(5000 + i)
    sender.Send("Broadcast message", srcIP, srcPort, destIP, 514)
}
```

## Testing Benefits

### Mockable Interface

```go
type MockSender struct {
    sentMessages []string
    sentFromIPs  []net.IP
    sentToIPs    []net.IP
}

func (m *MockSender) Send(msg string, srcIP net.IP, srcPort uint16, destIP net.IP, destPort uint16) (int, error) {
    m.sentMessages = append(m.sentMessages, msg)
    m.sentFromIPs = append(m.sentFromIPs, srcIP)
    m.sentToIPs = append(m.sentToIPs, destIP)
    return len(msg), nil
}

func (m *MockSender) Close() error { return nil }
```

### Test Through Interface

```go
func TestSendLogic(t *testing.T) {
    mock := &MockSender{}
    var sender PacketSender = mock
    
    // Test without needing root privileges
    srcIP := net.ParseIP("10.0.0.1")
    destIP := net.ParseIP("192.168.1.100")
    n, err := sender.Send("test", srcIP, 5000, destIP, 514)
    
    // Verify
    if err != nil || n != 4 {
        t.Errorf("unexpected result")
    }
    if len(mock.sentMessages) != 1 || mock.sentMessages[0] != "test" {
        t.Errorf("message not sent correctly")
    }
}
```

## Design Decisions

### Why Private Fields?

1. **Data Integrity**: Prevent external modification of internal state
2. **Future-Proof**: Can change internal representation without breaking API
3. **Validation**: Control how data is set and accessed
4. **Immutability**: Read-only access through getters

### Why Interface?

1. **Flexibility**: Easy to add new implementations
2. **Testing**: Mock implementations for unit tests
3. **Decoupling**: Code depends on abstraction, not concrete type
4. **Documentation**: Interface serves as contract

### Why Constructor Function?

1. **Validation**: Check inputs before creating object
2. **Resource Management**: Acquire system resources (raw socket)
3. **Error Handling**: Return errors for invalid configurations
4. **Initialization**: Complex setup logic in one place

## Encapsulation Architecture

```mermaid
graph TB
    subgraph "Public API - External Interface"
        A[PacketSender Interface]
        B[NewUDPSender Constructor]
        C[Public Methods:<br/>Send, Close, Getters]
    end
    
    subgraph "Implementation - Package Private"
        D[UDPSender Struct]
        E[Private Fields:<br/>fd, destIP, destPort]
        F[Private Methods:<br/>buildPacket, buildHeaders]
    end
    
    subgraph "System Resources"
        G[Raw Socket FD]
        H[Network Stack]
        I[Operating System]
    end
    
    A -.->|implements| D
    B -->|creates| D
    C -->|access via| D
    D -->|uses| E
    D -->|calls| F
    F -->|writes to| G
    G -->|syscalls| H
    H -->|kernel| I
    
    style A fill:#fff4e1
    style D fill:#e1f5ff
    style G fill:#ffe1e1
```

## Future Extensions

The class design allows for easy extensions:

### Alternative Implementations

```go
type TCPSender struct { ... }      // TCP-based sender
type QUICSender struct { ... }     // QUIC-based sender
type MockSender struct { ... }     // Testing mock
```

All can implement `PacketSender` interface.

### Additional Methods

```go
func (s *UDPSender) SetTTL(ttl uint8) error
func (s *UDPSender) SetTOS(tos uint8) error
func (s *UDPSender) GetStatistics() SendStats
```

Can be added without breaking existing code.

### Composition

```go
type RateLimitedSender struct {
    sender  PacketSender
    limiter *rate.Limiter
}

func (r *RateLimitedSender) Send(msg string, srcIP net.IP, srcPort uint16, destIP net.IP, destPort uint16) (int, error) {
    r.limiter.Wait(context.Background())
    return r.sender.Send(msg, srcIP, srcPort, destIP, destPort)
}

func (r *RateLimitedSender) Close() error {
    return r.sender.Close()
}
```

Wrap existing implementation with new behavior.

### Extension Architecture

```mermaid
graph TB
    subgraph "Interface Layer"
        A[PacketSender Interface]
    end
    
    subgraph "Core Implementations"
        B[UDPSender]
        C[TCPSender]
        D[QUICSender]
        E[MockSender]
    end
    
    subgraph "Decorators/Wrappers"
        F[RateLimitedSender<br/>wraps PacketSender]
        G[LoggingSender<br/>wraps PacketSender]
        H[MetricsSender<br/>wraps PacketSender]
    end
    
    subgraph "Application Layer"
        I[Stream Processor]
        J[Packet Generator]
        K[Test Suite]
    end
    
    A -.->|implements| B
    A -.->|implements| C
    A -.->|implements| D
    A -.->|implements| E
    A -.->|wraps| F
    A -.->|wraps| G
    A -.->|wraps| H
    
    I -->|uses| A
    J -->|uses| A
    K -->|uses| A
    
    F -->|delegates to| A
    G -->|delegates to| A
    H -->|delegates to| A
    
    style A fill:#fff4e1
    style B fill:#e1f5ff
    style E fill:#ffe1e1
```

## Conclusion

The `UDPSender` class demonstrates proper object-oriented design in Go:

- ✅ Encapsulation with private fields
- ✅ Interface-based abstraction
- ✅ Constructor pattern for initialization
- ✅ Getter methods for controlled access
- ✅ SOLID principles adherence
- ✅ Testable and extensible design

This design provides a clean, maintainable, and professional API for UDP packet sending with IP spoofing capabilities.

## References

- [RFC 791 - Internet Protocol](https://tools.ietf.org/html/rfc791)
- [RFC 768 - User Datagram Protocol](https://tools.ietf.org/html/rfc768)
- [RFC 1071 - Computing the Internet Checksum](https://tools.ietf.org/html/rfc1071)

