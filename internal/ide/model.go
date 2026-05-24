// NT-IDE — Lunex Integrated Development Environment
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package ide

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// editorMode is the current editing mode
type editorMode int

const (
	modeInsert  editorMode = iota // Normal text editing (default)
	modeNormal                    // Vim-like navigation mode
	modeCommand                   // Command bar (:w, :q, etc.)
	modeSaveAs                    // Save-as prompt
	modeQuitSave                  // Quit — save or not?
	modeSearch                    // Search/find
)

// panelFocus determines which panel is focused
type panelFocus int

const (
	focusEditor panelFocus = iota
	focusOutput
	focusErrors
)

// diagDebounceDelay is how long to wait after typing before checking errors
const diagDebounceDelay = 400 * time.Millisecond

// model is the main bubbletea model for NT-IDE
type model struct {
	// Editor state
	lines   []string
	cx, cy  int // cursor column (rune), cursor line
	scrollX int
	scrollY int

	// Window
	width  int
	height int

	// File
	filePath string
	dirty    bool
	isNew    bool

	// Mode
	mode  editorMode
	focus panelFocus

	// Command mode buffer
	cmdBuf string

	// Save-as buffer
	saveAsName string

	// Search
	searchQuery  string
	searchResult []searchMatch

	// Autocomplete
	ac AutocompleteState

	// Diagnostics
	diags         DiagnosticsResult
	diagPending   bool  // waiting for debounce tick
	diagCheckTime time.Time

	// Output panel
	outputLines []string
	outputSel   int
	outputScroll int
	showOutput  bool
	outputErr   bool // did last run have errors?
	running     bool
	runDuration time.Duration

	// Error panel (shows at bottom when errors exist)
	errPanelSel int

	// Status message
	statusMsg     string
	statusTimeout time.Time

	// Theme
	theme Theme

	// Quit-save prompt
	quitSaveMsg string

	// Layout
	outputHeight  int // number of lines for the output panel
	errPanelHeight int
}

type searchMatch struct {
	line, col int
}

// --- Init ---

func newModel(filePath string) model {
	m := model{
		lines:         []string{""},
		width:         80,
		height:        24,
		mode:          modeInsert,
		theme:         DefaultTheme,
		ac:            newAutocompleteState(),
		outputLines:   []string{},
		outputHeight:  8,
		errPanelHeight: 6,
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
			// Check errors on open
			m.requestDiagCheck()
		} else {
			// New file — insert template
			m.isNew = true
			m.insertTemplate()
			m.setStatus("New file: " + filepath.Base(filePath))
		}
	} else {
		m.isNew = true
		m.insertTemplate()
		m.setStatus("NT-IDE v" + IDEVersion + "  Ctrl+H: help")
	}
	return m
}

func (m *model) insertTemplate() {
	template := DefaultTemplate()
	m.lines = strings.Split(strings.ReplaceAll(template, "\r\n", "\n"), "\n")
	// Position cursor after the main fn opening brace
	m.cy = 3
	m.cx = 2
}

// --- tea.Model interface ---

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

	case tea.MouseMsg:
		return m.handleMouse(msg)

	case runMsg:
		m.running = false
		m.outputLines = msg.result.Output
		m.outputErr = msg.result.ExitCode != 0
		m.runDuration = msg.result.Duration
		m.showOutput = true
		if msg.result.ExitCode == 0 {
			m.setStatus(fmt.Sprintf("✓ Ran in %dms", msg.result.Duration.Milliseconds()))
		} else {
			m.setStatus(fmt.Sprintf("✗ Exit %d (%dms)", msg.result.ExitCode, msg.result.Duration.Milliseconds()))
		}
		return m, nil

	case diagMsg:
		m.diags = msg.result
		m.diagPending = false
		return m, nil

	case tickMsg:
		if m.statusMsg != "" && time.Now().After(m.statusTimeout) {
			m.statusMsg = ""
		}
		if m.diagPending && time.Since(m.diagCheckTime) >= diagDebounceDelay {
			return m, m.runDiagCheck()
		}
		return m, nil
	}

	return m, nil
}

// --- Key Handling ---

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch m.mode {
	case modeQuitSave:
		return m.handleQuitSave(key)
	case modeSaveAs:
		return m.handleSaveAs(key)
	case modeCommand:
		return m.handleCommand(key)
	case modeSearch:
		return m.handleSearchMode(key)
	case modeNormal:
		return m.handleNormal(key)
	default: // modeInsert
		return m.handleInsert(key)
	}
}

