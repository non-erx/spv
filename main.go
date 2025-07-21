package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

var Version = "v0.0.4"
var Commit = ""

const githubOwner = "non-erx"
const githubRepo = "spv"

type state int

const (
	listView state = iota
	addingName
	addingCommand
	addingDescription
	showingAbout
)

type tickMsg time.Time
type commitMsg string

type screenSession struct {
	id          string
	name        string
	status      string
	autostart   bool
	command     string
	description string
}

type model struct {
	sessions        []screenSession
	selected        int
	textInput       textinput.Model
	state           state
	width           int
	height          int
	tempName        string
	tempCommand     string
	tempDescription string
	cpuUsage        float64
	memUsage        float64
	commitMsg       string
	errorMsg        string
}

type Theme struct {
	HeaderBg       lipgloss.Color
	PanelBg        lipgloss.Color
	Border         lipgloss.Color
	Text           lipgloss.Color
	MutedText      lipgloss.Color
	Accent         lipgloss.Color
	SelectedBg     lipgloss.Color
	SelectedFg     lipgloss.Color
	StatusAttached lipgloss.Color
	StatusDetached lipgloss.Color
}

var themes = map[string]Theme{
	"slate": {
		HeaderBg:       lipgloss.Color("#1E293B"),
		PanelBg:        lipgloss.Color("#1E293B"),
		Border:         lipgloss.Color("#334155"),
		Text:           lipgloss.Color("#CBD5E1"),
		MutedText:      lipgloss.Color("#64748B"),
		Accent:         lipgloss.Color("#06B6D4"),
		SelectedBg:     lipgloss.Color("#06B6D4"),
		SelectedFg:     lipgloss.Color("#0F172A"),
		StatusAttached: lipgloss.Color("#10B981"),
		StatusDetached: lipgloss.Color("#F59E0B"),
	},
	"pink": {
		HeaderBg:       lipgloss.Color("#2A0A29"),
		PanelBg:        lipgloss.Color("#2A0A29"),
		Border:         lipgloss.Color("#5A1A59"),
		Text:           lipgloss.Color("#FAD4F9"),
		MutedText:      lipgloss.Color("#8A5A89"),
		Accent:         lipgloss.Color("#FF00FF"),
		SelectedBg:     lipgloss.Color("#FF00FF"),
		SelectedFg:     lipgloss.Color("#2A0A29"),
		StatusAttached: lipgloss.Color("#00FFAA"),
		StatusDetached: lipgloss.Color("#FFAA00"),
	},
	"forest": {
		HeaderBg:       lipgloss.Color("#1A2A1A"),
		PanelBg:        lipgloss.Color("#1A2A1A"),
		Border:         lipgloss.Color("#3A5A3A"),
		Text:           lipgloss.Color("#D4FAD4"),
		MutedText:      lipgloss.Color("#5A8A5A"),
		Accent:         lipgloss.Color("#00FF00"),
		SelectedBg:     lipgloss.Color("#00FF00"),
		SelectedFg:     lipgloss.Color("#1A2A1A"),
		StatusAttached: lipgloss.Color("#00FFAA"),
		StatusDetached: lipgloss.Color("#FFAA00"),
	},
	"mellow": {
		HeaderBg:       lipgloss.Color("#F0F8FF"),
		PanelBg:        lipgloss.Color("#F8F8FF"),
		Border:         lipgloss.Color("#ADD8E6"),
		Text:           lipgloss.Color("#4682B4"),
		MutedText:      lipgloss.Color("#87CEEB"),
		Accent:         lipgloss.Color("#6A5ACD"),
		SelectedBg:     lipgloss.Color("#B0C4DE"),
		SelectedFg:     lipgloss.Color("#191970"),
		StatusAttached: lipgloss.Color("#32CD32"),
		StatusDetached: lipgloss.Color("#FFD700"),
	},
	"arctic": {
		HeaderBg:       lipgloss.Color("#E0FFFF"),
		PanelBg:        lipgloss.Color("#F0FFFF"),
		Border:         lipgloss.Color("#B0E0E6"),
		Text:           lipgloss.Color("#2F4F4F"),
		MutedText:      lipgloss.Color("#696969"),
		Accent:         lipgloss.Color("#4682B4"),
		SelectedBg:     lipgloss.Color("#87CEFA"),
		SelectedFg:     lipgloss.Color("#1C1C1C"),
		StatusAttached: lipgloss.Color("#5F9EA0"),
		StatusDetached: lipgloss.Color("#FFA07A"),
	},
	"solarized": {
		HeaderBg:       lipgloss.Color("#002b36"),
		PanelBg:        lipgloss.Color("#073642"),
		Border:         lipgloss.Color("#586e75"),
		Text:           lipgloss.Color("#839496"),
		MutedText:      lipgloss.Color("#657b83"),
		Accent:         lipgloss.Color("#268bd2"),
		SelectedBg:     lipgloss.Color("#2aa198"),
		SelectedFg:     lipgloss.Color("#002b36"),
		StatusAttached: lipgloss.Color("#859900"),
		StatusDetached: lipgloss.Color("#b58900"),
	},
	"dracula": {
		HeaderBg:       lipgloss.Color("#282a36"),
		PanelBg:        lipgloss.Color("#282a36"),
		Border:         lipgloss.Color("#44475a"),
		Text:           lipgloss.Color("#f8f8f2"),
		MutedText:      lipgloss.Color("#6272a4"),
		Accent:         lipgloss.Color("#bd93f9"),
		SelectedBg:     lipgloss.Color("#ff79c6"),
		SelectedFg:     lipgloss.Color("#282a36"),
		StatusAttached: lipgloss.Color("#50fa7b"),
		StatusDetached: lipgloss.Color("#f1fa8c"),
	},
	"gruvbox": {
		HeaderBg:       lipgloss.Color("#282828"),
		PanelBg:        lipgloss.Color("#3c3836"),
		Border:         lipgloss.Color("#504945"),
		Text:           lipgloss.Color("#ebdbb2"),
		MutedText:      lipgloss.Color("#928374"),
		Accent:         lipgloss.Color("#fabd2f"),
		SelectedBg:     lipgloss.Color("#fe8019"),
		SelectedFg:     lipgloss.Color("#282828"),
		StatusAttached: lipgloss.Color("#b8bb26"),
		StatusDetached: lipgloss.Color("#d65d0e"),
	},
	"nord": {
		HeaderBg:       lipgloss.Color("#2E3440"),
		PanelBg:        lipgloss.Color("#3B4252"),
		Border:         lipgloss.Color("#4C566A"),
		Text:           lipgloss.Color("#D8DEE9"),
		MutedText:      lipgloss.Color("#4C566A"),
		Accent:         lipgloss.Color("#88C0D0"),
		SelectedBg:     lipgloss.Color("#81A1C1"),
		SelectedFg:     lipgloss.Color("#2E3440"),
		StatusAttached: lipgloss.Color("#A3BE8C"),
		StatusDetached: lipgloss.Color("#EBCB8B"),
	},
}

