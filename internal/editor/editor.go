// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package editor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const editorVersion = "0.3.0"

func Run(filePath string) {
	m := newModel(filePath)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "editor: %v\n", err)
		os.Exit(1)
	}
}

type editorMode int

const (
	modeNormal editorMode = iota
	modeInsert
	modeCommand
	modeFilePicker
	modeSaveAs
)

type model struct {
	lines      []string
	cx, cy     int
	scrollX    int
	scrollY    int
	width      int
	height     int
	filePath   string
	dirty      bool
	mode       editorMode
	cmdBuf     string
	statusMsg  string
	showAC     bool
	acItems    []string
	acSel      int
	acWord     string
	fpEntries  []string
	fpSel      int
	fpDir      string
	saveAsName string
	showHelp   bool
}

func newModel(filePath string) model {
	m := model{
		lines:  []string{""},
		width:  80,
		height: 24,
		mode:   modeNormal,
	}
	if filePath != "" {
		m.filePath = filePath
		data, err := os.ReadFile(filePath)
		if err == nil {
			content := strings.ReplaceAll(string(data), "\r\n", "\n")
			m.lines = strings.Split(content, "\n")
			if len(m.lines) == 0 {
				m.lines = []string{""}
			}
		} else {
			m.statusMsg = "new file: " + filePath
		}
	} else {
		m.statusMsg = "press Ctrl+H for help"
	}
	return m
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if m.mode == modeFilePicker {
		return m.handleFilePicker(key)
	}
	if m.mode == modeSaveAs {
		return m.handleSaveAs(key)
	}
	if m.mode == modeCommand {
		return m.handleCommand(key)
	}
	if m.mode == modeInsert {
		return m.handleInsert(key)
	}
	return m.handleNormal(key)
}

func (m model) handleNormal(key string) (tea.Model, tea.Cmd) {
	m.showAC = false
	m.showHelp = false
	switch key {
	case "ctrl+c", "ctrl+q":
		return m, tea.Quit
	case "ctrl+h":
		m.showHelp = !m.showHelp
	case "i":
		m.mode = modeInsert
		m.statusMsg = "INSERT"
	case "a":
		m.mode = modeInsert
		m.cx = len(m.currentLine())
		m.statusMsg = "INSERT"
	case "o":
		m.insertLineBelow()
		m.cy++
		m.cx = 0
		m.mode = modeInsert
		m.statusMsg = "INSERT"
	case "O":
		m.insertLineAbove()
		m.cx = 0
		m.mode = modeInsert
		m.statusMsg = "INSERT"
	case ":":
		m.mode = modeCommand
		m.cmdBuf = ""
	case "ctrl+s":
		m.save()
	case "ctrl+o":
		m.openFilePicker()
	case "h", "left":
		if m.cx > 0 {
			m.cx--
		}
	case "l", "right":
		if m.cx < len(m.currentLine()) {
			m.cx++
		}
	case "k", "up":
		if m.cy > 0 {
			m.cy--
			m.clampX()
		}
	case "j", "down":
		if m.cy < len(m.lines)-1 {
			m.cy++
			m.clampX()
		}
	case "0", "home":
		m.cx = 0
	case "$", "end":
		m.cx = len(m.currentLine())
	case "g":
		m.cy = 0
		m.cx = 0
	case "G":
		m.cy = len(m.lines) - 1
		m.cx = len(m.currentLine())
	case "ctrl+d":
		m.cy += m.contentHeight() / 2
		if m.cy >= len(m.lines) {
			m.cy = len(m.lines) - 1
		}
		m.clampX()
	case "ctrl+u":
		m.cy -= m.contentHeight() / 2
		if m.cy < 0 {
			m.cy = 0
		}
		m.clampX()
	case "d":
		m.deleteLine()
	case "x":
		m.deleteChar()
	case "w":
		m.cx = m.nextWordStart()
	case "b":
		m.cx = m.prevWordStart()
	}
	m.scroll()
	return m, nil
}

