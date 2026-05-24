// NT-IDE — Lunex Integrated Development Environment
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package ide

import (
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"
	"lunex/internal/lexer"
)

// Token kinds for highlighting
type hlKind int

const (
	hlDefault  hlKind = iota
	hlKeyword
	hlString
	hlTemplate
	hlNumber
	hlComment
	hlFunction
	hlModule
	hlOperator
	hlPunct
	hlBuiltin
	hlRegex
)

var kindColor = map[hlKind]string{
	hlKeyword:  colorKeyword,
	hlString:   colorString,
	hlTemplate: colorTemplate,
	hlNumber:   colorNumber,
	hlComment:  colorComment,
	hlFunction: colorFnName,
	hlModule:   colorModule,
	hlOperator: colorOperator,
	hlPunct:    colorPunct,
	hlBuiltin:  colorBuiltin,
	hlRegex:    colorString,
	hlDefault:  colorIdent,
}

// builtinIdents are common Lunex global identifiers
var builtinIdents = map[string]bool{
	"Math": true, "JSON": true, "Object": true, "Array": true,
	"String": true, "Number": true, "Boolean": true,
	"io": true, "fs": true, "http": true, "crypto": true,
	"db": true, "env": true, "validate": true, "events": true,
	"cache": true, "logger": true, "queue": true, "ws": true,
	"mail": true, "ai": true, "test": true, "alloc": true,
	"math": true, "datetime": true, "compress": true,
	"os": true, "path": true,
}

// Tokenize a single line for syntax highlighting.
// Returns a list of (text, color) segments.
type hlSegment struct {
	text  string
	color string
	bold  bool
}

func highlightLine(line string) []hlSegment {
	if line == "" {
		return nil
	}

	// Fast path: check if it's a comment line
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") {
		return []hlSegment{{text: line, color: colorComment}}
	}

	// Use the Lunex lexer for accurate tokenization
	tokens, err := lexer.Tokenize(line, "<highlight>")
	if err != nil || len(tokens) == 0 {
		return []hlSegment{{text: line, color: colorIdent}}
	}

	segs := make([]hlSegment, 0, len(tokens))
	lastPos := 0
	runes := []rune(line)
	totalLen := len(runes)

	for i, tok := range tokens {
		if tok.Type == lexer.EOF {
			break
		}

		tokStart := tok.Col - 1
		if tokStart < 0 {
			tokStart = 0
		}
		// Guard: never go backwards (overlapping tokens)
		if tokStart < lastPos {
			tokStart = lastPos
		}
		if tokStart >= totalLen {
			break
		}

		// Fill gap from lastPos to tokStart with default color
		if tokStart > lastPos {
			segs = append(segs, hlSegment{text: string(runes[lastPos:tokStart]), color: colorIdent})
		}

		// Always extract raw text from source runes to avoid losing delimiters.
		// tok.Raw is correct when present; when empty, tok.StrVal() strips quote
		// characters (e.g. "hello" -> hello), causing the opening quote to vanish.
		// We use the next token's column as the end boundary instead.
		var raw string
		if tok.Raw != "" {
			rawRunes := []rune(tok.Raw)
			end := tokStart + len(rawRunes)
			if end > totalLen {
				end = totalLen
			}
			raw = string(runes[tokStart:end])
		} else {
			// Find next token's start to bound this token in the source
			end := totalLen
			for j := i + 1; j < len(tokens); j++ {
				if tokens[j].Type == lexer.EOF {
					break
				}
				ns := tokens[j].Col - 1
				if ns > tokStart && ns <= totalLen {
					end = ns
					break
				}
			}
			raw = string(runes[tokStart:end])
		}

		if raw == "" {
			lastPos = tokStart
			continue
		}

		rawLen := len([]rune(raw))
		seg := hlSegment{text: raw, color: colorIdent}

		switch tok.Type {
		case lexer.KEYWORD:
			seg.color = colorKeyword
			seg.bold = true
		case lexer.STRING:
			seg.color = colorString
		case lexer.TEMPLATE:
			seg.color = colorTemplate
		case lexer.NUMBER:
			seg.color = colorNumber
		case lexer.REGEX:
			seg.color = colorString
		case lexer.OPERATOR:
			seg.color = colorOperator
		case lexer.PUNCTUATION:
			seg.color = colorPunct
		case lexer.IDENTIFIER:
			val := tok.StrVal()
			if builtinIdents[val] {
				seg.color = colorBuiltin
			} else if i > 0 && tokens[i-1].StrVal() == "fn" {
				seg.color = colorFnName
				seg.bold = true
			} else if i+1 < len(tokens) && tokens[i+1].StrVal() == "(" {
				seg.color = colorFnName
			} else {
				seg.color = colorIdent
			}
		}

		segs = append(segs, seg)
		lastPos = tokStart + rawLen
	}

	// Cover any remaining source characters after the last token
	if lastPos < totalLen {
		segs = append(segs, hlSegment{text: string(runes[lastPos:]), color: colorIdent})
	}

	return segs
}

