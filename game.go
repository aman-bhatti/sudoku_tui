package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	env "github.com/muesli/termenv"
)

const sudokuLen = 9

type coordinate struct {
	row, col int
}

type GameState int

const (
	Playing GameState = iota
	Won
	NeedsCorrection
	InMenu
)

type GameModel struct {
	board            [sudokuLen][sudokuLen]int
	solution         [sudokuLen][sudokuLen]int
	initialBoard     [sudokuLen][sudokuLen]int
	KeyMap           KeyMap
	cursor           coordinate
	cellsLeft        int
	errCoordinates   map[coordinate]bool
	startTime        time.Time
	width, height    int
	difficulty       Difficulty
	Err              error
	originalBgColor  env.Color
	output           *env.Output
	state            GameState
	menuOptions      []string
	selectedOption   int
	elapsedTimeOnWin time.Duration
	blinkOn          bool
}

type setBackgroundColorMsg struct {
	color env.Color
}

func setBackgroundColor(c env.Color) tea.Cmd {
	return func() tea.Msg {
		return setBackgroundColorMsg{color: c}
	}
}

func NewGameModel(width, height int, difficulty Difficulty) *GameModel {
	board, solution := generateSudoku(difficulty)
	cellsLeft := 0
	var initialBoard [sudokuLen][sudokuLen]int
	for i := 0; i < sudokuLen; i++ {
		for j := 0; j < sudokuLen; j++ {
			initialBoard[i][j] = board[i][j]
			if board[i][j] == 0 {
				cellsLeft++
			}
		}
	}

	return &GameModel{
		board:           board,
		solution:        solution,
		initialBoard:    initialBoard,
		KeyMap:          Keys,
		cellsLeft:       cellsLeft,
		errCoordinates:  make(map[coordinate]bool),
		startTime:       time.Now(),
		width:           width,
		height:          height,
		difficulty:      difficulty,
		originalBgColor: env.BackgroundColor(),
		output:          env.DefaultOutput(),
		state:           Playing,
		menuOptions:     []string{"Resume Game", "New Game", "Quit"},
		selectedOption:  0,
	}
}

func (m GameModel) Init() tea.Cmd {
	return setBackgroundColor(env.RGBColor("#1e1e1e")) // Dark background color
}

func (m GameModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case setBackgroundColorMsg:
		m.output.SetBackgroundColor(msg.color)
		return m, nil

	case tea.KeyMsg:
		switch {
		case m.state == InMenu:
			return m.updateMenu(msg)

		case key.Matches(msg, m.KeyMap.Menu):
			m.state = InMenu // Pressing "m" will switch to menu state

		case key.Matches(msg, m.KeyMap.Down):
			m.cursorDown()

		case key.Matches(msg, m.KeyMap.Up):
			m.cursorUp()

		case key.Matches(msg, m.KeyMap.Left):
			m.cursorLeft()

		case key.Matches(msg, m.KeyMap.Right):
			m.cursorRight()

		case key.Matches(msg, m.KeyMap.Clear):
			if m.initialBoard[m.cursor.row][m.cursor.col] == 0 {
				m.clear(m.cursor.row, m.cursor.col)
			}

		case key.Matches(msg, m.KeyMap.Number):
			if m.state == Playing {
				return m, m.set(m.cursor.row, m.cursor.col, int(msg.String()[0]-'0'))
			}

		case key.Matches(msg, m.KeyMap.Quit):
			return m, tea.Sequence(
				setBackgroundColor(m.originalBgColor),
				tea.Quit,
			)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case GameWon:
		m.state = Won
		m.elapsedTimeOnWin = time.Since(m.startTime) // Track elapsed time on winning

	case GameNeedsCorrection:
		m.state = NeedsCorrection
	}

	return m, nil
}
func (m GameModel) updateMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.KeyMap.Up):
		m.selectedOption = (m.selectedOption - 1 + len(m.menuOptions)) % len(m.menuOptions)
	case key.Matches(msg, m.KeyMap.Down):
		m.selectedOption = (m.selectedOption + 1) % len(m.menuOptions)
	case key.Matches(msg, m.KeyMap.Number), key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):

		switch m.selectedOption {
		case 0: // Resume Game
			m.state = Playing
			return m, nil
		case 1: // New Game
			// Instead of quitting, transition back to the difficulty selection menu
			return NewMenuModel(m.width, m.height), nil
		case 2: // Quit
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m GameModel) View() string {
	switch m.state {
	case InMenu:
		return m.renderMenu()
	case Won:
		return m.renderWinScreen()
	default:
		return m.renderGame()
	}
}

func (m GameModel) renderMenu() string {
	var s strings.Builder
	s.WriteString("Menu\n\n")
	for i, option := range m.menuOptions {
		if i == m.selectedOption {
			s.WriteString("> ")
		} else {
			s.WriteString("  ")
		}
		s.WriteString(option + "\n")
	}
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, s.String())
}

func (m GameModel) renderWinScreen() string {
	// Define styles for the box and text
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2).
		BorderForeground(lipgloss.Color("#FFD700")) // Gold border

	textStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00FF00")). // Green text for the time
		Bold(true)

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF4500")). // Orange text for the title
		Bold(true).
		Align(lipgloss.Center)

	// Win message content
	winMessage := fmt.Sprintf("%s\n\n%s\n\n%s",
		titleStyle.Render("You Win!!!"),
		textStyle.Render(fmt.Sprintf("Time: %02d:%02d", int(m.elapsedTimeOnWin.Minutes()), int(m.elapsedTimeOnWin.Seconds())%60)),
		"Press 'q' to quit or 'm' for menu")

	// Place the win message inside the styled box
	boxedWinMessage := boxStyle.Render(winMessage)

	// Center the box on the screen
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, boxedWinMessage)
}

