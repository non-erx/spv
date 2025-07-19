package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

const version = "v0.0.1"
const githubOwner = "non-erx"
const githubRepo = "spv"

type state int

const (
	listView state = iota
	addingName
	addingCommand
	addingDescription
	addingAutostart
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
}

var (
	headerStyle, sidebarStyle, contentStyle, selectedStyle,
	statusAttachedStyle, statusDetachedStyle, accentStyle,
	footerStyle, inputStyle, aboutStyle, overflowStyle,
	mutedTextStyle, normalTextStyle lipgloss.Style
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
}

type Config struct {
	Theme string `json:"theme"`
}

type SessionEntry struct {
	Name        string `json:"name"`
	Command     string `json:"command"`
	Description string `json:"description"`
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

func getScreens() []screenSession {
	cmd := exec.Command("screen", "-ls")
	output, _ := cmd.Output()

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

	lines := strings.Split(string(output), "\n")
	var sessions []screenSession

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

				status := "unknown"
				if strings.Contains(line, "Attached") {
					status = "attached"
				} else if strings.Contains(line, "Detached") {
					status = "detached"
				}

				session := screenSession{
					id:          id,
					name:        name,
					status:      status,
					command:     "shell",
					description: "A standard interactive shell session.",
					autostart:   autostartMap[name],
				}

				if entry, ok := sessionMap[name]; ok {
					session.command = entry.Command
					session.description = entry.Description
				}

				sessions = append(sessions, session)
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
	switch msg := msg.(type) {
	case tickMsg:
		if m.state == listView {
			m.cpuUsage, m.memUsage = getSystemStats()
			m.sessions = getScreens()
			if m.selected >= len(m.sessions) && len(m.sessions) > 0 {
				m.selected = len(m.sessions) - 1
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
					screenName := fmt.Sprintf("%s.%s", session.id, session.name)
					exec.Command("screen", "-S", screenName, "-X", "quit").Run()
					removeEntry(sessionFile, session.name)
					removeEntry(autostartFile, session.name)
					m.sessions = getScreens()
					if m.selected >= len(m.sessions) && len(m.sessions) > 0 {
						m.selected = len(m.sessions) - 1
					}
				}

			case "r":
				m.cpuUsage, m.memUsage = getSystemStats()
				m.sessions = getScreens()

			case "enter":
				if len(m.sessions) > 0 && m.selected < len(m.sessions) {
					session := m.sessions[m.selected]
					screenName := fmt.Sprintf("%s.%s", session.id, session.name)
					return m, tea.ExecProcess(
						exec.Command("screen", "-r", screenName),
						nil,
					)
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
					exec.Command("screen", "-dmS", m.tempName).Start()
					addSessionEntry(
						m.tempName,
						"shell",
						"A standard interactive shell session.",
					)
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
				m.textInput.SetValue("")
				m.state = addingAutostart
				m.textInput.Placeholder = "Autostart on reboot? (y/n)"
				return m, textinput.Blink

			case "esc":
				m.state = listView
				m.textInput.Blur()
				m.textInput.SetValue("")
			}

		case addingAutostart:
			switch msg.String() {
			case "enter":
				autostart := strings.ToLower(m.textInput.Value()) == "y"
				exec.Command(
					"screen",
					"-dmS",
					m.tempName,
					"bash",
					"-c",
					m.tempCommand,
				).Start()
				addSessionEntry(m.tempName, m.tempCommand, m.tempDescription)
				if autostart {
					addAutostartEntry(m.tempName, m.tempCommand)
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

	var cmd tea.Cmd
	switch m.state {
	case addingName, addingCommand, addingDescription, addingAutostart:
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
		about := aboutStyle.Render(
			accentStyle.Render("SPV "+version+" - Screen Process Viewer") + "\n" +
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
				lipgloss.NewStyle().Foreground(lipgloss.Color("#64748B")).Render("Press any key to continue..."),
		)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, about)

	case addingName, addingCommand, addingDescription, addingAutostart:
		prompt := "Session Name"
		if m.state == addingCommand {
			prompt = "Command"
		} else if m.state == addingDescription {
			prompt = "Description (optional)"
		} else if m.state == addingAutostart {
			prompt = "Autostart on reboot? (y/n)"
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

	versionStr := version
	if m.commitMsg != "" {
		versionStr = fmt.Sprintf("%s – %s", version, m.commitMsg)
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
			if i == m.selected {
				sidebar.WriteString(selectedStyle.Render(session.name) + "\n")
			} else {
				sidebar.WriteString(session.name + "\n")
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

	main := lipgloss.JoinHorizontal(
		lipgloss.Top,
		dynamicSidebarStyle.Render(sidebar.String()),
		dynamicContentStyle.Render(content.String()),
	)

	footer := footerStyle.Width(80).Render("↑↓ navigate • enter attach • a add • k kill • r refresh • ? about • q quit")

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
			fmt.Println("Available themes: slate, pink, forest")
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