var (
	headerStyle, sidebarStyle, contentStyle, selectedStyle,
	statusAttachedStyle, statusDetachedStyle, accentStyle,
	footerStyle, inputStyle, aboutStyle, overflowStyle,
	mutedTextStyle, normalTextStyle, errorTextStyle lipgloss.Style
)

func applyTheme(name string) {
	theme, ok := themes[name]
	if !ok {
		theme = themes["slate"]
	}

	headerStyle = lipgloss.NewStyle().
		Foreground(theme.Text).Background(theme.HeaderBg).
		Border(lipgloss.NormalBorder()).BorderForeground(theme.Border).
		Padding(0, 2).Align(lipgloss.Center)

	sidebarStyle = lipgloss.NewStyle().
		Foreground(theme.Text).Background(theme.PanelBg).
		Padding(1, 2).Width(32).Border(lipgloss.NormalBorder()).
		BorderForeground(theme.Border)

	contentStyle = lipgloss.NewStyle().
		Foreground(theme.Text).Background(theme.PanelBg).
		Padding(1, 2).Width(44).Border(lipgloss.NormalBorder()).
		BorderForeground(theme.Border)

	selectedStyle = lipgloss.NewStyle().
		Foreground(theme.SelectedFg).Background(theme.SelectedBg).
		Bold(true).Padding(0, 1)

	statusAttachedStyle = lipgloss.NewStyle().
		Foreground(theme.StatusAttached).Bold(true)

	statusDetachedStyle = lipgloss.NewStyle().
		Foreground(theme.StatusDetached).Bold(true)

	accentStyle = lipgloss.NewStyle().
		Foreground(theme.Accent).Bold(true)

	footerStyle = lipgloss.NewStyle().
		Foreground(theme.MutedText).Background(theme.HeaderBg).
		Border(lipgloss.NormalBorder()).BorderForeground(theme.Border).
		Padding(0, 2).Align(lipgloss.Center)

	inputStyle = lipgloss.NewStyle().
		Foreground(theme.Accent).Background(theme.PanelBg).
		Border(lipgloss.RoundedBorder()).BorderForeground(theme.Border).
		Padding(1, 2).Width(40)

	aboutStyle = lipgloss.NewStyle().
		Foreground(theme.Text).Background(theme.PanelBg).
		Border(lipgloss.RoundedBorder()).BorderForeground(theme.Accent).
		Padding(2, 4).Align(lipgloss.Center)

	overflowStyle = lipgloss.NewStyle().
		Foreground(theme.MutedText).Italic(true)

	mutedTextStyle = lipgloss.NewStyle().
		Foreground(theme.MutedText)

	normalTextStyle = lipgloss.NewStyle().
		Foreground(theme.Text)

	errorTextStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("#FF0000")).
		Foreground(lipgloss.Color("#FFFFFF")).
		Padding(0, 1).
		Bold(true)
}

