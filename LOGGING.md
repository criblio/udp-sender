# Logging

## Overview

`udp-sender` uses structured logging with newline-delimited JSON (ND-JSON) format. All logs are written to `stdout`, while the application reads packet data from `stdin`. This keeps logs parseable without interfering with the input stream.

By default, the application logs at the **info** level. Use the `-v` or `--verbose` flag to enable **debug** level logging for more detailed output including progress updates.

## Log Format

Each log entry is a single line of JSON with the following core fields:

- **level**: Log level (debug, info, warn, error, fatal)
- **message**: Human-readable log message

Additional fields are added directly at the top level of the JSON object. If a field name conflicts with one of the core fields (`level`, `message`), it will be prefixed with an underscore (e.g., `_level`, `_message`).

Ordering guarantee: the `level` and `message` fields always appear first (in that order) in each JSON log line. All other fields follow afterward in unspecified order.

### Example Log Entries

```json
{"level":"info","message":"udp-sender starting","version":"v0.0.6"}
{"level":"info","message":"Reading packets from stdin","protocol":"[Magic(3)][Flags(1)][SrcIP(4/16)][DestIP(4/16)][SrcPort(2)][DestPort(2)][PayloadLen(2)][Payload(N)]"}
{"level":"debug","message":"Progress update","bytes_sent":8192,"packets_sent":100}
{"level":"error","message":"Error creating UDP sender","error":"permission denied"}
{"level":"info","message":"Stream complete","bytes_sent":1048576,"packets_sent":1000}
```

Note how fields like `protocol`, `packets_sent`, `bytes_sent`, and `error` are at the top level, not nested in a `fields` object.

## Controlling Log Levels

Use the `-v` or `--verbose` flag to enable debug-level logging:

```bash
# Normal operation (info level and above)
cat packets.bin | sudo ./udp-sender

# Verbose mode (debug level and above, includes progress updates)
cat packets.bin | sudo ./udp-sender -v
cat packets.bin | sudo ./udp-sender --verbose
```

By default, only **info**, **warn**, **error**, and **fatal** messages are logged. The **debug** level is filtered out unless verbose mode is enabled.

## Log Levels

### debug
Detailed diagnostic information, including progress updates every 100 packets. Only shown when using `-v` or `--verbose` flag. Used for troubleshooting and development.

**Example:**
```json
{"level":"debug","message":"Progress update","bytes_sent":8192,"packets_sent":100}
```

**Note:** This level is filtered by default. Enable with `-v` flag.

### info
General informational messages about normal operation.

**Example:**
```json
{"level":"info","message":"Reading packets from stdin","protocol":"[Magic(3)][Flags(1)][SrcIP(4/16)][DestIP(4/16)][SrcPort(2)][DestPort(2)][PayloadLen(2)][Payload(N)]"}
```

### warn
Warning messages indicating potential issues that don't prevent operation.

### error
Error messages indicating failures that prevent specific operations.

**Example:**
```json
{"level":"error","message":"Error creating UDP sender","error":"permission denied"}
```

### fatal
Critical errors that cause the application to exit.

**Example:**
```json
{"level":"fatal","message":"Error processing stream","error":"invalid magic number"}
```

## Parsing Logs

### Using `jq`

Since logs are in ND-JSON format, you can easily parse and filter them using `jq`:

```bash
# Show only error and fatal logs
cat packets.bin | sudo ./udp-sender | jq 'select(.level == "error" or .level == "fatal")'

# Extract specific fields
cat packets.bin | sudo ./udp-sender | jq '{level: .level, message: .message}'

# Filter by field values (requires -v for progress updates)
cat packets.bin | sudo ./udp-sender -v | jq 'select(.packets_sent > 1000)'

# Show only debug logs (requires -v flag)
cat packets.bin | sudo ./udp-sender -v | jq 'select(.level == "debug")'

# Pretty print all logs
cat packets.bin | sudo ./udp-sender | jq '.'
```

### Using `grep` and JSON parsers

```bash
# Filter logs containing specific text
./udp-sender | grep -i "error"

# Filter by log level
./udp-sender | grep '"level":"error"'
```

### In Python

```python
import json
import sys

for line in sys.stdin:
    try:
        log = json.loads(line)
        if log.get('level') in ('error', 'fatal'):
            print(f"{log['message']}")
            # All additional fields are at top level
            details = {k: v for k, v in log.items() 
                      if k not in ('level', 'message')}
            if details:
                print(f"  Details: {details}")
    except json.JSONDecodeError:
        pass
```

## Integration with Log Management Systems

The ND-JSON format is compatible with most modern log management systems:

- **Splunk**: Ingest as JSON format with sourcetype configuration
- **Elasticsearch**: Direct indexing via Filebeat or Logstash
- **Datadog**: Use the JSON log format parser
- **Cribl Stream**: Parse as JSON with the JSON parser
- **CloudWatch**: Use JSON format with log insights
- **Loki**: Use the json pipeline stage

## Common Log Patterns

### Startup
```json
{"level":"info","message":"udp-sender starting","version":"v0.0.6"}
```

### Progress Updates (every 100 packets, requires -v flag)
```json
{"level":"debug","message":"Progress update","bytes_sent":8192,"packets_sent":100}
```

### Stream Completion
```json
{"level":"info","message":"Stream complete","bytes_sent":1048576,"packets_sent":1000}
```

### Errors
```json
{"level":"error","message":"Error creating UDP sender","error":"permission denied"}
{"level":"fatal","message":"Error processing stream","error":"invalid magic number: got [0x00 0x00 0x00], expected [0xC1 0x21 0xB1]"}
```

## Benefits of ND-JSON Logging

1. **Machine-Readable**: Easy to parse and process programmatically
2. **Structured**: Fields are typed and queryable
3. **Streamable**: Each line is a complete, valid JSON object
4. **Compatible**: Works with most log aggregation and analysis tools
5. **Human-Readable**: Can be pretty-printed with `jq` or similar tools
6. **Efficient**: No special parsing required for multi-line logs

