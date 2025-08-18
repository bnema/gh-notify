# gh-notify

A CLI tool that monitors GitHub notifications and sends desktop alerts for new ones.

## Features

- **Desktop Notifications**: Get real-time alerts for new GitHub notifications with clickable actions
- **Browser Integration**: Open notifications directly in your browser with numbered selection
- **Smart Caching**: Only stores unread notifications with automatic cleanup
- **gh CLI Integration**: Reuses your existing `gh` authentication
- **Systemd Service**: Easy setup for automatic background monitoring
- **Auto Cleanup**: Automatically removes read notifications and manages cache size
- **Status Monitoring**: Check service status and recent notifications
- **URL Handling**: Automatically converts GitHub API URLs to clickable web URLs
- **Waybar Integration**: JSON output with nerd font icons for status bar integration

## Installation

### Prerequisites

- Go 1.24 or later
- `gh` CLI tool installed and authenticated
- Linux desktop environment with `notify-send` (for clickable desktop notifications)

### Build from Source

```bash
git clone https://github.com/bnema/gh-notify.git
cd gh-notify
go build -o gh-notify
sudo mv gh-notify /usr/local/bin/
```

## Usage

### Basic Commands

```bash
# Sync notifications once (manual)
gh-notify sync

# Sync with verbose output
gh-notify sync --verbose

# List cached unread notifications (with numbers, types, and URLs)
gh-notify list

# Open a specific notification in browser (by number from list)
gh-notify open 1

# Clear notification cache
gh-notify clear

# Check service status
gh-notify status

# Output JSON for waybar integration
gh-notify sync --waybar-output
```

### Service Installation

Install as a systemd user service for automatic monitoring:

```bash
# Install service (runs every 10 seconds by default)
gh-notify install-service

# Install with custom interval
gh-notify install-service --interval 30s

# Check what would be installed (dry run)
gh-notify install-service --dry-run

# Uninstall service
gh-notify install-service --uninstall
```

### Service Management

```bash
# Check service status
systemctl --user status gh-notify.timer

# View logs
journalctl --user -u gh-notify.service -f

# Stop service
systemctl --user stop gh-notify.timer

# Start service
systemctl --user start gh-notify.timer
```

### Waybar Integration

For status bar integration with waybar:

```bash
# Add to your waybar config.jsonc
"custom/github": {
    "exec": "gh-notify sync --waybar-output",
    "interval": 60,
    "return-type": "json",
    "tooltip": true,
    "format": "{}",
    "on-click": "xdg-open https://github.com/notifications"
}
```

The waybar output includes:
- Notification count in parentheses (e.g., "(3)")
- Rich tooltip with repository grouping and nerd font icons
- Empty output when no notifications

## Configuration

### Cache Location

Notifications are cached at `~/.cache/gh-notify/notifications.json`

### Cache Settings

- **Content**: Only unread notifications (automatic cleanup)
- **Maximum entries**: 500 unread notifications
- **Retention period**: 30 days (safety fallback)
- **Automatic cleanup**: On each sync, removes notifications that are no longer unread

### Custom Cache Directory

```bash
gh-notify sync --cache-dir /path/to/custom/cache
```

### How It Works

The cache automatically stays small and relevant:
1. Each sync fetches only unread notifications from GitHub
2. Previously cached notifications that are no longer unread (handled on GitHub) are automatically removed
3. Only notifications requiring your attention remain in the cache
4. Fast loading and minimal storage usage

## Notification Types

The tool handles various GitHub notification reasons:

- **assign**: You were assigned to an issue/PR
- **mention**: You were mentioned in a comment
- **team_mention**: Your team was mentioned
- **review_requested**: Review requested on a PR
- **security_alert**: Security vulnerability detected
- **comment**: New comment on subscribed issue/PR
- **state_change**: Issue/PR state changed

## Examples

### Basic Workflow

```bash
# First time setup
gh-notify sync                    # Initial sync
gh-notify install-service         # Install automatic service

# Check everything is working
gh-notify status                  # Verify service status
gh-notify list                    # View cached unread notifications with URLs

# Open notifications in browser
gh-notify open 1                  # Opens first notification from list
gh-notify open 3                  # Opens third notification from list
```

### Troubleshooting

```bash
# Test authentication
gh auth status

# Debug sync issues
gh-notify sync --verbose

# Check service logs
journalctl --user -u gh-notify.service --since "1 hour ago"

# Reset cache if needed
gh-notify clear
```

## Architecture

```
gh-notify/
├── cmd/                    # CLI commands
├── internal/
│   ├── cache/             # Notification cache management
│   ├── github/            # GitHub API client
│   ├── notifier/          # Desktop notification system
│   └── service/           # Systemd service management
└── main.go
```

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [GitHub CLI](https://cli.github.com/) for authentication
- [Cobra](https://github.com/spf13/cobra) for CLI framework
- Linux `notify-send` for desktop notifications