// handleInsert handles insert mode (main editing mode)
func (m model) handleInsert(key string) (tea.Model, tea.Cmd) {
	switch key {
	// --- Mode switches ---
	case "ctrl+c":
		if m.dirty {
			m.mode = modeQuitSave
			m.quitSaveMsg = "Save before exit? (y/n/Esc)"
			return m, nil
		}
		return m, tea.Quit

	case "escape":
		if m.ac.Active {
			m.ac.Reset()
			return m, nil
		}
		m.mode = modeNormal
		m.setStatus("NORMAL — press i to insert, Ctrl+C to quit")
		if m.cx > 0 {
			m.cx--
		}
		return m, nil

	// --- File operations ---
	case "ctrl+s":
		return m.doSave()

	case "ctrl+r":
		return m.doRun()

	case "ctrl+f":
		return m.doFormat()

	case "ctrl+o":
		m.mode = modeSearch
		m.searchQuery = ""
		m.setStatus("Open file: (type path)")
		return m, nil

	// --- Autocomplete navigation ---
	case "up":
		if m.ac.Active && len(m.ac.Items) > 0 {
			m.ac.MoveUp()
			return m, nil
		}
		if m.cy > 0 {
			m.cy--
			m.clampX()
		}
		return m, nil

	case "down":
		if m.ac.Active && len(m.ac.Items) > 0 {
			m.ac.MoveDown()
			return m, nil
		}
		if m.cy < len(m.lines)-1 {
			m.cy++
			m.clampX()
		}
		return m, nil

	case "tab":
		if m.ac.Active && len(m.ac.Items) > 0 {
			return m.applyCompletion()
		}
		return m.insertText("  ")

	case "enter":
		if m.ac.Active && len(m.ac.Items) > 0 {
			// Accept completion on Enter
			nm, cmd := m.applyCompletion()
			return nm, cmd
		}
		m.ac.Reset()
		return m.insertNewline()

	case "backspace":
		m.ac.Reset()
		return m.doBackspace()

	// --- Cursor movement ---
	case "left":
		m.ac.Reset()
		if m.cx > 0 {
			m.cx--
		} else if m.cy > 0 {
			m.cy--
			m.cx = len([]rune(m.currentLine()))
		}
		return m, nil

	case "right":
		m.ac.Reset()
		lineLen := len([]rune(m.currentLine()))
		if m.cx < lineLen {
			m.cx++
		} else if m.cy < len(m.lines)-1 {
			m.cy++
			m.cx = 0
		}
		return m, nil

	case "home", "ctrl+a":
		m.ac.Reset()
		// Smart home: go to first non-whitespace, then actual start
		line := m.currentLine()
		firstNonWS := len(LeadingIndent(line))
		if m.cx == firstNonWS {
			m.cx = 0
		} else {
			m.cx = firstNonWS
		}
		return m, nil

	case "end":
		m.ac.Reset()
		m.cx = len([]rune(m.currentLine()))
		return m, nil

	case "ctrl+up", "alt+up":
		// Move line up
		return m.moveLineUp()

	case "ctrl+down", "alt+down":
		// Move line down
		return m.moveLineDown()

	case "pgup":
		m.ac.Reset()
		m.cy -= m.editorHeight() - 2
		if m.cy < 0 {
			m.cy = 0
		}
		m.clampX()
		return m, nil

	case "pgdown":
		m.ac.Reset()
		m.cy += m.editorHeight() - 2
		if m.cy >= len(m.lines) {
			m.cy = len(m.lines) - 1
		}
		m.clampX()
		return m, nil

	// --- Edit shortcuts ---
	case "ctrl+d":
		// Duplicate current line
		return m.duplicateLine()

	case "ctrl+k":
		// Delete current line
		return m.deleteLine()

	case "ctrl+z":
		// TODO: undo (future)
		m.setStatus("Undo not yet implemented")
		return m, nil

	case "ctrl+/":
		// Toggle comment
		return m.toggleComment()

	case "ctrl+h":
		return m.showHelp()

	case "ctrl+p":
		// Toggle output panel
		m.showOutput = !m.showOutput
		return m, nil

	case "ctrl+e":
		// Focus output
		m.focus = focusOutput
		return m, nil

	default:
		// Regular character input
		if len(key) == 1 {
			r := rune(key[0])
			if r >= 32 || r == '\t' {
				return m.insertChar(r)
			}
		} else {
			// Handle multi-byte unicode input (bubbletea sends these as rune sequences)
			runes := []rune(key)
			if len(runes) == 1 && runes[0] >= 32 {
				return m.insertChar(runes[0])
			}
		}
		return m, nil
	}
}

// handleNormal handles Vim-like normal mode
func (m model) handleNormal(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "ctrl+c", "q":
		if m.dirty {
			m.mode = modeQuitSave
			m.quitSaveMsg = "Save before exit? (y/n/Esc)"
			return m, nil
		}
		return m, tea.Quit

	case "i":
		m.mode = modeInsert
		m.setStatus("")
		return m, nil

	case "a":
		m.mode = modeInsert
		if m.cx < len([]rune(m.currentLine())) {
			m.cx++
		}
		return m, nil

	case "A":
		m.mode = modeInsert
		m.cx = len([]rune(m.currentLine()))
		return m, nil

	case "o":
		m.mode = modeInsert
		m.insertLineBelow()
		m.cy++
		indent := LeadingIndent(m.lines[m.cy-1])
		if strings.HasSuffix(strings.TrimRight(m.lines[m.cy-1], " \t"), "{") {
			indent += "  "
		}
		m.lines[m.cy] = indent
		m.cx = len([]rune(indent))
		return m, nil

	case "O":
		m.mode = modeInsert
		m.insertLineAbove()
		m.cx = 0
		return m, nil

	case ":":
		m.mode = modeCommand
		m.cmdBuf = ""
		return m, nil

	case "ctrl+s":
		return m.doSave()

	case "ctrl+r", "r":
		return m.doRun()

	// Navigation
	case "h", "left":
		if m.cx > 0 {
			m.cx--
		}
	case "l", "right":
		if m.cx < len([]rune(m.currentLine())) {
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
		m.cx = len([]rune(m.currentLine()))
	case "^":
		m.cx = len([]rune(LeadingIndent(m.currentLine())))
	case "g":
		m.cy = 0
		m.cx = 0
	case "G":
		m.cy = len(m.lines) - 1
		m.cx = 0
	case "w":
		m.cx = m.nextWordStart()
	case "b":
		m.cx = m.prevWordStart()
	case "e":
		m.cx = m.wordEnd()
	case "ctrl+d":
		m.cy += m.editorHeight() / 2
		if m.cy >= len(m.lines) {
			m.cy = len(m.lines) - 1
		}
		m.clampX()
	case "ctrl+u":
		m.cy -= m.editorHeight() / 2
		if m.cy < 0 {
			m.cy = 0
		}
		m.clampX()
	case "d":
		nm, cmd := m.deleteLine()
		return nm, cmd
	case "x":
		m.deleteChar()
	case "ctrl+p":
		m.showOutput = !m.showOutput
	}

	m.scroll()
	return m, nil
}

// handleCommand handles the command mode (:w, :q, etc.)
func (m model) handleCommand(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "escape":
		m.mode = modeInsert
		m.cmdBuf = ""
	case "enter":
		cmd := m.execCommand(m.cmdBuf)
		m.cmdBuf = ""
		return m, cmd
	case "backspace":
		if len(m.cmdBuf) > 0 {
			m.cmdBuf = m.cmdBuf[:len(m.cmdBuf)-1]
		} else {
			m.mode = modeInsert
		}
	default:
		runes := []rune(key)
		if len(runes) == 1 && runes[0] >= 32 {
			m.cmdBuf += string(runes[0])
		}
	}
	return m, nil
}

