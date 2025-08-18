# gh-notify

A CLI tool that monitors GitHub notifications and sends desktop alerts for new ones.

## Features

- 🔔 **Desktop Notifications**: Get real-time alerts for new GitHub notifications
- 🔄 **Smart Caching**: Avoids duplicate notifications with local cache management
- 🔐 **gh CLI Integration**: Reuses your existing `gh` authentication
- ⚡ **Systemd Service**: Easy setup for automatic background monitoring
- 🧹 **Auto Cleanup**: Manages cache size and removes old entries automatically
- 📊 **Status Monitoring**: Check service status and recent notifications

## Installation

### Prerequisites

- Go 1.24 or later
- `gh` CLI tool installed and authenticated
- Linux desktop environment with notify-send (for desktop notifications)

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

# Include all notifications (read and unread)
gh-notify sync --all

# List cached notifications
gh-notify list

# Clear notification cache
gh-notify clear

# Check service status
gh-notify status
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

## Configuration

### Cache Location

Notifications are cached at `~/.cache/gh-notify/notifications.json`

### Cache Settings

- **Maximum entries**: 500 notifications
- **Retention period**: 30 days
- **Automatic cleanup**: On each sync

### Custom Cache Directory

```bash
gh-notify sync --cache-dir /path/to/custom/cache
```

## Notification Types

The tool handles various GitHub notification reasons:

- 🎯 **assign**: You were assigned to an issue/PR
- 💬 **mention**: You were mentioned in a comment
- 👥 **team_mention**: Your team was mentioned
- 🔍 **review_requested**: Review requested on a PR
- 🚨 **security_alert**: Security vulnerability detected
- 📝 **comment**: New comment on subscribed issue/PR
- 🔄 **state_change**: Issue/PR state changed

## Examples

### Basic Workflow

```bash
# First time setup
gh-notify sync                    # Initial sync
gh-notify install-service         # Install automatic service

# Check everything is working
gh-notify status                  # Verify service status
gh-notify list                    # View cached notifications
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
- [notificator](https://github.com/0xAX/notificator) for desktop notifications