func (m model) handleInsert(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "escape":
		m.mode = modeNormal
		m.statusMsg = ""
		m.showAC = false
		if m.cx > 0 {
			m.cx--
		}
		return m, nil
	case "ctrl+c":
		return m, tea.Quit
	case "ctrl+s":
		m.save()
		return m, nil
	case "tab":
		if m.showAC && len(m.acItems) > 0 {
			m.applyAutocomplete()
			return m, nil
		}
		m.insertText("  ")
		return m, nil
	case "up":
		if m.showAC && len(m.acItems) > 0 {
			m.acSel--
			if m.acSel < 0 {
				m.acSel = len(m.acItems) - 1
			}
			return m, nil
		}
		if m.cy > 0 {
			m.cy--
			m.clampX()
		}
		return m, nil
	case "down":
		if m.showAC && len(m.acItems) > 0 {
			m.acSel = (m.acSel + 1) % len(m.acItems)
			return m, nil
		}
		if m.cy < len(m.lines)-1 {
			m.cy++
			m.clampX()
		}
		return m, nil
	case "left":
		if m.cx > 0 {
			m.cx--
		}
		m.showAC = false
		return m, nil
	case "right":
		if m.cx < len(m.currentLine()) {
			m.cx++
		}
		return m, nil
	case "enter":
		m.insertNewline()
		m.showAC = false
		m.dirty = true
		return m, nil
	case "backspace", "ctrl+h":
		m.backspace()
		m.updateAutocomplete()
		m.dirty = true
		return m, nil
	default:
		if len(key) == 1 && key[0] >= 32 {
			m.insertRune(rune(key[0]))
			m.updateAutocomplete()
			m.dirty = true
		}
	}
	m.scroll()
	return m, nil
}

func (m model) handleCommand(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "escape":
		m.mode = modeNormal
		m.cmdBuf = ""
		m.statusMsg = ""
	case "enter":
		m.execCommand(m.cmdBuf)
		m.mode = modeNormal
		m.cmdBuf = ""
	case "backspace":
		if len(m.cmdBuf) > 0 {
			m.cmdBuf = m.cmdBuf[:len(m.cmdBuf)-1]
		} else {
			m.mode = modeNormal
		}
	default:
		if len(key) == 1 {
			m.cmdBuf += key
		}
	}
	return m, nil
}

func (m *model) execCommand(cmd string) {
	cmd = strings.TrimSpace(cmd)
	switch {
	case cmd == "q" || cmd == "quit":
		os.Exit(0)
	case cmd == "q!" || cmd == "quit!":
		os.Exit(0)
	case cmd == "w" || cmd == "write":
		m.save()
	case cmd == "wq":
		m.save()
		os.Exit(0)
	case strings.HasPrefix(cmd, "w "):
		name := strings.TrimSpace(cmd[2:])
		m.filePath = name
		m.save()
	case strings.HasPrefix(cmd, "e "):
		name := strings.TrimSpace(cmd[2:])
		m.loadFile(name)
	case cmd == "new":
		m.lines = []string{""}
		m.cx, m.cy = 0, 0
		m.filePath = ""
		m.dirty = false
		m.statusMsg = "new file"
	case cmd == "help":
		m.showHelp = true
	default:
		m.statusMsg = "unknown command: " + cmd
	}
}

func (m model) handleFilePicker(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "escape", "ctrl+c":
		m.mode = modeNormal
	case "up", "k":
		if m.fpSel > 0 {
			m.fpSel--
		}
	case "down", "j":
		if m.fpSel < len(m.fpEntries)-1 {
			m.fpSel++
		}
	case "enter":
		if m.fpSel < len(m.fpEntries) {
			entry := m.fpEntries[m.fpSel]
			fullPath := filepath.Join(m.fpDir, entry)
			info, err := os.Stat(fullPath)
			if err == nil && info.IsDir() {
				m.fpDir = fullPath
				m.refreshFilePicker()
			} else {
				m.loadFile(fullPath)
				m.mode = modeNormal
			}
		}
	case "backspace":
		m.fpDir = filepath.Dir(m.fpDir)
		m.refreshFilePicker()
	}
	return m, nil
}

func (m model) handleSaveAs(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "escape":
		m.mode = modeNormal
	case "enter":
		if m.saveAsName != "" {
			m.filePath = m.saveAsName
			m.save()
		}
		m.mode = modeNormal
	case "backspace":
		if len(m.saveAsName) > 0 {
			m.saveAsName = m.saveAsName[:len(m.saveAsName)-1]
		}
	default:
		if len(key) == 1 {
			m.saveAsName += key
		}
	}
	return m, nil
}