type Config struct {
	Theme string `json:"theme"`
}

type SessionEntry struct {
	Name        string `json:"name"`
	Command     string `json:"command"`
	Description string `json:"description"`
}

type SystemInfo struct {
	OS           string
	Distribution string
	InitSystem   string
}

var configDir, configFile, sessionFile, autostartFile string

func setupPaths() {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	configDir = filepath.Join(home, ".config", "spv")
	configFile = filepath.Join(configDir, "config.json")
	sessionFile = filepath.Join(configDir, "sessions.json")
	autostartFile = filepath.Join(configDir, "autostart.json")
	os.MkdirAll(configDir, os.ModePerm)
}

func loadTheme() string {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return "slate"
	}
	var cfg Config
	if json.Unmarshal(data, &cfg) != nil {
		return "slate"
	}
	if _, ok := themes[cfg.Theme]; !ok {
		return "slate"
	}
	return cfg.Theme
}

func saveTheme(name string) error {
	cfg := Config{Theme: name}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configFile, data, 0644)
}

func readConfig(file string) ([]SessionEntry, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return []SessionEntry{}, nil
		}
		return nil, err
	}
	var entries []SessionEntry
	err = json.Unmarshal(data, &entries)
	return entries, err
}

func writeConfig(file string, entries []SessionEntry) error {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(file, data, 0644)
}

func addSessionEntry(name, command, description string) error {
	entries, err := readConfig(sessionFile)
	if err != nil {
		return err
	}
	entries = append(entries, SessionEntry{
		Name:        name,
		Command:     command,
		Description: description,
	})
	return writeConfig(sessionFile, entries)
}

func addAutostartEntry(name, command string) error {
	entries, err := readConfig(autostartFile)
	if err != nil {
		return err
	}
	entries = append(entries, SessionEntry{Name: name, Command: command})
	return writeConfig(autostartFile, entries)
}

func removeEntry(file, name string) error {
	entries, err := readConfig(file)
	if err != nil {
		return err
	}
	var updatedEntries []SessionEntry
	for _, entry := range entries {
		if entry.Name != name {
			updatedEntries = append(updatedEntries, entry)
		}
	}
	return writeConfig(file, updatedEntries)
}

func detectSystem() SystemInfo {
	info := SystemInfo{OS: runtime.GOOS}
	switch runtime.GOOS {
	case "linux":
		info.Distribution = detectLinuxDistribution()
		info.InitSystem = detectInitSystem()
	case "darwin":
		info.Distribution = "macOS"
		info.InitSystem = "launchd"
	default:
		info.Distribution = "unknown"
		info.InitSystem = "unknown"
	}
	return info
}

func detectLinuxDistribution() string {
	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "ID=") {
				return strings.Trim(strings.TrimPrefix(line, "ID="), "\"")
			}
		}
	}
	distroFiles := map[string]string{
		"/etc/redhat-release": "rhel",
		"/etc/debian_version": "debian",
		"/etc/arch-release":   "arch",
		"/etc/gentoo-release": "gentoo",
		"/etc/alpine-release": "alpine",
	}
	for file, distro := range distroFiles {
		if _, err := os.Stat(file); err == nil {
			return distro
		}
	}
	return "unknown"
}

func detectInitSystem() string {
	if _, err := os.Stat("/run/systemd/system"); err == nil {
		return "systemd"
	}
	if _, err := os.Stat("/sbin/openrc"); err == nil {
		return "openrc"
	}
	if _, err := os.Stat("/etc/init"); err == nil {
		return "upstart"
	}
	if _, err := os.Stat("/etc/init.d"); err == nil {
		return "sysvinit"
	}
	return "unknown"
}

func generateAutostartScriptContent(autostartSessions []SessionEntry) (string, error) {
	var script strings.Builder
	script.WriteString("#!/bin/bash\n")
	script.WriteString("sleep 15\n\n")

	for _, session := range autostartSessions {
		escapedCommand := strings.ReplaceAll(session.Command, `"`, `\"`)
		if session.Command == "shell" || session.Command == "" {
			script.WriteString(fmt.Sprintf("screen -dmS spv_%s\n", session.Name))
		} else {
			script.WriteString(fmt.Sprintf("screen -dmS spv_%s bash -c \"%s; exec bash\"\n", session.Name, escapedCommand))
		}
	}
	script.WriteString("\nexit 0\n")
	return script.String(), nil
}