func (m GameModel) renderGame() string {
	boardView := m.renderBoard()
	infoView := m.renderInfo()

	var statusView string
	switch m.state {
	case NeedsCorrection:
		statusView = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Render("The solution is incorrect. Please check the highlighted cells and try again.")
	}

	mainView := lipgloss.JoinVertical(lipgloss.Center, boardView, infoView, statusView)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, mainView)
}

func (m GameModel) renderBoard() string {
	var boardView string

	for i := 0; i < sudokuLen; i++ {
		row := ""
		for j := 0; j < sudokuLen; j++ {
			isError := m.errCoordinates[coordinate{i, j}]

			// Get the value of the current cell
			value := m.board[i][j]
			cellValue := " "
			if value != 0 {
				cellValue = fmt.Sprintf("%d", value)
			}

			isInitial := m.initialBoard[i][j] != 0

			// Apply custom cursor style if this is the cursor position
			if m.cursor.row == i && m.cursor.col == j {
				// Apply bold yellow background and black foreground
				row += fmt.Sprintf("\033[1m\033[48;5;220m\033[30m%s\033[0m", cellValue) // Reset colors after rendering the cell
			} else {
				// Use the regular format for other cells
				row += formatCell(isError, m.cursor.row == i && m.cursor.col == j, !isInitial, i, j, cellValue)
			}
		}
		boardView += formatRow(i, row) + "\n"
	}
	return boardView
}

func (m GameModel) renderInfo() string {
	var elapsedTime time.Duration
	if m.state == Won {
		elapsedTime = m.elapsedTimeOnWin
	} else {
		elapsedTime = time.Since(m.startTime).Round(time.Second)
	}

	info := fmt.Sprintf("Cells left: %d\n", m.cellsLeft)
	info += fmt.Sprintf("Elapsed time: %02d:%02d\n", int(elapsedTime.Minutes()), int(elapsedTime.Seconds())%60)
	info += "\n? toggle help • q/esc quit • m menu • c clear all\n"
	info += fmt.Sprintf("\nSudoku - %s\n", m.difficulty)
	info += "\nUse arrow keys to move, numbers to fill"
	return cellsLeftStyle.Render(info)
}

func (m *GameModel) cursorDown() {
	m.cursor.row = (m.cursor.row + 1) % sudokuLen
}

func (m *GameModel) cursorUp() {
	m.cursor.row = (m.cursor.row - 1 + sudokuLen) % sudokuLen
}

func (m *GameModel) cursorLeft() {
	m.cursor.col = (m.cursor.col - 1 + sudokuLen) % sudokuLen
}

func (m *GameModel) cursorRight() {
	m.cursor.col = (m.cursor.col + 1) % sudokuLen
}

func (m *GameModel) clear(row, col int) {
	if m.board[row][col] != 0 && m.initialBoard[row][col] == 0 {
		m.board[row][col] = 0
		m.cellsLeft++
		delete(m.errCoordinates, coordinate{row, col})
		m.state = Playing
	}
}

func (m *GameModel) clearAllUserInput() {
	for i := 0; i < sudokuLen; i++ {
		for j := 0; j < sudokuLen; j++ {
			if m.initialBoard[i][j] == 0 {
				m.board[i][j] = 0
				m.cellsLeft++
			}
		}
	}
	m.errCoordinates = make(map[coordinate]bool)
	m.state = Playing
}

func (m *GameModel) set(row, col, value int) tea.Cmd {
	if m.board[row][col] == 0 && m.initialBoard[row][col] == 0 {
		m.board[row][col] = value
		m.cellsLeft--

		// Only check if the grid is full
		if m.cellsLeft == 0 {
			return m.check()
		}
	}
	return nil
}

func (m *GameModel) isValidMove(row, col, value int) bool {
	// Check row
	for i := 0; i < sudokuLen; i++ {
		if m.board[row][i] == value {
			return false
		}
	}

	// Check column
	for i := 0; i < sudokuLen; i++ {
		if m.board[i][col] == value {
			return false
		}
	}

	// Check 3x3 box
	startRow, startCol := 3*(row/3), 3*(col/3)
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if m.board[i+startRow][j+startCol] == value {
				return false
			}
		}
	}

	return true
}

func (m *GameModel) check() tea.Cmd {
	return func() tea.Msg {
		m.errCoordinates = make(map[coordinate]bool)
		incorrectCellsFound := false

		// Iterate over the entire board
		for i := 0; i < sudokuLen; i++ {
			for j := 0; j < sudokuLen; j++ {
				if m.board[i][j] != m.solution[i][j] {
					m.errCoordinates[coordinate{i, j}] = true
					incorrectCellsFound = true
				}
			}
		}

		// Debugging: Check if the error coordinates are populated
		fmt.Println("Error Coordinates:", m.errCoordinates)

		// If incorrect cells were found, return the NeedsCorrection state
		if incorrectCellsFound {
			return GameNeedsCorrection{}
		}
		return GameWon{}
	}
}

type GameWon struct{}
type GameNeedsCorrection struct{}
