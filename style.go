package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	cellStyle = func(modifiable bool) lipgloss.Style {
		if modifiable {
			return lipgloss.NewStyle().
				PaddingLeft(1).PaddingRight(1).
				Background(lipgloss.Color("240")).
				Foreground(lipgloss.Color("15"))
		} else {
			return lipgloss.NewStyle().
				PaddingLeft(1).PaddingRight(1).
				Background(lipgloss.Color("236"))
		}
	}

	cursorCellStyle = func(modifiable bool) lipgloss.Style {
		if modifiable {
			return lipgloss.NewStyle().
				PaddingLeft(1).PaddingRight(1).
				Background(lipgloss.Color("34"))
		} else {
			return lipgloss.NewStyle().
				PaddingLeft(1).PaddingRight(1).
				Background(lipgloss.Color("22"))
		}
	}

	errorCellStyle = func(isCursor bool) lipgloss.Style {
		if isCursor {
			return lipgloss.NewStyle().
				PaddingLeft(1).PaddingRight(1).
				Background(lipgloss.Color("160")).
				Foreground(lipgloss.Color("15"))
		} else {
			return lipgloss.NewStyle().
				PaddingLeft(1).PaddingRight(1).
				Background(lipgloss.Color("196")).
				Foreground(lipgloss.Color("15"))
		}
	}

	formatCell = func(isError, isCursor, modifiable bool, row, col int, c string) string {
		var s lipgloss.Style

		if isError {
			s = errorCellStyle(isCursor)
			fmt.Printf("Applying error style to cell (%d, %d)\n", row, col)
		} else if isCursor {
			s = cursorCellStyle(modifiable)
		} else {
			s = cellStyle(modifiable)
		}

		renderedCell := s.Render(c)

		if col+1 == 3 || col+1 == 6 {
			renderedCell += lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), false, true, false, false).
				Margin(0, 1).
				Render("")
		}

		return renderedCell
	}

	formatRow = func(row int, r string) string {
		if row+1 == 3 || row+1 == 6 {
			rSize := lipgloss.Width(r)
			border := strings.Repeat("─", (rSize/3)-1)
			return r + "\n" + border + "┼" + "─" + border + "┼" + border
		}
		return r
	}

	cellsLeftStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Margin(1, 0, 0, 0)
)

