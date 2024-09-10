package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type Difficulty int

const (
	Easy Difficulty = iota
	Medium
	Hard
)

func (d Difficulty) String() string {
	return [...]string{"Easy", "Medium", "Hard"}[d]
}

type MenuModel struct {
	choices  []string
	cursor   int
	selected int
	width    int
	height   int
}

func NewMenuModel(width, height int) *MenuModel {
	return &MenuModel{
		choices: []string{"Easy", "Medium", "Hard", "Quit"},
		width:   width,
		height:  height,
	}
}

func (m MenuModel) Init() tea.Cmd {
	return nil
}

func (m MenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case "enter":
			m.selected = m.cursor
			if m.selected == len(m.choices)-1 {
				return m, tea.Quit
			}
			return NewGameModel(m.width, m.height, Difficulty(m.selected)), nil
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func (m MenuModel) View() string {
	s := boldWhite + "Sudoku Game" + reset + "\n\n"
	s += "Select difficulty:\n\n"

	for i, choice := range m.choices {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		s += fmt.Sprintf("%s %s\n", cursor, choice)
	}

	// Center the menu
	lines := strings.Split(s, "\n")
	centeredLines := make([]string, len(lines))
	for i, line := range lines {
		centeredLines[i] = centerText(line, m.width)
	}

	return strings.Join(centeredLines, "\n")
}