func (m *model) openFilePicker() {
	cwd, _ := os.Getwd()
	m.fpDir = cwd
	m.fpSel = 0
	m.refreshFilePicker()
	m.mode = modeFilePicker
}

func (m *model) refreshFilePicker() {
	entries, err := os.ReadDir(m.fpDir)
	if err != nil {
		return
	}
	m.fpEntries = []string{".."}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			name += "/"
		}
		m.fpEntries = append(m.fpEntries, name)
	}
}

func (m *model) loadFile(path string) {
	abs, err := filepath.Abs(path)
	if err != nil {
		m.statusMsg = "error: " + err.Error()
		return
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		m.statusMsg = "error: " + err.Error()
		return
	}
	content := strings.ReplaceAll(string(data), "\r\n", "\n")
	m.lines = strings.Split(content, "\n")
	if len(m.lines) == 0 {
		m.lines = []string{""}
	}
	m.filePath = abs
	m.cx, m.cy = 0, 0
	m.scrollX, m.scrollY = 0, 0
	m.dirty = false
	m.statusMsg = "opened " + filepath.Base(abs)
}

func (m *model) save() {
	if m.filePath == "" {
		m.mode = modeSaveAs
		m.saveAsName = ""
		m.statusMsg = "save as:"
		return
	}
	content := strings.Join(m.lines, "\n")
	if err := os.WriteFile(m.filePath, []byte(content), 0644); err != nil {
		m.statusMsg = "error saving: " + err.Error()
		return
	}
	m.dirty = false
	m.statusMsg = "saved " + filepath.Base(m.filePath)
}

func (m *model) currentLine() string {
	if m.cy >= len(m.lines) {
		return ""
	}
	return m.lines[m.cy]
}

func (m *model) insertRune(r rune) {
	line := m.currentLine()
	if m.cx > len(line) {
		m.cx = len(line)
	}
	m.lines[m.cy] = line[:m.cx] + string(r) + line[m.cx:]
	m.cx++
}

func (m *model) insertText(s string) {
	for _, r := range s {
		m.insertRune(r)
	}
}

func (m *model) backspace() {
	if m.cx > 0 {
		line := m.currentLine()
		m.lines[m.cy] = line[:m.cx-1] + line[m.cx:]
		m.cx--
	} else if m.cy > 0 {
		prevLine := m.lines[m.cy-1]
		currLine := m.currentLine()
		m.cx = len(prevLine)
		m.lines[m.cy-1] = prevLine + currLine
		m.lines = append(m.lines[:m.cy], m.lines[m.cy+1:]...)
		m.cy--
	}
}

func (m *model) insertNewline() {
	line := m.currentLine()
	indent := leadingIndent(line)
	before := line[:m.cx]
	after := line[m.cx:]
	m.lines[m.cy] = before
	newLine := indent + after
	m.lines = append(m.lines[:m.cy+1], append([]string{newLine}, m.lines[m.cy+1:]...)...)
	m.cy++
	m.cx = len(indent)
}

func (m *model) insertLineBelow() {
	indent := leadingIndent(m.currentLine())
	m.lines = append(m.lines[:m.cy+1], append([]string{indent}, m.lines[m.cy+1:]...)...)
}

func (m *model) insertLineAbove() {
	indent := leadingIndent(m.currentLine())
	m.lines = append(m.lines[:m.cy], append([]string{indent}, m.lines[m.cy:]...)...)
}

func (m *model) deleteLine() {
	if len(m.lines) == 1 {
		m.lines[0] = ""
		m.cx = 0
		return
	}
	m.lines = append(m.lines[:m.cy], m.lines[m.cy+1:]...)
	if m.cy >= len(m.lines) {
		m.cy = len(m.lines) - 1
	}
	m.clampX()
}

func (m *model) deleteChar() {
	line := m.currentLine()
	if m.cx < len(line) {
		m.lines[m.cy] = line[:m.cx] + line[m.cx+1:]
	}
}

func (m *model) clampX() {
	lineLen := len(m.currentLine())
	if m.cx > lineLen {
		m.cx = lineLen
	}
}