func createLinuxAutostart(autostartSessions []SessionEntry, sysInfo SystemInfo) error {
	scriptContent, err := generateAutostartScriptContent(autostartSessions)
	if err != nil {
		return err
	}
	scriptPath := "/usr/local/bin/spv-autostart.sh"
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		return fmt.Errorf("failed to create autostart script %s: %v", scriptPath, err)
	}

	switch sysInfo.InitSystem {
	case "systemd":
		serviceContent := `[Unit]
Description=SPV Screen Session Autostart
After=multi-user.target
Wants=network-online.target
After=network-online.target

[Service]
Type=forking
User=root
ExecStart=/usr/local/bin/spv-autostart.sh
Restart=on-failure
RestartSec=5
RemainAfterExit=yes

[Install]
WantedBy=multi-user.target
`
		servicePath := "/etc/systemd/system/spv-autostart.service"
		if err := os.WriteFile(servicePath, []byte(serviceContent), 0644); err != nil {
			return fmt.Errorf("failed to create systemd service: %v", err)
		}
		if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
			return fmt.Errorf("failed to reload systemd: %v", err)
		}
		if err := exec.Command("systemctl", "enable", "spv-autostart.service").Run(); err != nil {
			return fmt.Errorf("failed to enable service: %v", err)
		}
		return nil
	case "sysvinit":
		initContent := fmt.Sprintf(`#!/bin/bash
# chkconfig: 2345 99 10
# description: SPV Screen Session Autostart
### BEGIN INIT INFO
# Provides: spv-autostart
# Required-Start: $remote_fs $syslog
# Required-Stop: $remote_fs $syslog
# Default-Start: 2 3 4 5
# Default-Stop: 0 1 6
# Short-Description: SPV Screen Session Autostart
# Description: Automatically starts screen sessions defined by SPV
### END INIT INFO

. /etc/init.d/functions

prog="spv-autostart.sh"
daemon="%s"
lockfile=/var/lock/subsys/$prog

start() {
    echo -n $"Starting $prog: "
    daemon --background $daemon
    retval=$?
    echo
    [ $retval -eq 0 ] && touch $lockfile
    return $retval
}

stop() {
    echo -n $"Stopping $prog: "
    killproc $daemon
    retval=$?
    echo
    [ $retval -eq 0 ] && rm -f $lockfile
    return $retval
}

restart() {
    stop
    start
}

case "$1" in
    start)
        start
        ;;
    stop)
        stop
        ;;
    restart)
        restart
        ;;
    status)
        status $daemon
        ;;
    *)
        echo "Usage: %%s {start|stop|restart|status}"
        exit 1
esac
exit 0
`, scriptPath)

		initScriptPath := "/etc/init.d/spv-autostart"
		if err := os.WriteFile(initScriptPath, []byte(initContent), 0755); err != nil {
			return fmt.Errorf("failed to create SysVinit script: %v", err)
		}
		switch sysInfo.Distribution {
		case "debian", "ubuntu":
			if err := exec.Command("update-rc.d", "spv-autostart", "defaults").Run(); err != nil {
				return fmt.Errorf("failed to enable service with update-rc.d: %v", err)
			}
		case "rhel", "centos", "fedora":
			if err := exec.Command("chkconfig", "--add", "spv-autostart").Run(); err != nil {
				return fmt.Errorf("failed to add service with chkconfig: %v", err)
			}
			if err := exec.Command("chkconfig", "spv-autostart", "on").Run(); err != nil {
				return fmt.Errorf("failed to enable service with chkconfig: %v", err)
			}
		}
		return nil
	case "openrc":
		rcScriptContent := fmt.Sprintf(`#!/sbin/openrc-run

name="spv-autostart"
description="SPV Screen Session Autostart"

depend() {
    need net
    after bootmisc
}

start() {
    ebegin "Starting SPV autostart sessions"
    %s
    eend $?
}

stop() {
    ebegin "Stopping SPV sessions"
    screen -ls | grep -E "spv_" | cut -d. -f1 | awk '{print $2}' | xargs -r -I {} screen -S {} -X quit
    eend $?
}
`, strings.ReplaceAll(scriptContent, "\n", "\n    "))

		rcScriptPath := "/etc/init.d/spv-autostart"
		if err := os.WriteFile(rcScriptPath, []byte(rcScriptContent), 0755); err != nil {
			return fmt.Errorf("failed to create OpenRC script: %v", err)
		}
		if err := exec.Command("rc-update", "add", "spv-autostart", "default").Run(); err != nil {
			return fmt.Errorf("failed to enable OpenRC service: %v", err)
		}
		return nil
	default:
		return fmt.Errorf("unsupported init system for autostart: %s", sysInfo.InitSystem)
	}
}

