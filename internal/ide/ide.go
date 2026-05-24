// NT-IDE — Lunex Integrated Development Environment
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

// Package ide provides a full-featured terminal IDE for the Lunex programming language.
// Launch with: lunex ide run [file.lx]
package ide

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

const IDEVersion = "1.0.0"

// Run launches the NT-IDE. filePath may be empty to start with a blank file.
func Run(filePath string) {
	m := newModel(filePath)
	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "nt-ide: %v\n", err)
		os.Exit(1)
	}
}