func (m *model) scroll() {
	ch := m.contentHeight()
	if m.cy < m.scrollY {
		m.scrollY = m.cy
	}
	if m.cy >= m.scrollY+ch {
		m.scrollY = m.cy - ch + 1
	}
	cw := m.contentWidth()
	if m.cx < m.scrollX {
		m.scrollX = m.cx
	}
	if m.cx >= m.scrollX+cw {
		m.scrollX = m.cx - cw + 1
	}
}

func (m *model) contentHeight() int {
	h := m.height - 3
	if h < 1 {
		h = 1
	}
	return h
}

func (m *model) contentWidth() int {
	w := m.width - 6
	if w < 10 {
		w = 10
	}
	return w
}

func (m *model) nextWordStart() int {
	line := m.currentLine()
	i := m.cx
	for i < len(line) && !unicode.IsSpace(rune(line[i])) {
		i++
	}
	for i < len(line) && unicode.IsSpace(rune(line[i])) {
		i++
	}
	return i
}

func (m *model) prevWordStart() int {
	line := m.currentLine()
	i := m.cx
	if i > 0 {
		i--
	}
	for i > 0 && unicode.IsSpace(rune(line[i])) {
		i--
	}
	for i > 0 && !unicode.IsSpace(rune(line[i-1])) {
		i--
	}
	return i
}

var ntlKeywords = []string{
	"val", "var", "fn", "return", "if", "else", "elif", "while", "for", "each",
	"in", "of", "break", "continue", "match", "case", "default", "try", "catch",
	"finally", "throw", "raise", "class", "extends", "new", "this", "super",
	"static", "abstract", "override", "import", "export", "from", "as",
	"use", "null", "true", "false", "and", "or", "not", "range", "sleep",
	"spawn", "channel", "loop", "repeat", "guard", "defer", "module",
	"io", "fs", "http", "crypto", "db", "env", "utils", "validate",
	"events", "cache", "logger", "queue", "ws", "mail", "ai", "test",
}

func (m *model) updateAutocomplete() {
	line := m.currentLine()
	if m.cx == 0 {
		m.showAC = false
		return
	}
	word := currentWord(line, m.cx)
	if len(word) < 2 {
		m.showAC = false
		return
	}
	m.acWord = word
	m.acItems = nil
	for _, kw := range ntlKeywords {
		if strings.HasPrefix(kw, word) && kw != word {
			m.acItems = append(m.acItems, kw)
		}
	}
	m.showAC = len(m.acItems) > 0
	m.acSel = 0
}

func (m *model) applyAutocomplete() {
	if m.acSel >= len(m.acItems) {
		return
	}
	completion := m.acItems[m.acSel]
	line := m.currentLine()
	wordStart := m.cx - len(m.acWord)
	if wordStart < 0 {
		wordStart = 0
	}
	m.lines[m.cy] = line[:wordStart] + completion + line[m.cx:]
	m.cx = wordStart + len(completion)
	m.showAC = false
}

func currentWord(line string, cx int) string {
	if cx == 0 {
		return ""
	}
	i := cx - 1
	for i >= 0 && (unicode.IsLetter(rune(line[i])) || line[i] == '_') {
		i--
	}
	return line[i+1 : cx]
}

func leadingIndent(s string) string {
	i := 0
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	return s[:i]
}