// renderHighlightedLine renders a highlighted line as a styled string.
// cursorX: cursor column (-1 = no cursor on this line)
// scrollX: horizontal scroll offset
// maxWidth: maximum display width
// isCurrentLine: whether this line has the cursor
func renderHighlightedLine(line string, cursorX, scrollX, maxWidth int, isCurrentLine bool, mode editorMode) string {
	if maxWidth <= 0 {
		return ""
	}

	runes := []rune(line)
	totalLen := len(runes)

	// Apply scroll offset
	startRune := scrollX
	if startRune > totalLen {
		startRune = totalLen
	}
	visibleRunes := runes[startRune:]

	// Clamp to maxWidth
	if len(visibleRunes) > maxWidth {
		visibleRunes = visibleRunes[:maxWidth]
	}

	visibleLine := string(visibleRunes)

	// Compute cursor position in visible area
	cursorInVisible := -1
	if isCurrentLine && cursorX >= scrollX && cursorX-scrollX < len(visibleRunes) {
		cursorInVisible = cursorX - scrollX
	} else if isCurrentLine && cursorX-scrollX == len(visibleRunes) {
		cursorInVisible = len(visibleRunes)
	}

	// Get highlight segments for the visible portion
	segs := highlightLine(visibleLine)
	if len(segs) == 0 {
		segs = []hlSegment{{text: visibleLine, color: colorIdent}}
	}

	// Build the rendered string
	var sb strings.Builder
	pos := 0

	for _, seg := range segs {
		segRunes := []rune(seg.text)
		for _, r := range segRunes {
			style := lipgloss.NewStyle().Foreground(lipgloss.Color(seg.color))
			if seg.bold {
				style = style.Bold(true)
			}

			// Apply cursor highlight
			if isCurrentLine && pos == cursorInVisible && (mode == modeInsert || mode == modeNormal) {
				style = lipgloss.NewStyle().
					Background(lipgloss.Color(colorBgCursor)).
					Foreground(lipgloss.Color("#ffffff"))
			}

			sb.WriteString(style.Render(string(r)))
			pos++
		}
	}

	// Render cursor at end of line (past all characters)
	if isCurrentLine && cursorInVisible == len(visibleRunes) && (mode == modeInsert || mode == modeNormal) {
		cursorStyle := lipgloss.NewStyle().
			Background(lipgloss.Color(colorBgCursor)).
			Foreground(lipgloss.Color("#ffffff"))
		sb.WriteString(cursorStyle.Render(" "))
		pos++
	}

	// Pad to maxWidth
	rendered := sb.String()
	visWidth := lipgloss.Width(rendered)
	if visWidth < maxWidth {
		sb.WriteString(strings.Repeat(" ", maxWidth-visWidth))
	}

	// Highlight current line background subtly
	if isCurrentLine {
		lineStyle := lipgloss.NewStyle().Background(lipgloss.Color("#161b22"))
		return lineStyle.Render(sb.String())
	}

	return sb.String()
}

// isWordChar returns true if the character is valid in an identifier
func isWordChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '$'
}

// wordAtCursor extracts the word being typed at cursor position
func wordAtCursor(line string, cx int) string {
	runes := []rune(line)
	if cx > len(runes) {
		cx = len(runes)
	}
	i := cx - 1
	for i >= 0 && isWordChar(runes[i]) {
		i--
	}
	return string(runes[i+1 : cx])
}

// dotContextAtCursor returns the receiver before the dot, if any
// e.g. "io.lo" at cursor 5 → "io", "lo"
func dotContextAtCursor(line string, cx int) (receiver, partial string) {
	runes := []rune(line)
	if cx > len(runes) {
		cx = len(runes)
	}

	// Get word before cursor
	i := cx - 1
	for i >= 0 && isWordChar(runes[i]) {
		i--
	}
	partial = string(runes[i+1 : cx])

	// Check if there's a dot before the partial word
	dotPos := i
	if dotPos >= 0 && runes[dotPos] == '.' {
		// Get receiver before dot
		j := dotPos - 1
		for j >= 0 && isWordChar(runes[j]) {
			j--
		}
		receiver = string(runes[j+1 : dotPos])
	}

	return receiver, partial
}