func (m *model) execCommand(cmd string) tea.Cmd {
	cmd = strings.TrimSpace(cmd)
	m.mode = modeInsert
	switch {
	case cmd == "q" || cmd == "quit":
		if m.dirty {
			m.mode = modeQuitSave
			m.quitSaveMsg = "Save before exit? (y/n/Esc)"
			return nil
		}
		return tea.Quit
	case cmd == "q!" || cmd == "quit!":
		return tea.Quit
	case cmd == "w" || cmd == "write":
		_, cmd2 := m.doSave()
		return cmd2
	case cmd == "wq" || cmd == "x":
		m.doSaveNow()
		return tea.Quit
	case cmd == "fmt" || cmd == "format":
		_, c := m.doFormat()
		return c
	case cmd == "run":
		_, c := m.doRun()
		return c
	case cmd == "new":
		m.lines = []string{""}
		m.cx, m.cy = 0, 0
		m.filePath = ""
		m.dirty = false
		m.insertTemplate()
		m.isNew = true
		m.setStatus("New file")
	case strings.HasPrefix(cmd, "w "):
		name := strings.TrimSpace(cmd[2:])
		m.filePath = name
		m.doSaveNow()
	case strings.HasPrefix(cmd, "e "):
		name := strings.TrimSpace(cmd[2:])
		m.loadFile(name)
	case cmd == "errors" || cmd == "err":
		m.focus = focusErrors
		m.setStatus("Error panel focused — Esc to return")
	case cmd == "output" || cmd == "out":
		m.showOutput = !m.showOutput
	case cmd == "help":
		_, c := m.showHelp()
		return c
	default:
		m.setStatus("Unknown command: " + cmd)
	}
	return nil
}

// handleQuitSave handles the quit-save prompt
func (m model) handleQuitSave(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "y", "Y":
		m.doSaveNow()
		return m, tea.Quit
	case "n", "N":
		return m, tea.Quit
	case "escape", "ctrl+c":
		m.mode = modeInsert
		m.setStatus("")
	}
	return m, nil
}

// handleSaveAs handles the save-as prompt
func (m model) handleSaveAs(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "escape":
		m.mode = modeInsert
	case "enter":
		if m.saveAsName != "" {
			m.filePath = m.saveAsName
			m.doSaveNow()
		}
		m.mode = modeInsert
	case "backspace":
		if len(m.saveAsName) > 0 {
			m.saveAsName = m.saveAsName[:len(m.saveAsName)-1]
		}
	default:
		runes := []rune(key)
		if len(runes) == 1 && runes[0] >= 32 {
			m.saveAsName += string(runes[0])
		}
	}
	return m, nil
}

// handleSearchMode handles the search/open-file mode
func (m model) handleSearchMode(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "escape", "ctrl+c":
		m.mode = modeInsert
		m.searchQuery = ""
		m.setStatus("")
	case "enter":
		if m.searchQuery != "" {
			m.loadFile(m.searchQuery)
		}
		m.mode = modeInsert
		m.searchQuery = ""
	case "backspace":
		if len(m.searchQuery) > 0 {
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
		}
	default:
		runes := []rune(key)
		if len(runes) == 1 && runes[0] >= 32 {
			m.searchQuery += string(runes[0])
		}
	}
	return m, nil
}

// handleMouse handles mouse events
func (m model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch msg.Button {
	case tea.MouseButtonLeft:
		if msg.Action == tea.MouseActionPress {
			// Click in editor area
			editorTop := 1 // title bar
			editorBottom := m.editorBottom()
			if msg.Y >= editorTop && msg.Y < editorBottom {
				lineNumW := m.lineNumWidth() + 1
				newCY := m.scrollY + (msg.Y - editorTop)
				newCX := m.scrollX + (msg.X - lineNumW)
				if newCY < len(m.lines) {
					m.cy = newCY
					lineLen := len([]rune(m.lines[m.cy]))
					if newCX < 0 {
						newCX = 0
					}
					if newCX > lineLen {
						newCX = lineLen
					}
					m.cx = newCX
					m.mode = modeInsert
				}
			}
		}
	case tea.MouseButtonWheelUp:
		if m.scrollY > 0 {
			m.scrollY--
		}
	case tea.MouseButtonWheelDown:
		if m.scrollY < len(m.lines)-1 {
			m.scrollY++
		}
	}
	return m, nil
}

// --- Actions ---

func (m model) doSave() (tea.Model, tea.Cmd) {
	if m.filePath == "" {
		m.mode = modeSaveAs
		m.saveAsName = ""
		m.setStatus("Save as: ")
		return m, nil
	}
	m.doSaveNow()
	return m, nil
}

func (m *model) doSaveNow() {
	if m.filePath == "" {
		return
	}
	content := strings.Join(m.lines, "\n")
	if err := os.WriteFile(m.filePath, []byte(content), 0644); err != nil {
		m.setStatus("Error saving: " + err.Error())
		return
	}
	m.dirty = false
	m.isNew = false
	m.setStatus("Saved " + filepath.Base(m.filePath))
}

func (m model) doRun() (tea.Model, tea.Cmd) {
	if m.running {
		m.setStatus("Already running...")
		return m, nil
	}
	// Save first
	if m.filePath == "" {
		m.setStatus("Save file first (Ctrl+S)")
		return m, nil
	}
	m.doSaveNow()
	m.running = true
	m.showOutput = true
	m.outputLines = []string{"Running " + filepath.Base(m.filePath) + "..."}
	m.outputErr = false
	m.setStatus("Running...")

	content := strings.Join(m.lines, "\n")
	fp := m.filePath
	return m, func() tea.Msg {
		result := RunFile(fp, content)
		return runMsg{result: result}
	}
}

func (m model) doFormat() (tea.Model, tea.Cmd) {
	content := strings.Join(m.lines, "\n")
	formatted, err := FormatSource(content)
	if err != nil {
		m.setStatus("Format error: " + err.Error())
		return m, nil
	}
	m.lines = strings.Split(strings.ReplaceAll(formatted, "\r\n", "\n"), "\n")
	if len(m.lines) == 0 {
		m.lines = []string{""}
	}
	m.dirty = true
	m.clampCursor()
	m.setStatus("Formatted")
	return m, m.requestDiagCheckCmd()
}

