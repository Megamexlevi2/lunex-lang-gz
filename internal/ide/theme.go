// NT-IDE — Lunex Integrated Development Environment
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package ide

import "github.com/charmbracelet/lipgloss"

// Color palette — deep dark theme with vivid accents
const (
	colorBg          = "#0d1117"
	colorBgPanel     = "#161b22"
	colorBgSelected  = "#1f2937"
	colorBgCursor    = "#1d4ed8"
	colorBgAC        = "#1c2333"
	colorBgACHover   = "#2d333b"
	colorBgError     = "#3b0000"
	colorBgOutput    = "#0d1117"
	colorBgTitleBar  = "#010409"
	colorBgStatusBar = "#010409"

	colorFg         = "#e6edf3"
	colorFgDim      = "#8b949e"
	colorFgMuted    = "#484f58"
	colorFgLineNum  = "#3d444d"
	colorFgCurLine  = "#6e7681"

	colorAccent  = "#58a6ff"
	colorGreen   = "#3fb950"
	colorYellow  = "#d29922"
	colorRed     = "#f85149"
	colorOrange  = "#d18616"
	colorPurple  = "#bc8cff"
	colorPink    = "#ff7b72"
	colorCyan    = "#79c0ff"
	colorTeal    = "#56d364"

	// Syntax colors
	colorKeyword  = "#ff7b72"
	colorString   = "#a5d6ff"
	colorNumber   = "#79c0ff"
	colorComment  = "#8b949e"
	colorFnName   = "#d2a8ff"
	colorModule   = "#79c0ff"
	colorOperator = "#ff7b72"
	colorPunct    = "#e6edf3"
	colorIdent    = "#e6edf3"
	colorBuiltin  = "#ffa657"
	colorTemplate = "#a5d6ff"
)

type Theme struct {
	TitleBar  lipgloss.Style
	StatusBar lipgloss.Style
	LineNum   lipgloss.Style
	CurLineNum lipgloss.Style
	Cursor    lipgloss.Style
	Selection lipgloss.Style

	// Autocomplete
	ACBox       lipgloss.Style
	ACItem      lipgloss.Style
	ACSelected  lipgloss.Style
	ACKind      lipgloss.Style
	ACDetail    lipgloss.Style

	// Diagnostics
	ErrorLine   lipgloss.Style
	ErrorMsg    lipgloss.Style
	WarnMsg     lipgloss.Style
	ErrorBadge  lipgloss.Style

	// Output panel
	OutputPanel  lipgloss.Style
	OutputHeader lipgloss.Style
	OutputText   lipgloss.Style
	OutputErr    lipgloss.Style

	// Status bar parts
	ModeInsert  lipgloss.Style
	ModeNormal  lipgloss.Style
	ModeCommand lipgloss.Style
	ModeSave    lipgloss.Style
	ModeRun     lipgloss.Style
	Position    lipgloss.Style
	FileName    lipgloss.Style
	Dirty       lipgloss.Style
	ErrorCount  lipgloss.Style
	OkStatus    lipgloss.Style

	// Borders
	SepH lipgloss.Style
}

var DefaultTheme = buildTheme()

func buildTheme() Theme {
	base := lipgloss.NewStyle()
	t := Theme{}

	t.TitleBar = base.Background(lipgloss.Color(colorBgTitleBar)).
		Foreground(lipgloss.Color(colorFg)).Bold(true)

	t.StatusBar = base.Background(lipgloss.Color(colorBgStatusBar)).
		Foreground(lipgloss.Color(colorFgDim))

	t.LineNum = base.Foreground(lipgloss.Color(colorFgLineNum))
	t.CurLineNum = base.Foreground(lipgloss.Color(colorAccent)).Bold(true)
	t.Cursor = base.Background(lipgloss.Color(colorBgCursor)).Foreground(lipgloss.Color("#ffffff"))
	t.Selection = base.Background(lipgloss.Color(colorBgSelected))

	t.ACBox = base.Background(lipgloss.Color(colorBgAC)).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(colorAccent))
	t.ACItem = base.Background(lipgloss.Color(colorBgAC)).
		Foreground(lipgloss.Color(colorFg))
	t.ACSelected = base.Background(lipgloss.Color(colorAccent)).
		Foreground(lipgloss.Color("#ffffff")).Bold(true)
	t.ACKind = base.Foreground(lipgloss.Color(colorPurple))
	t.ACDetail = base.Foreground(lipgloss.Color(colorFgDim))

	t.ErrorLine = base.Background(lipgloss.Color(colorBgError))
	t.ErrorMsg = base.Foreground(lipgloss.Color(colorRed))
	t.WarnMsg = base.Foreground(lipgloss.Color(colorYellow))
	t.ErrorBadge = base.Background(lipgloss.Color(colorRed)).
		Foreground(lipgloss.Color("#ffffff")).Bold(true).Padding(0, 1)

	t.OutputPanel = base.Background(lipgloss.Color(colorBgOutput))
	t.OutputHeader = base.Background(lipgloss.Color(colorBgPanel)).
		Foreground(lipgloss.Color(colorAccent)).Bold(true)
	t.OutputText = base.Foreground(lipgloss.Color(colorFg))
	t.OutputErr = base.Foreground(lipgloss.Color(colorRed))

	t.ModeInsert = base.Background(lipgloss.Color(colorGreen)).
		Foreground(lipgloss.Color("#000000")).Bold(true).Padding(0, 1)
	t.ModeNormal = base.Background(lipgloss.Color(colorAccent)).
		Foreground(lipgloss.Color("#000000")).Bold(true).Padding(0, 1)
	t.ModeCommand = base.Background(lipgloss.Color(colorYellow)).
		Foreground(lipgloss.Color("#000000")).Bold(true).Padding(0, 1)
	t.ModeSave = base.Background(lipgloss.Color(colorOrange)).
		Foreground(lipgloss.Color("#000000")).Bold(true).Padding(0, 1)
	t.ModeRun = base.Background(lipgloss.Color(colorTeal)).
		Foreground(lipgloss.Color("#000000")).Bold(true).Padding(0, 1)

	t.Position = base.Foreground(lipgloss.Color(colorFgDim))
	t.FileName = base.Foreground(lipgloss.Color(colorFg)).Bold(true)
	t.Dirty = base.Foreground(lipgloss.Color(colorYellow)).Bold(true)
	t.ErrorCount = base.Background(lipgloss.Color(colorRed)).
		Foreground(lipgloss.Color("#ffffff")).Bold(true).Padding(0, 1)
	t.OkStatus = base.Foreground(lipgloss.Color(colorGreen))

	t.SepH = base.Foreground(lipgloss.Color(colorFgMuted))

	return t
}
