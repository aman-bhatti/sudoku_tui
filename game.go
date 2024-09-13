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
	ViewingLeaderboard
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
	leaderboard      *Leaderboard
	playerName       string
	nameEntered      bool
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

	leaderboard, err := LoadLeaderboardFromFile("sudoku_leaderboard.json")
	if err != nil {
		// Handle error (maybe log it)
		leaderboard = NewLeaderboard()
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
		menuOptions:     []string{"Resume Game", "New Game", "View Leaderboard", "Quit"},
		selectedOption:  0,
		leaderboard:     leaderboard,
		playerName:      "",
		nameEntered:     false,
	}
}

func (m GameModel) Init() tea.Cmd {
	return setBackgroundColor(env.RGBColor("#1e1e1e"))
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

		case m.state == Won:
			if !m.nameEntered {
				switch msg.Type {
				case tea.KeyEnter:
					if m.playerName != "" {
						m.nameEntered = true
						m.SaveScore()
						m.state = ViewingLeaderboard
						return m, nil
					}
				case tea.KeyBackspace:
					if len(m.playerName) > 0 {
						m.playerName = m.playerName[:len(m.playerName)-1]
					}
				case tea.KeyRunes:
					if len(m.playerName) < 20 { // Limit name length
						m.playerName += string(msg.Runes)
					}
				}
			}

		case m.state == ViewingLeaderboard:
			if msg.Type == tea.KeyEsc || msg.String() == "q" {
				return NewMenuModel(m.width, m.height), nil
			}

		case key.Matches(msg, m.KeyMap.Menu):
			m.state = InMenu

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
				cmd := m.set(m.cursor.row, m.cursor.col, int(msg.String()[0]-'0'))
				if m.cellsLeft == 0 {
					return m, tea.Sequence(cmd, func() tea.Msg {
						return m.check()()
					})
				}
				return m, cmd
			}

		case key.Matches(msg, m.KeyMap.Quit):
			return m, tea.Sequence(
				setBackgroundColor(m.originalBgColor),
				tea.Quit,
			)

		case key.Matches(msg, m.KeyMap.ViewLeaderboard):
			if m.state == Playing {
				m.state = ViewingLeaderboard
				return m, nil
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case GameWon:
		m.state = Won
		m.elapsedTimeOnWin = time.Since(m.startTime)

	case GameNeedsCorrection:
		m.state = NeedsCorrection
		return m, tea.Cmd(func() tea.Msg {
			return ForceRender{}
		})

	case ForceRender:
		// Do nothing, just trigger a re-render
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
			return NewMenuModel(m.width, m.height), nil
		case 2: // View Leaderboard
			m.state = ViewingLeaderboard
			return m, nil
		case 3: // Quit
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
	case ViewingLeaderboard:
		return m.renderLeaderboard()
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
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(2, 6).
		BorderBackground(lipgloss.Color("11")).
		BorderForeground(lipgloss.Color("11")).
		Background(lipgloss.Color("11"))

	textStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("0")).
		Bold(true)

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true).
		Align(lipgloss.Center)

	namePrompt := "Enter your name: "
	if m.nameEntered {
		namePrompt = fmt.Sprintf("Name: %s", m.playerName)
	}

	var instructionText string
	if m.nameEntered {
		instructionText = "Press Enter to save score"
	} else {
		instructionText = "Type your name and press Enter"
	}

	winMessage := fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s",
		titleStyle.Bold(true).Render("You Win!!!"),
		textStyle.Render(fmt.Sprintf("Time: %02d:%02d", int(m.elapsedTimeOnWin.Minutes()), int(m.elapsedTimeOnWin.Seconds())%60)),
		textStyle.Render(namePrompt+m.playerName),
		textStyle.Render(instructionText))

	boxedWinMessage := boxStyle.Render(winMessage)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, boxedWinMessage)
}

func (m GameModel) renderLeaderboard() string {
	topScores := m.leaderboard.GetTopScores(m.difficulty, 10)

	var s strings.Builder
	s.WriteString(fmt.Sprintf("Leaderboard - %s\n\n", m.difficulty))
	s.WriteString(fmt.Sprintf("%-4s %-20s %-10s %-10s\n", "Rank", "Name", "Time", "Date"))
	s.WriteString(fmt.Sprintf("%-4s %-20s %-10s %-10s\n", "----", "----", "----", "----"))

	for i, entry := range topScores {
		formattedTime := formatDuration(entry.Time)
		formattedDate := entry.Date.Format("2006-01-02")
		s.WriteString(fmt.Sprintf("%-4d %-20s %-10s %-10s\n",
			i+1,
			truncateString(entry.Name, 20),
			formattedTime,
			formattedDate,
		))
	}

	s.WriteString("\nPress 'q' or 'esc' to return to menu")

	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		s.String())
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
			value := m.board[i][j]
			cellValue := " "
			if value != 0 {
				cellValue = fmt.Sprintf("%d", value)
			}

			isInitial := m.initialBoard[i][j] != 0
			isCursor := m.cursor.row == i && m.cursor.col == j

			row += formatCell(isError, isCursor, !isInitial, i, j, cellValue)
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
	info += "\n• q/esc quit • m menu • b leaderboard\n"
	info += fmt.Sprintf("\nSudoku - %s\n", m.difficulty)
	info += "\nUse arrow keys to move, numbers to fill"

	whiteTextStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("15"))

	return cellsLeftStyle.Render(whiteTextStyle.Render(info))
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

		if incorrectCellsFound {
			return GameNeedsCorrection{}
		}
		return GameWon{}
	}
}

func (m *GameModel) SaveScore() {
	if m.playerName != "" {
		m.leaderboard.AddEntry(m.playerName, m.elapsedTimeOnWin, m.difficulty)
		err := m.leaderboard.SaveToFile("sudoku_leaderboard.json")
		if err != nil {
			// Handle error (maybe log it)
			fmt.Println("Error saving leaderboard:", err)
		}
	}
	m.nameEntered = false
}

// Helper function to format duration as "Xm Ys"
func formatDuration(d time.Duration) string {
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	return fmt.Sprintf("%dm %ds", m, s)
}

// Helper function to truncate string if it's too long
func truncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	return s[:maxLength-3] + "..."
}

type GameWon struct{}
type GameNeedsCorrection struct{}
type ForceRender struct{}