func (m model) showHelp() (tea.Model, tea.Cmd) {
	m.outputLines = []string{
		"╔══════════════════════════════════════════════════════╗",
		"║              NT-IDE v" + IDEVersion + " — Keyboard Shortcuts        ║",
		"╠══════════════════════════════════════════════════════╣",
		"║  Ctrl+S      Save file                              ║",
		"║  Ctrl+R      Run file (saves first)                 ║",
		"║  Ctrl+F      Format/auto-organize code              ║",
		"║  Ctrl+C      Quit (prompts to save if dirty)        ║",
		"║  Ctrl+O      Open file                              ║",
		"║  Ctrl+D      Duplicate current line                 ║",
		"║  Ctrl+K      Delete current line                    ║",
		"║  Ctrl+P      Toggle output panel                    ║",
		"║  Ctrl+/      Toggle line comment                    ║",
		"╠══════════════════════════════════════════════════════╣",
		"║  Tab         Accept autocomplete / indent 2 spaces  ║",
		"║  Up/Down     Navigate autocomplete suggestions       ║",
		"║  Esc         Dismiss autocomplete / enter NORMAL     ║",
		"╠══════════════════════════════════════════════════════╣",
		"║  NORMAL mode (press Esc):                           ║",
		"║  i/a/o/O   Enter insert mode                       ║",
		"║  h/j/k/l   Navigate                                ║",
		"║  w/b/e     Word movement                           ║",
		"║  d         Delete line                             ║",
		"║  :w        Save   :q  Quit   :wq  Save+quit        ║",
		"║  :fmt      Format code                             ║",
		"║  :run      Run file                                ║",
		"╠══════════════════════════════════════════════════════╣",
		"║  Mouse: Click to place cursor, scroll wheel to scroll║",
		"╚══════════════════════════════════════════════════════╝",
	}
	m.showOutput = true
	m.setStatus("Help — Ctrl+P to toggle panel")
	return m, nil
}

// --- Autocomplete ---

func (m *model) updateAutocomplete() {
	line := m.currentLine()

	// Check if we're in an @import context (trigger even before dot)
	lineUpToCursor := ""
	if m.cx <= len([]rune(line)) {
		lineUpToCursor = string([]rune(line)[:m.cx])
	}
	trimmedUp := strings.TrimSpace(lineUpToCursor)
	isAtImport := strings.HasPrefix(trimmedUp, "@")

	items := GetCompletions(m.lines, line, m.cx)
	if len(items) == 0 {
		if !isAtImport {
			m.ac.Reset()
		}
		return
	}

	receiver, partial := dotContextAtCursor(line, m.cx)
	m.ac.Active = true
	m.ac.Items = items
	m.ac.Word = partial
	m.ac.Receiver = receiver
	m.ac.CursorX = m.cx
	m.ac.CursorY = m.cy
	if m.ac.Selected >= len(m.ac.Items) {
		m.ac.Selected = 0
	}
}

func (m model) applyCompletion() (tea.Model, tea.Cmd) {
	item := m.ac.Current()
	if item == nil {
		return m, nil
	}

	line := m.currentLine()
	runes := []rune(line)
	cx := m.cx

	// For @import completions: replace everything from the @ sign
	insertText := item.InsertText
	if insertText == "" {
		insertText = item.Label
	}

	lineUpToCursor := string(runes[:cx])
	trimmedUp := strings.TrimSpace(lineUpToCursor)
	isAtImport := strings.HasPrefix(trimmedUp, "@")

	var wordStart int
	if isAtImport && strings.HasPrefix(strings.TrimSpace(item.InsertText), "@import") {
		// Find the @ position
		atPos := strings.LastIndex(lineUpToCursor, "@")
		if atPos >= 0 {
			wordStart = atPos
		} else {
			wordStart = cx - len([]rune(m.ac.Word))
		}
	} else if m.ac.Receiver != "" {
		wordStart = cx - len([]rune(m.ac.Word))
	} else {
		wordStart = cx - len([]rune(m.ac.Word))
	}
	if wordStart < 0 {
		wordStart = 0
	}

	before := string(runes[:wordStart])
	after := string(runes[cx:])
	newLine := before + insertText + after
	m.lines[m.cy] = newLine
	m.cx = wordStart + len([]rune(insertText))

	// Handle multi-line snippets
	if strings.Contains(insertText, "\n") {
		insertLines := strings.Split(insertText, "\n")
		firstLine := before + insertLines[0]
		lastLine := insertLines[len(insertLines)-1] + after
		newLines := make([]string, 0, len(insertLines))
		newLines = append(newLines, firstLine)
		newLines = append(newLines, insertLines[1:len(insertLines)-1]...)
		newLines = append(newLines, lastLine)

		rest := append([]string{}, m.lines[m.cy+1:]...)
		m.lines = append(m.lines[:m.cy], newLines...)
		m.lines = append(m.lines, rest...)

		m.cy += 1
		m.cx = len([]rune(insertLines[1]))
	}

	m.ac.Reset()
	m.dirty = true
	return m, m.requestDiagCheckCmd()
}

// --- Diagnostics ---

func (m *model) requestDiagCheck() {
	m.diagPending = true
	m.diagCheckTime = time.Now()
}

func (m model) requestDiagCheckCmd() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}

func (m model) runDiagCheck() tea.Cmd {
	content := strings.Join(m.lines, "\n")
	fp := m.filePath
	return func() tea.Msg {
		result := CheckSource(content, fp)
		return diagMsg{result: result}
	}
}

// --- Text Editing ---

func (m model) insertChar(r rune) (model, tea.Cmd) {
	line := m.currentLine()
	runes := []rune(line)
	if m.cx > len(runes) {
		m.cx = len(runes)
	}

	newRunes := make([]rune, 0, len(runes)+2)
	newRunes = append(newRunes, runes[:m.cx]...)
	newRunes = append(newRunes, r)

	// Auto-close brackets
	closing := AutoCloseBracket(r)
	if closing != "" && ShouldAutoClose(line, m.cx, r) {
		if r != '"' && r != '\'' && r != '`' {
			newRunes = append(newRunes, []rune(closing)...)
		} else {
			newRunes = append(newRunes, []rune(closing)...)
		}
	}

	newRunes = append(newRunes, runes[m.cx:]...)
	m.lines[m.cy] = string(newRunes)
	m.cx++
	m.dirty = true
	m.updateAutocomplete()
	m.scroll()

	return m, m.requestDiagCheckCmd()
}

