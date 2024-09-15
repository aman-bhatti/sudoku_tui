package main

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	menuBgColor := lipgloss.Color("11")
	var cursorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("9")).
		Background(menuBgColor).
		Bold(true)
	s := lipgloss.NewStyle().
		Foreground(lipgloss.Color("0")).
		Background(menuBgColor).
		Bold(true).
		Render("SUDOKU BUILT BY AMAN") + "\n\n"
	s += lipgloss.NewStyle().
		Foreground(lipgloss.Color("0")).
		Background(menuBgColor).
		Render("Select difficulty:") + "\n"
	for i, choice := range m.choices {
		cursor := " "
		if m.cursor == i {
			cursor = cursorStyle.Render(">")
		}
		choiceStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).
			Background(menuBgColor)
		if m.cursor == i {
			choiceStyle = choiceStyle.
				Foreground(lipgloss.Color("201")).
				Bold(true).
				Background(menuBgColor)
		}
		s += fmt.Sprintf("%s%s\n", cursor, choiceStyle.Render(choice))
	}

	// Create a box for the menu
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("11")).
		BorderBackground(lipgloss.Color("11")).
		Background(lipgloss.Color("11")).
		Padding(2, 9).
		BorderTop(true).
		BorderLeft(true).
		BorderRight(true).
		BorderBottom(true)

	menu := boxStyle.Render(s)

	// Center the menu both vertically and horizontally
	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		menu)
}
