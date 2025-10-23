# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- `make release` command for automated releases via GoReleaser

## [1.2.1] - 2025-10-24

### Fixed
- GraphQL API rate limit exhaustion by implementing hourly rate limiting for star fetching
- Waybar mode making redundant API calls by using cached star data

## [1.2.0] - 2025-10-23

### Added
- GoReleaser configuration for automated Linux releases (amd64 & arm64)
- Version injection via ldflags (version, commit, build date)
- Enhanced version command output showing commit and build date
- Makefile build targets with automatic version detection

### Changed
- Version is now injected at build time instead of hardcoded

## [1.1.0] - 2025-01-23

### Added
- **Star tracking feature**: Monitor stars on your repositories in real-time
  - GraphQL API integration for efficient star fetching
  - Concurrent processing with 6-worker pool for multiple repositories
  - Pagination support to handle repositories with many stars
  - Smart time-based filtering to fetch only recent stars
  - Desktop notifications for new stars
  - Waybar integration showing star count
- Rate limit handling with exponential backoff (2s, 4s, 8s delays)
- 30-second timeout protection for API queries
- Comprehensive error classification system (rate_limit, timeout, permission, not_found, network, unknown)
- Structured logging with zerolog for better observability
- Performance tracking and metrics
- Makefile with test automation and mock generation
- Extensive test coverage:
  - Unit tests for GitHub client, cache, sync command
  - Mock-based testing with gomock
  - Error classification tests
  - Rate limit retry behavior tests

### Changed
- Refactored `internal/github/client.go` from 512 lines into focused modules:
  - `client.go`: Core client and authentication (73 lines)
  - `notifications.go`: Notification fetching logic
  - `stars.go`: Star tracking with worker pool
  - `errors.go`: Error classification
  - `util.go`: URL conversion utilities
- Migrated to interface-based design for better testability
- All timestamps now use UTC consistently for GitHub API compatibility
- Replaced `--include-stars` with `--exclude-stars` flag for better UX
- Cache now persists star events with deduplication

### Fixed
- Race condition in `LastEventSync` update that could cause data loss
- UTC time handling inconsistencies across codebase
- GraphQL query injection vulnerabilities
- Redundant condition in `GetRepositoryOwner`
- Missing ID field in StarEvent for proper deduplication

## [1.0.0] - 2025-01-18

### Added
- Initial implementation of gh-notify CLI tool
- GitHub notification monitoring via gh CLI authentication
- Smart cache management for unread notifications
- Desktop notifications using notify-send
- Waybar integration with notification count display
- Nerd font icons for visual indicators
- Clickable notifications that open GitHub in browser
- URL extraction and browser navigation
- Systemd service support for background monitoring
- Commands:
  - `sync`: Fetch and cache notifications
  - `list`: Display cached notifications
  - `open`: Open notification in browser
  - `clear`: Clear cache
  - `install-service`: Install systemd service
  - `status`: Show cache status

### Features
- Leverages existing gh CLI authentication
- Maintains local cache of notifications requiring attention
- Only shows GitHub icon in Waybar when notifications exist
- Unread-only cache management to reduce noise
- MIT License

[Unreleased]: https://github.com/bnema/gh-notify/compare/v1.2.1...HEAD
[1.2.1]: https://github.com/bnema/gh-notify/compare/v1.2.0...v1.2.1
[1.2.0]: https://github.com/bnema/gh-notify/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/bnema/gh-notify/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/bnema/gh-notify/releases/tag/v1.0.0