func (m model) insertText(s string) (tea.Model, tea.Cmd) {
	for _, r := range s {
		var cmd tea.Cmd
		m, cmd = m.insertChar(r)
		_ = cmd
	}
	m.dirty = true
	m.scroll()
	return m, nil
}

func (m model) insertNewline() (tea.Model, tea.Cmd) {
	line := m.currentLine()
	runes := []rune(line)
	if m.cx > len(runes) {
		m.cx = len(runes)
	}

	before := string(runes[:m.cx])
	after := string(runes[m.cx:])
	indent := LeadingIndent(before)

	// Increase indent after {
	newIndent := AutoIndent(before, indent)

	// Handle closing brace: if after starts with }, de-indent it
	trimAfter := strings.TrimSpace(after)
	if strings.HasPrefix(trimAfter, "}") || strings.HasPrefix(trimAfter, ")") || strings.HasPrefix(trimAfter, "]") {
		// Insert extra line for the closing brace at original indent
		middle := newIndent
		closingLine := indent + trimAfter
		m.lines[m.cy] = before
		rest := append([]string{middle, closingLine}, m.lines[m.cy+1:]...)
		m.lines = append(m.lines[:m.cy+1], rest...)
		m.cy++
		m.cx = len([]rune(newIndent))
	} else {
		newLine := newIndent + after
		m.lines[m.cy] = before
		m.lines = append(m.lines[:m.cy+1], append([]string{newLine}, m.lines[m.cy+1:]...)...)
		m.cy++
		m.cx = len([]rune(newIndent))
	}

	m.dirty = true
	m.ac.Reset()
	m.scroll()
	return m, m.requestDiagCheckCmd()
}

func (m model) doBackspace() (tea.Model, tea.Cmd) {
	line := m.currentLine()
	runes := []rune(line)

	if m.cx > 0 {
		// Delete char before cursor
		// Handle bracket pair deletion
		if m.cx < len(runes) {
			prev := runes[m.cx-1]
			next := runes[m.cx]
			closing := AutoCloseBracket(prev)
			if closing != "" && string(next) == closing {
				// Delete both opening and closing
				m.lines[m.cy] = string(runes[:m.cx-1]) + string(runes[m.cx+1:])
				m.cx--
				m.dirty = true
				m.scroll()
				return m, m.requestDiagCheckCmd()
			}
		}
		m.lines[m.cy] = string(runes[:m.cx-1]) + string(runes[m.cx:])
		m.cx--
	} else if m.cy > 0 {
		// Join with previous line
		prevLine := m.lines[m.cy-1]
		m.cx = len([]rune(prevLine))
		m.lines[m.cy-1] = prevLine + line
		m.lines = append(m.lines[:m.cy], m.lines[m.cy+1:]...)
		m.cy--
	}

	m.dirty = true
	m.updateAutocomplete()
	m.scroll()
	return m, m.requestDiagCheckCmd()
}

func (m model) deleteLine() (tea.Model, tea.Cmd) {
	if len(m.lines) == 1 {
		m.lines[0] = ""
		m.cx = 0
		m.dirty = true
		return m, nil
	}
	m.lines = append(m.lines[:m.cy], m.lines[m.cy+1:]...)
	if m.cy >= len(m.lines) {
		m.cy = len(m.lines) - 1
	}
	m.clampX()
	m.dirty = true
	return m, m.requestDiagCheckCmd()
}

func (m *model) deleteChar() {
	line := m.currentLine()
	runes := []rune(line)
	if m.cx < len(runes) {
		m.lines[m.cy] = string(runes[:m.cx]) + string(runes[m.cx+1:])
		m.dirty = true
	}
}

func (m model) duplicateLine() (tea.Model, tea.Cmd) {
	line := m.currentLine()
	m.lines = append(m.lines[:m.cy+1], append([]string{line}, m.lines[m.cy+1:]...)...)
	m.cy++
	m.dirty = true
	return m, nil
}

func (m model) moveLineUp() (tea.Model, tea.Cmd) {
	if m.cy == 0 {
		return m, nil
	}
	m.lines[m.cy], m.lines[m.cy-1] = m.lines[m.cy-1], m.lines[m.cy]
	m.cy--
	m.dirty = true
	return m, nil
}

func (m model) moveLineDown() (tea.Model, tea.Cmd) {
	if m.cy >= len(m.lines)-1 {
		return m, nil
	}
	m.lines[m.cy], m.lines[m.cy+1] = m.lines[m.cy+1], m.lines[m.cy]
	m.cy++
	m.dirty = true
	return m, nil
}

func (m model) toggleComment() (tea.Model, tea.Cmd) {
	line := m.currentLine()
	trimmed := strings.TrimSpace(line)
	indent := LeadingIndent(line)
	if strings.HasPrefix(trimmed, "//") {
		m.lines[m.cy] = indent + strings.TrimPrefix(strings.TrimPrefix(trimmed, "//"), " ")
	} else {
		m.lines[m.cy] = indent + "// " + trimmed
	}
	m.dirty = true
	return m, m.requestDiagCheckCmd()
}

func (m *model) insertLineBelow() {
	indent := LeadingIndent(m.currentLine())
	m.lines = append(m.lines[:m.cy+1], append([]string{indent}, m.lines[m.cy+1:]...)...)
}

func (m *model) insertLineAbove() {
	indent := LeadingIndent(m.currentLine())
	m.lines = append(m.lines[:m.cy], append([]string{indent}, m.lines[m.cy:]...)...)
}

// --- File operations ---