func (m model) View() string {
	if m.showHelp {
		return m.renderHelp()
	}
	if m.mode == modeFilePicker {
		return m.renderFilePicker()
	}

	var sb strings.Builder
	sb.WriteString(m.renderTitleBar())
	sb.WriteString("\n")

	ch := m.contentHeight()
	cw := m.contentWidth()
	lineNumWidth := 5

	for row := 0; row < ch; row++ {
		lineIdx := row + m.scrollY
		lineNumStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Width(lineNumWidth).
			Align(lipgloss.Right)

		if lineIdx >= len(m.lines) {
			sb.WriteString(lineNumStyle.Render("~"))
			sb.WriteString(" ")
			sb.WriteString(strings.Repeat(" ", cw))
		} else {
			lineNum := fmt.Sprintf("%d", lineIdx+1)
			if lineIdx == m.cy {
				lineNumStyle = lineNumStyle.Foreground(lipgloss.Color("220"))
			}
			sb.WriteString(lineNumStyle.Render(lineNum))
			sb.WriteString(" ")

			line := m.lines[lineIdx]
			displayed := highlightLine(line, lineIdx == m.cy)

			if m.scrollX > 0 && m.scrollX < len(line) {
				if m.scrollX < len(displayed) {
					displayed = displayed[m.scrollX:]
				} else {
					displayed = ""
				}
			}

			if lipgloss.Width(displayed) > cw {
				displayed = displayed[:cw]
			}

			if lineIdx == m.cy && m.mode == modeInsert {
				cursorPos := m.cx - m.scrollX
				rawLine := line
				if m.scrollX < len(rawLine) {
					rawLine = rawLine[m.scrollX:]
				}
				if cursorPos >= 0 {
					displayed = renderWithCursor(rawLine, cursorPos, cw)
				}
			} else if lineIdx == m.cy {
				rawLine := line
				if m.scrollX < len(rawLine) {
					rawLine = rawLine[m.scrollX:]
				}
				cursorPos := m.cx - m.scrollX
				if cursorPos < 0 {
					cursorPos = 0
				}
				displayed = renderWithCursorNormal(rawLine, cursorPos, cw)
			}

			sb.WriteString(displayed)
		}
		sb.WriteString("\n")
	}

	sb.WriteString(m.renderStatusBar())

	if m.mode == modeCommand {
		sb.WriteString("\n")
		sb.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")).
			Background(lipgloss.Color("236")).
			Render(":" + m.cmdBuf + "█"))
	} else if m.mode == modeSaveAs {
		sb.WriteString("\n")
		sb.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")).
			Background(lipgloss.Color("236")).
			Render("save as: " + m.saveAsName + "█"))
	}

	if m.showAC && len(m.acItems) > 0 {
		sb.WriteString(m.renderAutocomplete())
	}

	return sb.String()
}

func (m model) renderTitleBar() string {
	name := m.filePath
	if name == "" {
		name = "[untitled]"
	} else {
		name = filepath.Base(name)
	}
	if m.dirty {
		name += " ●"
	}
	modeStr := ""
	switch m.mode {
	case modeInsert:
		modeStr = " INSERT "
	case modeCommand:
		modeStr = " COMMAND "
	case modeFilePicker:
		modeStr = " FILES "
	}

	left := lipgloss.NewStyle().
		Background(lipgloss.Color("25")).
		Foreground(lipgloss.Color("255")).
		Bold(true).
		Padding(0, 1).
		Render("lunex edit v" + editorVersion)

	center := lipgloss.NewStyle().
		Background(lipgloss.Color("237")).
		Foreground(lipgloss.Color("255")).
		Padding(0, 1).
		Render(name)

	modeLabel := ""
	if modeStr != "" {
		modeLabel = lipgloss.NewStyle().
			Background(lipgloss.Color("34")).
			Foreground(lipgloss.Color("255")).
			Bold(true).
			Render(modeStr)
	}

	ntlLabel := lipgloss.NewStyle().
		Background(lipgloss.Color("25")).
		Foreground(lipgloss.Color("255")).
		Padding(0, 1).
		Render("Lunex")

	used := lipgloss.Width(left) + lipgloss.Width(center) + lipgloss.Width(modeLabel) + lipgloss.Width(ntlLabel)
	gap := m.width - used
	if gap < 0 {
		gap = 0
	}
	filler := lipgloss.NewStyle().
		Background(lipgloss.Color("237")).
		Render(strings.Repeat(" ", gap))

	return left + center + filler + modeLabel + ntlLabel
}

func (m model) renderStatusBar() string {
	pos := fmt.Sprintf("Ln %d, Col %d", m.cy+1, m.cx+1)
	total := fmt.Sprintf("%d lines", len(m.lines))
	msg := m.statusMsg

	right := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Render(pos + "  " + total)

	left := lipgloss.NewStyle().
		Foreground(lipgloss.Color("250")).
		Render(msg)

	used := lipgloss.Width(left) + lipgloss.Width(right)
	gap := m.width - used - 2
	if gap < 0 {
		gap = 0
	}
	filler := strings.Repeat(" ", gap)

	return lipgloss.NewStyle().
		Background(lipgloss.Color("235")).
		Foreground(lipgloss.Color("250")).
		Width(m.width).
		Render(left + filler + right)
}