func removeLinuxAutostart(sysInfo SystemInfo) error {
	scriptPath := "/usr/local/bin/spv-autostart.sh"

	switch sysInfo.InitSystem {
	case "systemd":
		servicePath := "/etc/systemd/system/spv-autostart.service"
		exec.Command("systemctl", "stop", "spv-autostart.service").Run()
		exec.Command("systemctl", "disable", "spv-autostart.service").Run()
		os.Remove(servicePath)
		os.Remove(scriptPath)
		exec.Command("systemctl", "daemon-reload").Run()
		return nil
	case "sysvinit":
		initScriptPath := "/etc/init.d/spv-autostart"
		exec.Command("service", "spv-autostart", "stop").Run()
		switch sysInfo.Distribution {
		case "debian", "ubuntu":
			exec.Command("update-rc.d", "-f", "spv-autostart", "remove").Run()
		case "rhel", "centos", "fedora":
			exec.Command("chkconfig", "--del", "spv-autostart").Run()
		}
		os.Remove(initScriptPath)
		os.Remove(scriptPath)
		return nil
	case "openrc":
		rcScriptPath := "/etc/init.d/spv-autostart"
		exec.Command("rc-service", "spv-autostart", "stop").Run()
		exec.Command("rc-update", "del", "spv-autostart", "default").Run()
		os.Remove(rcScriptPath)
		os.Remove(scriptPath)
		return nil
	default:
		return fmt.Errorf("unsupported init system for autostart removal: %s", sysInfo.InitSystem)
	}
}

func updateAutostartScript(sessions []screenSession) error {
	sysInfo := detectSystem()

	if sysInfo.OS == "darwin" || sysInfo.OS == "windows" {
		return fmt.Errorf("autostart is not supported on your OS")
	}

	autostartSessions := []SessionEntry{}
	for _, session := range sessions {
		if session.autostart {
			autostartSessions = append(autostartSessions, SessionEntry{
				Name:        session.name,
				Command:     session.command,
				Description: session.description,
			})
		}
	}

	if len(autostartSessions) == 0 {
		return removeLinuxAutostart(sysInfo)
	} else {
		return createLinuxAutostart(autostartSessions, sysInfo)
	}
}

func toggleSessionAutostart(sessionName string) error {
	autostartEntries, err := readConfig(autostartFile)
	if err != nil {
		return err
	}

	sessionEntries, err := readConfig(sessionFile)
	if err != nil {
		return err
	}

	var sessionEntry *SessionEntry
	for _, entry := range sessionEntries {
		if entry.Name == sessionName {
			sessionEntry = &entry
			break
		}
	}

	if sessionEntry == nil {
		return fmt.Errorf("session not found")
	}

	var updatedEntries []SessionEntry
	found := false
	for _, entry := range autostartEntries {
		if entry.Name != sessionName {
			updatedEntries = append(updatedEntries, entry)
		} else {
			found = true
		}
	}

	if !found {
		updatedEntries = append(updatedEntries, *sessionEntry)
	}

	if err := writeConfig(autostartFile, updatedEntries); err != nil {
		return fmt.Errorf("failed to update autostart configuration file: %v", err)
	}

	updatedManagedSessions := getScreens()
	return updateAutostartScript(updatedManagedSessions)
}

func getScreens() []screenSession {
	cmd := exec.Command("screen", "-ls")
	outputBytes, err := cmd.CombinedOutput()

	sessionEntries, _ := readConfig(sessionFile)
	sessionMap := make(map[string]SessionEntry)
	for _, entry := range sessionEntries {
		sessionMap[entry.Name] = entry
	}

	autostartEntries, _ := readConfig(autostartFile)
	autostartMap := make(map[string]bool)
	for _, entry := range autostartEntries {
		autostartMap[entry.Name] = true
	}

	var sessions []screenSession

	if err == nil || strings.Contains(strings.ToLower(string(outputBytes)), "socket") {
		lines := strings.Split(string(outputBytes), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.Contains(line, ".") &&
				(strings.Contains(line, "Attached") ||
					strings.Contains(line, "Detached")) {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					fullName := parts[0]
					nameParts := strings.Split(fullName, ".")

					id := ""
					name := fullName
					if len(nameParts) >= 2 {
						id = nameParts[0]
						name = strings.Join(nameParts[1:], ".")
					}

					if !strings.HasPrefix(name, "spv_") {
						continue
					}
					displayName := strings.TrimPrefix(name, "spv_")

					status := "unknown"
					if strings.Contains(line, "Attached") {
						status = "attached"
					} else if strings.Contains(line, "Detached") {
						status = "detached"
					}

					session := screenSession{
						id:          id,
						name:        displayName,
						status:      status,
						command:     "shell",
						description: "A standard interactive shell session.",
						autostart:   autostartMap[displayName],
					}

					if entry, ok := sessionMap[displayName]; ok {
						session.command = entry.Command
						session.description = entry.Description
					}

					sessions = append(sessions, session)
				}
			}
		}
	}

	return sessions
}

