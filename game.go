package main

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type GameModel struct {
	board        [9][9]int
	initialBoard [9][9]int
	solution     [9][9]int
	cursor       [2]int
	userInput    string
	err          string
	message      string
	completed    bool
	width        int
	height       int
	difficulty   Difficulty
}

func NewGameModel(width, height int, difficulty Difficulty) *GameModel {
	board, solution := generateSudoku(difficulty)
	return &GameModel{
		board:        board,
		initialBoard: board,
		solution:     solution,
		cursor:       [2]int{0, 0},
		width:        width,
		height:       height,
		difficulty:   difficulty,
	}
}

func (m GameModel) Init() tea.Cmd {
	return nil
}

func (m GameModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "down", "left", "right":
			m.moveCursor(msg.String())
		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			m.userInput = msg.String()
			m.updateBoard()
		case "m":
			return NewMenuModel(m.width, m.height), nil
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func (m GameModel) View() string {
	var s strings.Builder

	// Calculate the length of the longest line in the board
	boardLines := strings.Split(m.renderBoard(), "\n")
	boardWidth := len(boardLines[0])
	boardHeight := len(boardLines)

	// Calculate horizontal and vertical centering
	horizontalPadding := (m.width - boardWidth) / 2
	verticalPadding := (m.height - (boardHeight + 4)) / 2 // 4 lines for instructions and messages

	// Add vertical padding
	for i := 0; i < verticalPadding; i++ {
		s.WriteString("\n")
	}

	// Render the board with horizontal padding
	for _, line := range boardLines {
		s.WriteString(strings.Repeat(" ", horizontalPadding))
		s.WriteString(line + "\n")
	}

	// Add controls info with padding
	s.WriteString(strings.Repeat(" ", horizontalPadding))
	s.WriteString(fmt.Sprintf("%sSudoku - %s%s\n\n", boldWhite, m.difficulty.String(), reset))
	s.WriteString(strings.Repeat(" ", horizontalPadding))
	s.WriteString(fmt.Sprintf("%sUse arrow keys to move, numbers to fill, 'q' to quit, 'm' for menu%s\n", white, reset))

	// Add error or success message if applicable with padding
	if m.err != "" {
		s.WriteString(strings.Repeat(" ", horizontalPadding))
		s.WriteString(fmt.Sprintf("%s%s%s\n", red, m.err, reset))
	}
	if m.message != "" {
		s.WriteString(strings.Repeat(" ", horizontalPadding))
		s.WriteString(fmt.Sprintf("%s%s%s\n", green, m.message, reset))
	}

	return s.String()
}

func (m *GameModel) moveCursor(direction string) {
	switch direction {
	case "up":
		m.cursor[0] = (m.cursor[0] - 1 + 9) % 9
	case "down":
		m.cursor[0] = (m.cursor[0] + 1) % 9
	case "left":
		m.cursor[1] = (m.cursor[1] - 1 + 9) % 9
	case "right":
		m.cursor[1] = (m.cursor[1] + 1) % 9
	}
}

func (m GameModel) renderBoard() string {
	var board strings.Builder
	for i, row := range m.board {
		board.WriteString("   ") // Add some indentation for spacing

		// Add vertical dividers between 3x3 blocks
		for j, cell := range row {
			// Highlight the cell where the cursor is located
			if m.cursor[0] == i && m.cursor[1] == j {
				board.WriteString(bold + "\033[48;5;220m" + "\033[30m") // Background Yellow, Foreground Black
			}

			// Fill in the board values
			if cell == 0 {
				board.WriteString(" . ") // Empty cell
			} else if m.initialBoard[i][j] != 0 {
				board.WriteString(white + fmt.Sprintf(" %d ", cell) + reset)
			} else {
				board.WriteString(red + fmt.Sprintf(" %d ", cell) + reset)
			}

			// Reset formatting after rendering cursor
			if m.cursor[0] == i && m.cursor[1] == j {
				board.WriteString(reset)
			}

			// Add vertical dividers between the 3x3 blocks
			if j == 2 || j == 5 {
				board.WriteString("|")
			}
		}
		board.WriteString("\n")

		// Add horizontal dividers between the 3x3 blocks
		if i == 2 || i == 5 {
			board.WriteString("   --------+---------+--------\n")
		}
	}
	return board.String()
}

func (m *GameModel) updateBoard() {
	row, col := m.cursor[0], m.cursor[1]
	if m.initialBoard[row][col] == 0 {
		num, _ := strconv.Atoi(m.userInput)
		if isValid(m.board, row, col, num) {
			m.board[row][col] = num
			m.err = ""
			if isBoardFull(m.board) {
				if m.checkSolution() {
					m.message = "Congratulations! You have solved the Sudoku correctly!"
				} else {
					m.message = "The board is full, but the solution is incorrect. Keep trying!"
				}
			}
		} else {
			m.err = "Invalid move"
		}
	} else {
		m.err = "Cell is not empty"
	}
}

func (m *GameModel) checkSolution() bool {
	for i := 0; i < 9; i++ {
		for j := 0; j < 9; j++ {
			if m.board[i][j] != m.solution[i][j] {
				return false
			}
		}
	}
	return true
}

func isBoardFull(board [9][9]int) bool {
	for i := 0; i < 9; i++ {
		for j := 0; j < 9; j++ {
			if board[i][j] == 0 {
				return false
			}
		}
	}
	return true
}