func (m *model) loadFile(path string) {
	abs, err := filepath.Abs(path)
	if err != nil {
		m.setStatus("Error: " + err.Error())
		return
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		m.setStatus("Error: " + err.Error())
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
	m.isNew = false
	m.ac.Reset()
	m.requestDiagCheck()
	m.setStatus("Opened " + filepath.Base(abs))
}

// --- Cursor helpers ---

func (m *model) currentLine() string {
	if m.cy >= len(m.lines) {
		return ""
	}
	return m.lines[m.cy]
}

func (m *model) clampX() {
	lineLen := len([]rune(m.currentLine()))
	if m.cx > lineLen {
		m.cx = lineLen
	}
}

func (m *model) clampCursor() {
	if m.cy >= len(m.lines) {
		m.cy = len(m.lines) - 1
	}
	if m.cy < 0 {
		m.cy = 0
	}
	m.clampX()
}

func (m *model) scroll() {
	ch := m.editorHeight()
	if m.cy < m.scrollY {
		m.scrollY = m.cy
	}
	if m.cy >= m.scrollY+ch {
		m.scrollY = m.cy - ch + 1
	}
	cw := m.editorWidth()
	if m.cx < m.scrollX {
		m.scrollX = m.cx
	}
	if m.cx >= m.scrollX+cw {
		m.scrollX = m.cx - cw + 1
	}
}

func (m *model) nextWordStart() int {
	line := m.currentLine()
	runes := []rune(line)
	i := m.cx
	for i < len(runes) && !unicode.IsSpace(runes[i]) {
		i++
	}
	for i < len(runes) && unicode.IsSpace(runes[i]) {
		i++
	}
	return i
}

func (m *model) prevWordStart() int {
	line := m.currentLine()
	runes := []rune(line)
	i := m.cx
	if i > 0 {
		i--
	}
	for i > 0 && unicode.IsSpace(runes[i]) {
		i--
	}
	for i > 0 && !unicode.IsSpace(runes[i-1]) {
		i--
	}
	return i
}

func (m *model) wordEnd() int {
	line := m.currentLine()
	runes := []rune(line)
	i := m.cx
	if i < len(runes) {
		i++
	}
	for i < len(runes) && unicode.IsSpace(runes[i]) {
		i++
	}
	for i < len(runes)-1 && !unicode.IsSpace(runes[i+1]) {
		i++
	}
	return i
}

// --- Status ---

func (m *model) setStatus(msg string) {
	m.statusMsg = msg
	if msg != "" {
		m.statusTimeout = time.Now().Add(4 * time.Second)
	}
}

// --- Layout helpers ---

func (m *model) lineNumWidth() int {
	n := len(m.lines)
	w := 1
	for n >= 10 {
		n /= 10
		w++
	}
	if w < 3 {
		w = 3
	}
	return w
}

func (m *model) editorWidth() int {
	w := m.width - m.lineNumWidth() - 2
	if w < 10 {
		w = 10
	}
	return w
}

func (m *model) editorBottom() int {
	bottom := m.height - 2 // title bar + status bar
	if m.showOutput {
		bottom -= m.outputHeight + 1
	}
	if len(m.diags.Diagnostics) > 0 {
		bottom -= m.errPanelHeight + 1
	}
	return bottom
}

func (m *model) editorHeight() int {
	h := m.editorBottom() - 1 // minus title bar
	if h < 1 {
		h = 1
	}
	return h
}

// --- View ---

func (m model) View() string {
	var sb strings.Builder
	sb.WriteString(m.renderTitleBar())
	sb.WriteString("\n")
	sb.WriteString(m.renderEditor())
	if len(m.diags.Diagnostics) > 0 {
		sb.WriteString("\n")
		sb.WriteString(m.renderErrorPanel())
	}
	if m.showOutput {
		sb.WriteString("\n")
		sb.WriteString(m.renderOutputPanel())
	}
	sb.WriteString("\n")
	sb.WriteString(m.renderStatusBar())

	// Overlay: autocomplete dropdown
	if m.ac.Active && len(m.ac.Items) > 0 {
		return m.overlayAutocomplete(sb.String())
	}

	// Overlay: save-as prompt
	if m.mode == modeSaveAs {
		return m.overlayPrompt(sb.String(), "Save as: "+m.saveAsName+"█")
	}

	// Overlay: open file prompt
	if m.mode == modeSearch {
		return m.overlayPrompt(sb.String(), "Open: "+m.searchQuery+"█")
	}

	// Overlay: quit-save
	if m.mode == modeQuitSave {
		return m.overlayPrompt(sb.String(), m.quitSaveMsg)
	}

	return sb.String()
}

func (m model) renderTitleBar() string {
	t := m.theme

	fileName := "untitled.lx"
	if m.filePath != "" {
		fileName = filepath.Base(m.filePath)
	}

	dirPart := ""
	if m.filePath != "" {
		dir := filepath.Dir(m.filePath)
		if dir != "." {
			dirPart = " " + dir + "/"
		}
	}

	dirStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(colorFgDim))
	nameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(colorFg)).Bold(true)

	dirtyMark := ""
	if m.dirty {
		dirtyMark = " " + t.Dirty.Render("●")
	}

	errBadge := ""
	n := len(m.diags.Diagnostics)
	if n > 0 {
		errBadge = " " + t.ErrorBadge.Render(fmt.Sprintf(" ✗ %d error%s ", n, plural(n)))
	} else if !m.diagPending {
		errBadge = " " + t.OkStatus.Render("✓")
	}

	left := "  NT-IDE " + dirStyle.Render(dirPart) + nameStyle.Render(fileName) + dirtyMark + errBadge
	right := " Lunex Language  v" + IDEVersion + "  "

	rightStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(colorFgDim))
	leftW := lipgloss.Width(left)
	rightW := lipgloss.Width(right)
	pad := m.width - leftW - rightW
	if pad < 0 {
		pad = 0
	}
	padding := strings.Repeat(" ", pad)

	bar := left + padding + rightStyle.Render(right)
	return t.TitleBar.Width(m.width).Render(bar)
}