func getSystemStats() (float64, float64) {
	cpuPercent, _ := cpu.Percent(0, false)
	memStat, _ := mem.VirtualMemory()

	var cpuUsage float64
	if len(cpuPercent) > 0 {
		cpuUsage = cpuPercent[0]
	}

	return cpuUsage, memStat.UsedPercent
}

func createScreenSession(name, command, description string) error {
	fullSessionName := fmt.Sprintf("spv_%s", name)

	var cmd *exec.Cmd
	if command == "shell" || command == "" {
		cmd = exec.Command("screen", "-dmS", fullSessionName)
	} else {
		cmd = exec.Command("screen", "-dmS", fullSessionName, "bash", "-c", command+"; exec bash")
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to create screen session: %v", err)
	}

	return addSessionEntry(name, command, description)
}

func fetchLatestCommit() tea.Msg {
	url := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/commits",
		githubOwner,
		githubRepo,
	)
	resp, err := http.Get(url)
	if err != nil {
		return commitMsg("")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return commitMsg("")
	}

	type CommitInfo struct {
		Commit struct {
			Message string `json:"message"`
		} `json:"commit"`
	}

	var commits []CommitInfo
	if err := json.Unmarshal(body, &commits); err != nil || len(commits) == 0 {
		return commitMsg("")
	}

	firstLine := strings.Split(commits[0].Commit.Message, "\n")[0]
	return commitMsg(firstLine)
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		tea.Tick(time.Second, func(t time.Time) tea.Msg {
			return tickMsg(t)
		}),
		fetchLatestCommit,
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tickMsg:
		if m.state == listView {
			m.cpuUsage, m.memUsage = getSystemStats()
			m.sessions = getScreens()
			if m.selected >= len(m.sessions) && len(m.sessions) > 0 {
				m.selected = len(m.sessions) - 1
			} else if len(m.sessions) == 0 {
				m.selected = 0
			}
		}
		return m, tea.Tick(time.Second, func(t time.Time) tea.Msg {
			return tickMsg(t)
		})

	case commitMsg:
		m.commitMsg = string(msg)
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch m.state {
		case listView:
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit

			case "up":
				if len(m.sessions) > 0 && m.selected > 0 {
					m.selected--
				}

			case "down":
				if len(m.sessions) > 0 && m.selected < len(m.sessions)-1 {
					m.selected++
				}

			case "a":
				m.state = addingName
				m.textInput.Placeholder = "Enter session name"
				m.textInput.Focus()
				return m, textinput.Blink

			case "k":
				if len(m.sessions) > 0 && m.selected < len(m.sessions) {
					session := m.sessions[m.selected]
					fullScreenName := fmt.Sprintf("spv_%s", session.name)

					exec.Command("screen", "-S", fullScreenName, "-X", "quit").Run()

					removeEntry(sessionFile, session.name)
					removeEntry(autostartFile, session.name)

					m.sessions = getScreens()
					if m.selected >= len(m.sessions) && len(m.sessions) > 0 {
						m.selected = len(m.sessions) - 1
					} else if len(m.sessions) == 0 {
						m.selected = 0
					}

					if err := updateAutostartScript(m.sessions); err != nil {
						m.errorMsg = "Issues creating autostart script"
					}
				}

			case "r":
				m.cpuUsage, m.memUsage = getSystemStats()
				m.sessions = getScreens()

			case "enter":
				if len(m.sessions) > 0 && m.selected < len(m.sessions) {
					session := m.sessions[m.selected]
					fullScreenName := fmt.Sprintf("%s.spv_%s", session.id, session.name)
					return m, tea.ExecProcess(
						exec.Command("screen", "-r", fullScreenName),
						nil,
					)
				}

			case "t":
				if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
					m.errorMsg = "Autostart is not supported on macOS/Windows"
					go func() {
						time.Sleep(3 * time.Second)
					}()
					return m, nil
				}

				if len(m.sessions) > 0 && m.selected < len(m.sessions) {
					session := m.sessions[m.selected]
					if err := toggleSessionAutostart(session.name); err != nil {
						m.errorMsg = "Issues creating autostart script"
					}
					m.sessions = getScreens()
				}

			case "?":
				m.state = showingAbout
			}

		case showingAbout:
			m.state = listView

		case addingName:
			switch msg.String() {
			case "enter":
				m.tempName = m.textInput.Value()
				m.textInput.SetValue("")
				if m.tempName == "" {
					exec.Command("screen").Start()
					m.state = listView
					m.textInput.Blur()
					m.sessions = getScreens()
				} else {
					m.state = addingCommand
					m.textInput.Placeholder = "Enter command (blank for shell)"
				}
				return m, textinput.Blink

			case "esc":
				m.state = listView
				m.textInput.Blur()
				m.textInput.SetValue("")
			}

		case addingCommand:
			switch msg.String() {
			case "enter":
				m.tempCommand = m.textInput.Value()
				m.textInput.SetValue("")
				if m.tempCommand == "" {
					m.tempCommand = "shell"
					m.tempDescription = "A standard interactive shell session."
					if err := createScreenSession(m.tempName, m.tempCommand, m.tempDescription); err != nil {
						m.errorMsg = "Issues creating autostart script"
					}
					m.state = listView
					m.textInput.Blur()
					m.sessions = getScreens()
				} else {
					m.state = addingDescription
					m.textInput.Placeholder = "Enter description (optional)"
				}
				return m, textinput.Blink

			case "esc":
				m.state = listView
				m.textInput.Blur()
				m.textInput.SetValue("")
			}

		case addingDescription:
			switch msg.String() {
			case "enter":
				m.tempDescription = m.textInput.Value()
				if m.tempDescription == "" {
					m.tempDescription = "A screen session running a custom command."
				}

				if err := createScreenSession(m.tempName, m.tempCommand, m.tempDescription); err != nil {
					m.errorMsg = "Issues creating autostart script"
				}

				m.state = listView
				m.textInput.Blur()
				m.textInput.SetValue("")
				m.sessions = getScreens()

			case "esc":
				m.state = listView
				m.textInput.Blur()
				m.textInput.SetValue("")
			}
		}
	}

	switch m.state {
	case addingName, addingCommand, addingDescription:
		m.textInput, cmd = m.textInput.Update(msg)
	}

	return m, cmd
}