func (m model) renderAutocomplete() string {
	maxShow := 6
	start := 0
	if m.acSel >= maxShow {
		start = m.acSel - maxShow + 1
	}
	end := start + maxShow
	if end > len(m.acItems) {
		end = len(m.acItems)
	}

	var sb strings.Builder
	sb.WriteString("\n")
	for i := start; i < end; i++ {
		item := m.acItems[i]
		style := lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("252")).
			Padding(0, 1)
		if i == m.acSel {
			style = style.
				Background(lipgloss.Color("25")).
				Foreground(lipgloss.Color("255")).
				Bold(true)
		}
		prefix := lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Render(m.acWord)
		rest := item[len(m.acWord):]
		sb.WriteString(style.Render("  " + prefix + rest))
		sb.WriteString("\n")
	}
	return sb.String()
}

func (m model) renderFilePicker() string {
	var sb strings.Builder

	title := lipgloss.NewStyle().
		Background(lipgloss.Color("25")).
		Foreground(lipgloss.Color("255")).
		Bold(true).
		Padding(0, 1).
		Width(m.width).
		Render("lunex edit  File Browser  " + m.fpDir)
	sb.WriteString(title)
	sb.WriteString("\n\n")

	maxShow := m.height - 6
	start := 0
	if m.fpSel >= maxShow {
		start = m.fpSel - maxShow + 1
	}
	end := start + maxShow
	if end > len(m.fpEntries) {
		end = len(m.fpEntries)
	}

	for i := start; i < end; i++ {
		entry := m.fpEntries[i]
		style := lipgloss.NewStyle().
			Foreground(lipgloss.Color("250")).
			Padding(0, 2)
		if i == m.fpSel {
			style = style.Background(lipgloss.Color("25")).Foreground(lipgloss.Color("255"))
		}
		if strings.HasSuffix(entry, "/") {
			style = style.Foreground(lipgloss.Color("75"))
			if i == m.fpSel {
				style = style.Background(lipgloss.Color("25"))
			}
		} else if strings.HasSuffix(entry, ".lx") || strings.HasSuffix(entry, ".nc") || strings.HasSuffix(entry, ".nax") {
			style = style.Foreground(lipgloss.Color("118"))
			if i == m.fpSel {
				style = style.Background(lipgloss.Color("25"))
			}
		}
		sb.WriteString(style.Render(entry))
		sb.WriteString("\n")
	}

	help := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Padding(0, 2).
		Render("\nEnter: open  Backspace: up dir  Esc: cancel")
	sb.WriteString(help)
	return sb.String()
}

func (m model) renderHelp() string {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Padding(1, 3)
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("75")).
		Render("lunex edit v" + editorVersion + "  Lunex Editor\n")
	content := title + `
Navigation (Normal mode):
  h/l/j/k  or  arrow keys     Move cursor
  0 / $                        Start / end of line
  g / G                        First / last line
  Ctrl+D / Ctrl+U              Half page down / up
  w / b                        Next / prev word

Editing:
  i                            Insert before cursor
  a                            Insert after cursor
  o / O                        New line below / above
  d                            Delete line
  x                            Delete character

Insert mode:
  Tab                          Accept autocomplete / indent 2 spaces
  Up/Down                      Navigate autocomplete list
  Esc                          Return to Normal mode

File operations:
  Ctrl+S                       Save
  Ctrl+O                       Open file browser
  :w filename                  Save as filename
  :e filename                  Open file
  :q                           Quit  :wq                          Save and quit

Commands (press : then type):
  w          save
  q          quit
  wq         save and quit
  new        create new file
  help       show this help

Ctrl+H                         Toggle this help panel
Ctrl+Q / Ctrl+C                Quit (from Normal mode)
`
	return style.Render(content)
}