func (m model) renderEditor() string {
	ch := m.editorHeight()
	lnw := m.lineNumWidth()
	cw := m.editorWidth()

	var sb strings.Builder
	t := m.theme

	// Collect error lines for quick lookup
	errLines := map[int]bool{}
	for _, d := range m.diags.Diagnostics {
		errLines[d.Line-1] = true
	}

	for row := 0; row < ch; row++ {
		lineIdx := row + m.scrollY

		// Line number
		var lineNumStr string
		if lineIdx >= len(m.lines) {
			lineNumStr = "~"
		} else {
			lineNumStr = fmt.Sprintf("%d", lineIdx+1)
		}

		var lineNumRendered string
		if lineIdx == m.cy {
			lineNumRendered = t.CurLineNum.Width(lnw).Align(lipgloss.Right).Render(lineNumStr)
		} else {
			lineNumRendered = t.LineNum.Width(lnw).Align(lipgloss.Right).Render(lineNumStr)
		}

		// Error indicator
		errInd := " "
		if errLines[lineIdx] {
			errInd = lipgloss.NewStyle().Foreground(lipgloss.Color(colorRed)).Render("●")
		}

		sb.WriteString(lineNumRendered)
		sb.WriteString(errInd)

		if lineIdx >= len(m.lines) {
			// Empty line past end of file
			sb.WriteString(strings.Repeat(" ", cw))
		} else {
			line := m.lines[lineIdx]
			isCurrentLine := lineIdx == m.cy
			rendered := renderHighlightedLine(line, m.cx, m.scrollX, cw, isCurrentLine, m.mode)
			sb.WriteString(rendered)
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func (m model) renderErrorPanel() string {
	t := m.theme
	header := t.OutputHeader.Width(m.width).Render(
		" ✗ Errors (" + itoa(len(m.diags.Diagnostics)) + ") ",
	)

	maxW := m.width - 4
	if maxW < 10 {
		maxW = 10
	}

	// Wrap each diagnostic into display lines
	var lines []string
	for i, d := range m.diags.Diagnostics {
		prefix := "  "
		if i == m.errPanelSel {
			prefix = "▶ "
		}
		header1 := prefix + "L" + itoa(d.Line) + ": "
		msg := d.Message

		// First line: prefix + location + start of message
		first := header1 + msg
		if len([]rune(first)) <= maxW {
			lines = append(lines, t.ErrorMsg.Render(first))
		} else {
			// Fit as much of msg on first line as possible
			avail := maxW - len([]rune(header1))
			if avail < 8 {
				avail = 8
			}
			msgRunes := []rune(msg)
			if avail > len(msgRunes) {
				avail = len(msgRunes)
			}
			lines = append(lines, t.ErrorMsg.Render(header1+string(msgRunes[:avail])))
			// Continuation lines
			rest := msgRunes[avail:]
			for len(rest) > 0 {
				chunk := maxW - 4
				if chunk > len(rest) {
					chunk = len(rest)
				}
				lines = append(lines, t.ErrorMsg.Render("    "+string(rest[:chunk])))
				rest = rest[chunk:]
			}
		}
	}

	count := m.errPanelHeight
	if len(lines) < count {
		count = len(lines)
	}

	var sb strings.Builder
	sb.WriteString(header)
	for i := 0; i < count; i++ {
		sb.WriteString("\n")
		if i < len(lines) {
			sb.WriteString(lines[i])
		}
	}
	return sb.String()
}

func (m model) renderOutputPanel() string {
	t := m.theme
	runInfo := ""
	if m.running {
		runInfo = " ⟳ running..."
	} else if m.runDuration > 0 {
		runInfo = fmt.Sprintf(" (%dms)", m.runDuration.Milliseconds())
	}

	statusIcon := " ▶"
	statusColor := colorAccent
	if m.outputErr {
		statusIcon = " ✗"
		statusColor = colorRed
	} else if !m.running && m.runDuration > 0 {
		statusIcon = " ✓"
		statusColor = colorGreen
	}

	iconStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(statusColor)).Bold(true)
	header := t.OutputHeader.Width(m.width).Render(
		iconStyle.Render(statusIcon) + " Output" + runInfo + "  (Ctrl+P to toggle)",
	)

	h := m.outputHeight
	total := len(m.outputLines)
	start := m.outputScroll
	if start > total-h {
		start = total - h
	}
	if start < 0 {
		start = 0
	}

	var sb strings.Builder
	sb.WriteString(header)
	for i := 0; i < h; i++ {
		sb.WriteString("\n")
		idx := start + i
		if idx < total {
			line := m.outputLines[idx]
			lineRunes := []rune(line)
			maxOut := m.width - 4
			if maxOut < 1 {
				maxOut = 1
			}
			if len(lineRunes) > maxOut {
				line = string(lineRunes[:maxOut])
			}
			if m.outputErr {
				sb.WriteString(t.OutputErr.Render("  " + line))
			} else {
				sb.WriteString(t.OutputText.Render("  " + line))
			}
		} else {
			sb.WriteString("  ")
		}
	}
	return sb.String()
}

func (m model) renderStatusBar() string {
	t := m.theme

	// Mode badge
	var modeBadge string
	switch m.mode {
	case modeInsert:
		modeBadge = t.ModeInsert.Render("  INSERT  ")
	case modeNormal:
		modeBadge = t.ModeNormal.Render("  NORMAL  ")
	case modeCommand:
		modeBadge = t.ModeCommand.Render("  :" + m.cmdBuf + "█  ")
	case modeSaveAs:
		modeBadge = t.ModeSave.Render("  SAVE AS  ")
	case modeQuitSave:
		modeBadge = t.ModeSave.Render("  QUIT  ")
	case modeSearch:
		modeBadge = t.ModeCommand.Render("  OPEN  ")
	default:
		if m.running {
			modeBadge = t.ModeRun.Render("  RUNNING  ")
		} else {
			modeBadge = t.ModeInsert.Render("  INSERT  ")
		}
	}

	// Position
	pos := fmt.Sprintf("  Ln %d, Col %d  ", m.cy+1, m.cx+1)
	posStyle := t.Position

	// Status message or hints (adaptive to screen width)
	statusText := m.statusMsg
	if statusText == "" {
		if m.mode == modeInsert {
			avail := m.width - lipgloss.Width(modeBadge) - lipgloss.Width(pos) - 2
			if avail >= 60 {
				statusText = " ^R:Run  ^S:Save  ^F:Fmt  Tab:Complete  ^H:Help"
			} else if avail >= 36 {
				statusText = " ^R:Run  ^S:Save  ^H:Help"
			} else if avail >= 18 {
				statusText = " ^H:Help"
			}
		}
	}
	statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(colorFgDim))

	// Running indicator
	runIndicator := ""
	if m.running {
		runIndicator = lipgloss.NewStyle().Foreground(lipgloss.Color(colorTeal)).Render(" ⟳ ")
	}

	left := modeBadge + runIndicator + statusStyle.Render(statusText)
	right := posStyle.Render(pos)

	leftW := lipgloss.Width(left)
	rightW := lipgloss.Width(right)
	pad := m.width - leftW - rightW
	if pad < 0 {
		pad = 0
	}
	bar := left + strings.Repeat(" ", pad) + right
	return t.StatusBar.Width(m.width).Render(bar)
}

// overlayAutocomplete places the autocomplete dropdown over the existing view
func (m model) overlayAutocomplete(base string) string {
	items := m.ac.Items
	if len(items) == 0 {
		return base
	}

	// Max visible items
	maxItems := m.ac.MaxItems
	if maxItems > len(items) {
		maxItems = len(items)
	}

	// Find widest label
	maxLabelW := 0
	maxDetailW := 0
	for i := m.ac.ScrollY; i < m.ac.ScrollY+maxItems && i < len(items); i++ {
		lw := len([]rune(items[i].Label))
		dw := len([]rune(items[i].Detail))
		if lw > maxLabelW { maxLabelW = lw }
		if dw > maxDetailW { maxDetailW = dw }
	}
	if maxLabelW > 24 { maxLabelW = 24 }
	if maxDetailW > 32 { maxDetailW = 32 }

	// Render each row
	rows := make([]string, 0, maxItems+2)
	for i := m.ac.ScrollY; i < m.ac.ScrollY+maxItems && i < len(items); i++ {
		item := items[i]
		kindIcon := lipgloss.NewStyle().
			Foreground(lipgloss.Color(item.Kind.Color())).
			Bold(true).
			Render(item.Kind.Icon())

		label := item.Label
		if len([]rune(label)) > maxLabelW {
			label = string([]rune(label)[:maxLabelW-1]) + "…"
		}
		labelPadded := label + strings.Repeat(" ", maxLabelW-len([]rune(label)))

		detail := item.Detail
		if len([]rune(detail)) > maxDetailW {
			detail = string([]rune(detail)[:maxDetailW-1]) + "…"
		}
		detailPadded := detail + strings.Repeat(" ", maxDetailW-len([]rune(detail)))

		var labelStyle lipgloss.Style
		var detailStyle lipgloss.Style

		if i == m.ac.Selected {
			labelStyle = lipgloss.NewStyle().Background(lipgloss.Color(colorAccent)).
				Foreground(lipgloss.Color("#ffffff")).Bold(true)
			detailStyle = lipgloss.NewStyle().Background(lipgloss.Color(colorAccent)).
				Foreground(lipgloss.Color("#d0e8ff"))
		} else {
			labelStyle = lipgloss.NewStyle().Background(lipgloss.Color(colorBgAC)).
				Foreground(lipgloss.Color(colorFg))
			detailStyle = lipgloss.NewStyle().Background(lipgloss.Color(colorBgAC)).
				Foreground(lipgloss.Color(colorFgDim))
		}

		row := " " + kindIcon + " " + labelStyle.Render(labelPadded) + "  " + detailStyle.Render(detailPadded) + " "
		rows = append(rows, row)
	}

	// Scroll indicator
	if len(items) > maxItems {
		scrollInfo := fmt.Sprintf(" %d/%d ", m.ac.Selected+1, len(items))
		rows = append(rows, lipgloss.NewStyle().
			Background(lipgloss.Color(colorBgAC)).
			Foreground(lipgloss.Color(colorFgDim)).
			Render(scrollInfo))
	}

	// Build the dropdown box
	boxW := maxLabelW + maxDetailW + 10
	boxLines := make([]string, len(rows))
	for i, row := range rows {
		rw := lipgloss.Width(row)
		if rw < boxW {
			row += strings.Repeat(" ", boxW-rw)
		}
		boxLines[i] = row
	}

	// Overlay box into base string
	// Compute position: right below cursor
	dropX := m.lineNumWidth() + 2 + (m.cx - m.scrollX)
	dropY := 1 + (m.cy - m.scrollY) + 1 // +1 for title bar, +1 for below cursor

	// Clamp to screen
	if dropX+boxW > m.width {
		dropX = m.width - boxW
		if dropX < 0 { dropX = 0 }
	}
	if dropY+len(boxLines) > m.height-1 {
		// Show above cursor instead
		dropY = 1 + (m.cy - m.scrollY) - len(boxLines)
		if dropY < 1 { dropY = 1 }
	}

	return overlayBox(base, boxLines, dropX, dropY, m.width)
}

// overlayBox places a box of lines at (x, y) over the base string
func overlayBox(base string, box []string, x, y, width int) string {
	baseLines := strings.Split(base, "\n")
	for i, boxLine := range box {
		targetY := y + i
		if targetY < 0 || targetY >= len(baseLines) {
			continue
		}

		bl := baseLines[targetY]
		blRunes := []rune(stripANSI(bl))

		// Pad to x if needed
		for len(blRunes) < x {
			blRunes = append(blRunes, ' ')
		}

		boxRunes := []rune(stripANSI(boxLine))
		boxW := len(boxRunes)

		// Build new line: before + boxLine + after
		before := blRunes[:x]
		afterStart := x + boxW
		var after []rune
		if afterStart < len(blRunes) {
			after = blRunes[afterStart:]
		}

		newLine := string(before) + boxLine + string(after)
		// Clamp to width
		nw := lipgloss.Width(newLine)
		if nw > width {
			// Simple truncation
			newLine = string([]rune(newLine)[:width])
		}
		baseLines[targetY] = newLine
	}
	return strings.Join(baseLines, "\n")
}

// overlayPrompt renders a centered prompt over the view
func (m model) overlayPrompt(base, prompt string) string {
	promptStyle := lipgloss.NewStyle().
		Background(lipgloss.Color(colorBgPanel)).
		Foreground(lipgloss.Color(colorFg)).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(colorAccent)).
		Padding(0, 2)

	box := promptStyle.Render(prompt)
	bh := lipgloss.Height(box)
	bw := lipgloss.Width(box)

	y := (m.height - bh) / 2
	x := (m.width - bw) / 2
	if x < 0 { x = 0 }
	if y < 0 { y = 1 }

	boxLines := strings.Split(box, "\n")
	return overlayBox(base, boxLines, x, y, m.width)
}

// stripANSI removes ANSI escape sequences from a string (for width calculation)
func stripANSI(s string) string {
	var out strings.Builder
	inEsc := false
	for _, r := range s {
		if r == '\x1b' {
			inEsc = true
			continue
		}
		if inEsc {
			if r == 'm' || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inEsc = false
			}
			continue
		}
		out.WriteRune(r)
	}
	return out.String()
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
