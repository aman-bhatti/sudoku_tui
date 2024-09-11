package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	cellStyle = func(modifiable bool) lipgloss.Style {
		if modifiable {
			return lipgloss.NewStyle().
				PaddingLeft(1).PaddingRight(1).
				Background(lipgloss.Color("240")).
				Foreground(lipgloss.Color("15")) // Modifiable cells: light gray background, white text
		} else {
			return lipgloss.NewStyle().
				PaddingLeft(1).PaddingRight(1).
				Background(lipgloss.Color("236")) // Non-modifiable cells: dark gray background
		}
	}

	cursorCellStyle = func(modifiable bool) lipgloss.Style {
		if modifiable {
			return lipgloss.NewStyle().
				PaddingLeft(1).PaddingRight(1).
				Background(lipgloss.Color("34")) // Modifiable cell with cursor: green background
		} else {
			return lipgloss.NewStyle().
				PaddingLeft(1).PaddingRight(1).
				Background(lipgloss.Color("22")) // Non-modifiable cell with cursor: dark green background
		}
	}

	errorCellStyle = func(isCursor bool) lipgloss.Style {
		if isCursor {
			// Red background with white text when the cursor is on an error cell
			return lipgloss.NewStyle().PaddingLeft(1).PaddingRight(1).
				Background(lipgloss.Color("160")).Foreground(lipgloss.Color("15"))
		} else {
			// Solid red background for non-cursor error cells
			return lipgloss.NewStyle().PaddingLeft(1).PaddingRight(1).
				Background(lipgloss.Color("196")).Foreground(lipgloss.Color("15"))
		}
	}

	formatCell = func(isError, isCursor, modifiable bool, row, col int, c string) string {
		var s lipgloss.Style

		if isError {
			// Apply the error style for incorrect cells
			s = errorCellStyle(isCursor)
		} else if isCursor {
			// Apply the cursor style when the cursor is on the cell
			s = cursorCellStyle(modifiable)
		} else {
			// Apply the normal cell style
			s = cellStyle(modifiable)
		}

		// Add vertical borders between groups of 3 cells
		if col+1 == 3 || col+1 == 6 {
			return s.Render(c) + lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), false, true, false, false).Margin(0, 1).Render("")
		}

		return s.Render(c)
	}

	formatRow = func(row int, r string) string {
		// Add horizontal borders between groups of 3 rows
		if row+1 == 3 || row+1 == 6 {
			rSize, _ := lipgloss.Size(r)
			border := strings.Repeat("─", (rSize/3)-1)
			return r + "\n" + border + "┼" + "─" + border + "┼" + border
		}
		return r
	}

	// Style for the cells left and info text at the bottom
	cellsLeftStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Margin(1, 0, 0, 0)
)