var ntlColorKeywords = map[string]string{
	"val": "75", "var": "75", "fn": "213", "return": "213",
	"if": "213", "else": "213", "elif": "213", "while": "213",
	"for": "213", "each": "213", "in": "213", "of": "213",
	"break": "213", "continue": "213", "match": "213", "case": "213",
	"default": "213", "try": "213", "catch": "213", "finally": "213",
	"throw": "213", "raise": "213", "class": "220", "extends": "220",
	"new": "220", "this": "220", "super": "220", "static": "220",
	"abstract": "220", "override": "220", "import": "147", "export": "147",
	"from": "147", "as": "147", "use": "147", "null": "172",
	"true": "76", "false": "196", "and": "213", "or": "213",
	"not": "213", "spawn": "213", "loop": "213", "repeat": "213",
	"guard": "213", "defer": "213", "range": "33", "sleep": "33",
}

func highlightLine(line string, current bool) string {
	if len(line) == 0 {
		return ""
	}
	bg := lipgloss.NewStyle()
	if current {
		bg = bg.Background(lipgloss.Color("235"))
	}
	if strings.HasPrefix(strings.TrimSpace(line), "//") {
		return bg.Foreground(lipgloss.Color("240")).Italic(true).Render(line)
	}

	var sb strings.Builder
	i := 0
	for i < len(line) {
		if line[i] == '"' || line[i] == '\'' || line[i] == '`' {
			q := line[i]
			j := i + 1
			for j < len(line) && line[j] != q {
				if line[j] == '\\' {
					j++
				}
				j++
			}
			if j < len(line) {
				j++
			}
			sb.WriteString(bg.Foreground(lipgloss.Color("215")).Render(line[i:j]))
			i = j
			continue
		}
		if i+1 < len(line) && line[i] == '/' && line[i+1] == '/' {
			sb.WriteString(bg.Foreground(lipgloss.Color("240")).Italic(true).Render(line[i:]))
			break
		}
		if line[i] >= '0' && line[i] <= '9' {
			j := i
			for j < len(line) && ((line[j] >= '0' && line[j] <= '9') || line[j] == '.' || line[j] == 'x' || line[j] == 'X') {
				j++
			}
			sb.WriteString(bg.Foreground(lipgloss.Color("172")).Render(line[i:j]))
			i = j
			continue
		}
		if isIdentStart(line[i]) {
			j := i
			for j < len(line) && isIdentCharByte(line[j]) {
				j++
			}
			word := line[i:j]
			if color, ok := ntlColorKeywords[word]; ok {
				sb.WriteString(bg.Foreground(lipgloss.Color(color)).Bold(true).Render(word))
			} else {
				sb.WriteString(bg.Foreground(lipgloss.Color("255")).Render(word))
			}
			i = j
			continue
		}
		ch := string(line[i])
		if isOperatorByte(line[i]) {
			sb.WriteString(bg.Foreground(lipgloss.Color("197")).Render(ch))
		} else if isPunctByte(line[i]) {
			sb.WriteString(bg.Foreground(lipgloss.Color("245")).Render(ch))
		} else {
			sb.WriteString(bg.Render(ch))
		}
		i++
	}
	return sb.String()
}

func renderWithCursor(line string, cx, maxW int) string {
	var sb strings.Builder
	for i := 0; i < maxW; i++ {
		ch := " "
		if i < len(line) {
			ch = string(line[i])
		}
		if i == cx {
			sb.WriteString(lipgloss.NewStyle().
				Background(lipgloss.Color("255")).
				Foreground(lipgloss.Color("0")).
				Render(ch))
		} else {
			sb.WriteString(lipgloss.NewStyle().
				Background(lipgloss.Color("235")).
				Render(ch))
		}
	}
	return sb.String()
}

func renderWithCursorNormal(line string, cx, maxW int) string {
	var sb strings.Builder
	for i := 0; i < maxW; i++ {
		ch := " "
		if i < len(line) {
			ch = string(line[i])
		}
		if i == cx {
			sb.WriteString(lipgloss.NewStyle().
				Background(lipgloss.Color("33")).
				Foreground(lipgloss.Color("255")).
				Render(ch))
		} else {
			sb.WriteString(lipgloss.NewStyle().Render(ch))
		}
	}
	return sb.String()
}

func isIdentStart(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}

func isIdentCharByte(c byte) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9')
}

func isOperatorByte(c byte) bool {
	return strings.ContainsRune("+-*/=<>!&|^~%?:", rune(c))
}

func isPunctByte(c byte) bool {
	return strings.ContainsRune("{}()[].,;", rune(c))
}
