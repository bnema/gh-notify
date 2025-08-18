package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

const serviceTemplate = `[Unit]
Description=GitHub Notification Monitor
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
ExecStart={{.BinaryPath}} sync
StandardOutput=journal
StandardError=journal
# Restart on failure with delay
Restart=on-failure
RestartSec=30s

[Install]
WantedBy=default.target`

const timerTemplate = `[Unit]
Description=Run GitHub notification sync every {{.Interval}}
Requires=gh-notify.service

[Timer]
OnBootSec=30s
OnUnitActiveSec={{.Interval}}
AccuracySec=1s
Persistent=true

[Install]
WantedBy=timers.target`

type SystemdManager struct {
	serviceDir string
}

type TemplateData struct {
	BinaryPath string
	Interval   string
}

func NewSystemdManager() (*SystemdManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	serviceDir := filepath.Join(homeDir, ".config", "systemd", "user")
	
	return &SystemdManager{
		serviceDir: serviceDir,
	}, nil
}

func (sm *SystemdManager) Install(interval time.Duration, dryRun bool) error {
	// Get binary path
	binaryPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Prepare template data
	data := TemplateData{
		BinaryPath: binaryPath,
		Interval:   formatDuration(interval),
	}

	if dryRun {
		return sm.showDryRun(data)
	}

	// Create service directory
	if err := os.MkdirAll(sm.serviceDir, 0755); err != nil {
		return fmt.Errorf("failed to create systemd user directory: %w", err)
	}

	// Generate and write service file
	if err := sm.writeServiceFile(data); err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}

	// Generate and write timer file
	if err := sm.writeTimerFile(data); err != nil {
		return fmt.Errorf("failed to write timer file: %w", err)
	}

	// Reload systemd daemon
	if err := sm.runSystemctl("daemon-reload"); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}

	// Enable and start timer
	if err := sm.runSystemctl("enable", "gh-notify.timer"); err != nil {
		return fmt.Errorf("failed to enable timer: %w", err)
	}

	if err := sm.runSystemctl("start", "gh-notify.timer"); err != nil {
		return fmt.Errorf("failed to start timer: %w", err)
	}

	return nil
}

func (sm *SystemdManager) Uninstall() error {
	// Stop and disable timer
	sm.runSystemctl("stop", "gh-notify.timer")    // Don't fail if already stopped
	sm.runSystemctl("disable", "gh-notify.timer") // Don't fail if already disabled

	// Remove service and timer files
	serviceFile := filepath.Join(sm.serviceDir, "gh-notify.service")
	timerFile := filepath.Join(sm.serviceDir, "gh-notify.timer")

	os.Remove(serviceFile) // Don't fail if file doesn't exist
	os.Remove(timerFile)   // Don't fail if file doesn't exist

	// Reload daemon
	if err := sm.runSystemctl("daemon-reload"); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}

	return nil
}

func (sm *SystemdManager) Status() (string, error) {
	cmd := exec.Command("systemctl", "--user", "status", "gh-notify.timer", "--no-pager")
	output, _ := cmd.CombinedOutput()
	
	// systemctl status returns non-zero exit code for inactive services, but that's not an error for us
	return string(output), nil
}

func (sm *SystemdManager) writeServiceFile(data TemplateData) error {
	tmpl, err := template.New("service").Parse(serviceTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse service template: %w", err)
	}

	serviceFile := filepath.Join(sm.serviceDir, "gh-notify.service")
	file, err := os.Create(serviceFile)
	if err != nil {
		return fmt.Errorf("failed to create service file: %w", err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute service template: %w", err)
	}

	return nil
}

func (sm *SystemdManager) writeTimerFile(data TemplateData) error {
	tmpl, err := template.New("timer").Parse(timerTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse timer template: %w", err)
	}

	timerFile := filepath.Join(sm.serviceDir, "gh-notify.timer")
	file, err := os.Create(timerFile)
	if err != nil {
		return fmt.Errorf("failed to create timer file: %w", err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute timer template: %w", err)
	}

	return nil
}

func (sm *SystemdManager) runSystemctl(args ...string) error {
	cmdArgs := append([]string{"--user"}, args...)
	cmd := exec.Command("systemctl", cmdArgs...)
	
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("systemctl %s failed: %s", strings.Join(args, " "), string(output))
	}

	return nil
}

func (sm *SystemdManager) showDryRun(data TemplateData) error {
	fmt.Println("=== Dry Run: Service Installation ===")
	fmt.Printf("Service directory: %s\n", sm.serviceDir)
	fmt.Printf("Binary path: %s\n", data.BinaryPath)
	fmt.Printf("Sync interval: %s\n", data.Interval)
	fmt.Println()

	fmt.Println("--- gh-notify.service ---")
	tmpl, _ := template.New("service").Parse(serviceTemplate)
	tmpl.Execute(os.Stdout, data)
	
	fmt.Println("\n--- gh-notify.timer ---")
	tmpl, _ = template.New("timer").Parse(timerTemplate)
	tmpl.Execute(os.Stdout, data)
	
	fmt.Println("\n--- Commands to run ---")
	fmt.Println("systemctl --user daemon-reload")
	fmt.Println("systemctl --user enable gh-notify.timer")
	fmt.Println("systemctl --user start gh-notify.timer")

	return nil
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh", int(d.Hours()))
}

func (sm *SystemdManager) IsInstalled() bool {
	serviceFile := filepath.Join(sm.serviceDir, "gh-notify.service")
	timerFile := filepath.Join(sm.serviceDir, "gh-notify.timer")
	
	_, serviceExists := os.Stat(serviceFile)
	_, timerExists := os.Stat(timerFile)
	
	return serviceExists == nil && timerExists == nil
}