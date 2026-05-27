# GitHub Actions Workflows

This directory contains GitHub Actions workflows for CI/CD.

## Workflows

### CI Workflow (`ci.yml`)

**Trigger**: Push to `master`/`main` branch, Pull Requests

**Purpose**: Continuous Integration

**Jobs**:

- **Test**: Runs unit tests on Go 1.24 and 1.25 (matrix); integration tests run in a Go 1.24 container only
- **Build**: Verifies the project builds successfully
- **Lint**: Runs golangci-lint for code quality

**Status Badge**:

```markdown
[![CI](https://github.com/criblio/udp-sender/actions/workflows/ci.yml/badge.svg)](https://github.com/criblio/udp-sender/actions/workflows/ci.yml)
```

### Release Workflow (`release.yml`)

**Trigger**: Push tags matching `v*.*.*` (e.g., `v1.0.0`)

**Purpose**: Automated binary releases

**Jobs**:

- **Build**: Cross-compiles binaries for:
  - Linux x64
  - Linux ARM64
  - macOS x64 (Intel)
  - macOS ARM64 (Apple Silicon)
- **Archive**: Creates `.tar.gz` archives for each binary
- **Packages**: Builds DEB and RPM packages for Linux
- **Checksum**: Generates SHA256 checksums for all artifacts
- **Docker**: Builds and pushes multi-arch container images to GitHub Container Registry
- **Release**: Creates GitHub release with all artifacts

**Outputs**:

- `udp-sender-v1.0.0-linux-x64.tar.gz`
- `udp-sender-v1.0.0-linux-arm64.tar.gz`
- `udp-sender-v1.0.0-darwin-x64.tar.gz`
- `udp-sender-v1.0.0-darwin-arm64.tar.gz`
- `udp-sender_1.0.0_x64.deb`, `udp-sender_1.0.0_arm64.deb`
- `udp-sender-1.0.0-x64.rpm`, `udp-sender-1.0.0-arm64.rpm`
- Individual SHA256 checksums (`.sha256`) for each file
- Combined `checksums.txt`
- Docker images: `ghcr.io/criblio/udp-sender:1.0.0`, `ghcr.io/criblio/udp-sender:latest`

## Creating a Release

1. Ensure all changes are committed and pushed
2. Create and push a tag:

   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

3. GitHub Actions automatically builds and releases
4. Check the release at: <https://github.com/criblio/udp-sender/releases>

## Permissions

The workflows require the following permissions:

- **CI**: Read access to repository
- **Release**:
  - Write access to `contents` for creating releases
  - Write access to `packages` for publishing container images

These are automatically granted by `GITHUB_TOKEN`.

## Docker Images

Container images are published to GitHub Container Registry (ghcr.io) with each release:

```bash
# Pull and run the latest version
docker pull ghcr.io/criblio/udp-sender:latest
cat packets.bin | docker run --rm -i --cap-add=NET_RAW ghcr.io/criblio/udp-sender:latest

# Pull a specific version
docker pull ghcr.io/criblio/udp-sender:1.0.0
```

**Tags:**

- `latest` - Most recent release
- `X.Y.Z` - Specific version (e.g., `1.0.0`)
- `X.Y` - Minor version (e.g., `1.0`)
- `X` - Major version (e.g., `1`)

**Architectures:**

- `linux/amd64`
- `linux/arm64`

View images at: <https://github.com/criblio/udp-sender/pkgs/container/udp-sender>

## Monitoring

View workflow runs at: <https://github.com/criblio/udp-sender/actions>