func (m model) View() string {
	if m.width == 0 || m.height < 10 {
		return "Initializing or window too small..."
	}

	switch m.state {
	case showingAbout:
		versionDisplay := Version
		if Commit != "" {
			versionDisplay = fmt.Sprintf("%s (%s)", Version, Commit)
		}

		about := aboutStyle.Render(
			accentStyle.Render("SPV "+versionDisplay+" - Screen Process Viewer") + "\n" +
				"Minimal TUI for Linux screen management\n\n" +
				accentStyle.Render("Author:") + "\n" +
				"Git: " +
				lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")).Render("@non-erx") +
				"\n" +
				"Bluesky: " +
				lipgloss.NewStyle().Foreground(lipgloss.Color("#0EA5E9")).Render("@mean2ya") +
				"\n" +
				"LinkedIn: " +
				lipgloss.NewStyle().Foreground(lipgloss.Color("#0077B5")).Render("@symonchuk") +
				"\n" +
				"SoundCloud: " +
				lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5500")).Render("@mean2ya") +
				"\n\n" +
				(func() string {
					if m.commitMsg != "" && Commit != m.commitMsg && Commit != "" {
						return "\n" + mutedTextStyle.Render("Latest commit on GitHub: ") + accentStyle.Render(m.commitMsg) + "\n"
					}
					if m.commitMsg != "" && Commit == "" {
						return "\n" + mutedTextStyle.Render("Latest commit fetched: ") + accentStyle.Render(m.commitMsg) + "\n"
					}
					return ""
				}()) +
				lipgloss.NewStyle().Foreground(lipgloss.Color("#64748B")).Render("Press any key to continue..."),
		)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, about)

	case addingName, addingCommand, addingDescription:
		prompt := "Session Name"
		if m.state == addingCommand {
			prompt = "Command"
		} else if m.state == addingDescription {
			prompt = "Description (optional)"
		}

		content := lipgloss.JoinVertical(
			lipgloss.Center,
			accentStyle.Render(prompt+":"),
			"",
			m.textInput.View(),
			"",
			mutedTextStyle.Render("Enter to confirm • Esc to cancel"),
		)

		box := inputStyle.Render(content)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
	}

	versionStr := Version
	if Commit != "" {
		versionStr = fmt.Sprintf("%s (%s)", Version, Commit)
	} else {
		if m.commitMsg != "" {
			versionStr = fmt.Sprintf("%s (%s)", Version, m.commitMsg)
		}
	}

	header := headerStyle.Width(80).Render(
		lipgloss.JoinHorizontal(
			lipgloss.Left,
			lipgloss.NewStyle().Width(40).Render(versionStr),
			lipgloss.NewStyle().Width(40).Render(fmt.Sprintf(
				"cpu %.1f%% ram %.1f%% [%d]",
				m.cpuUsage,
				m.memUsage,
				len(m.sessions),
			)),
		),
	)

	mainPanelContentHeight := m.height - 10
	dynamicSidebarStyle := sidebarStyle.Copy().Height(mainPanelContentHeight)
	dynamicContentStyle := contentStyle.Copy().Height(mainPanelContentHeight)

	var sidebar strings.Builder
	sidebar.WriteString(accentStyle.Render("sessions") + "\n\n")

	listViewportHeight := mainPanelContentHeight - 2
	if listViewportHeight < 1 {
		listViewportHeight = 1
	}

	start := 0
	end := len(m.sessions)

	if len(m.sessions) > listViewportHeight {
		if m.selected >= start+listViewportHeight {
			start = m.selected - listViewportHeight + 1
		} else if m.selected < start {
			start = m.selected
		}
		end = start + listViewportHeight
		if end > len(m.sessions) {
			end = len(m.sessions)
		}
	}

	hasMoreAbove := start > 0
	hasMoreBelow := end < len(m.sessions)

	if hasMoreAbove {
		sidebar.WriteString(overflowStyle.Render("... ↑ more above") + "\n")
	}

	if len(m.sessions) == 0 {
		sidebar.WriteString(mutedTextStyle.Render("no active sessions") + "\n")
	} else {
		for i := start; i < end; i++ {
			session := m.sessions[i]
			sessionDisplay := session.name
			if session.autostart {
				sessionDisplay += " ●"
			}
			if i == m.selected {
				sidebar.WriteString(selectedStyle.Render(sessionDisplay) + "\n")
			} else {
				sidebar.WriteString(sessionDisplay + "\n")
			}
		}
	}

	if hasMoreBelow {
		sidebar.WriteString(overflowStyle.Render("... ↓ more below"))
	}

	var content strings.Builder
	if len(m.sessions) > 0 && m.selected < len(m.sessions) {
		session := m.sessions[m.selected]

		content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#F1F5F9")).Bold(true).Render(session.name) + "\n")

		statusStyle := statusDetachedStyle
		statusText := "detached"
		if session.status == "attached" {
			statusStyle = statusAttachedStyle
			statusText = "attached"
		}
		content.WriteString(statusStyle.Render(statusText) + "\n\n")

		content.WriteString(accentStyle.Render("ID: ") + session.id + "\n")
		content.WriteString(accentStyle.Render("Autostart: "))
		if session.autostart {
			content.WriteString("On\n\n")
		} else {
			content.WriteString("Off\n\n")
		}

		content.WriteString(accentStyle.Render("command") + "\n")
		content.WriteString(mutedTextStyle.Render(session.command) + "\n\n")

		content.WriteString(accentStyle.Render("description") + "\n")
		content.WriteString(mutedTextStyle.Render(session.description))

	} else {
		content.WriteString(mutedTextStyle.Render("No session selected") + "\n\n" + normalTextStyle.Render("Press 'a' to create a new session"))
	}

	if m.errorMsg != "" {
		content.WriteString("\n\n" + errorTextStyle.Render(m.errorMsg))
		go func() {
			time.Sleep(3 * time.Second)
		}()
	}

	main := lipgloss.JoinHorizontal(
		lipgloss.Top,
		dynamicSidebarStyle.Render(sidebar.String()),
		dynamicContentStyle.Render(content.String()),
	)

	footer := footerStyle.Width(80).Render("↑↓ navigate • enter attach • a add • k kill • r refresh • t toggle autostart • ? about • q quit")

	layout := lipgloss.JoinVertical(
		lipgloss.Center,
		header,
		main,
		footer,
	)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, layout)
}

func main() {
	setupPaths()

	if len(os.Args) == 3 && os.Args[1] == "theme" {
		themeName := os.Args[2]
		if _, ok := themes[themeName]; !ok {
			fmt.Printf("Error: Theme '%s' not found.\n", themeName)
			fmt.Println("Available themes: slate, pink, forest, mellow, arctic, solarized, dracula, gruvbox, nord")
			os.Exit(1)
		}
		if err := saveTheme(themeName); err != nil {
			fmt.Printf("Error saving theme: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Theme set to '%s'.\n", themeName)
		os.Exit(0)
	}

	applyTheme(loadTheme())

	sessions := getScreens()
	cpuUsage, memUsage := getSystemStats()

	ti := textinput.New()
	ti.CharLimit = 150
	ti.Width = 35

	m := model{
		sessions:  sessions,
		selected:  0,
		textInput: ti,
		state:     listView,
		cpuUsage:  cpuUsage,
		memUsage:  memUsage,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
