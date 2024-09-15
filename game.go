package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	env "github.com/muesli/termenv"
)

const sudokuLen = 9

const adminPasswordFileName = "admin_password.txt"

func findAdminPasswordFile() (string, error) {
	// Try current directory first
	if _, err := os.Stat(adminPasswordFileName); err == nil {
		return adminPasswordFileName, nil
	}

	// Try the directory of the executable
	exePath, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exePath)
		filePath := filepath.Join(exeDir, adminPasswordFileName)
		if _, err := os.Stat(filePath); err == nil {
			return filePath, nil
		}
	}

	// Try home directory
	homeDir, err := os.UserHomeDir()
	if err == nil {
		filePath := filepath.Join(homeDir, adminPasswordFileName)
		if _, err := os.Stat(filePath); err == nil {
			return filePath, nil
		}
	}

	return "", fmt.Errorf("admin password file not found")
}

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
	AdminPasswordEntry
	AdminLeaderboardEdit
)

type GameModel struct {
	board                    [sudokuLen][sudokuLen]int
	solution                 [sudokuLen][sudokuLen]int
	initialBoard             [sudokuLen][sudokuLen]int
	KeyMap                   KeyMap
	cursor                   coordinate
	cellsLeft                int
	errCoordinates           map[coordinate]bool
	originalErrCoordinates   map[coordinate]bool
	modifiedErrCoordinates   map[coordinate]bool
	remainingErrCoordinates  map[coordinate]bool
	startTime                time.Time
	width, height            int
	difficulty               Difficulty
	Err                      error
	originalBgColor          env.Color
	output                   *env.Output
	state                    GameState
	menuOptions              []string
	selectedOption           int
	elapsedTimeOnWin         time.Duration
	blinkOn                  bool
	leaderboard              *Leaderboard
	playerName               string
	nameEntered              bool
	adminPassword            string
	adminPasswordAttempt     string
	adminMode                bool
	selectedLeaderboardEntry int
	adminModeBuffer          string
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

	var adminPassword string
	passwordFilePath, err := findAdminPasswordFile()
	if err == nil {
		content, err := os.ReadFile(passwordFilePath)
		if err == nil {
			adminPassword = strings.TrimSpace(string(content))
		}
	}

	leaderboard, err := LoadLeaderboardFromFile("sudoku_leaderboard.json")
	if err != nil {
		// Handle error (maybe log it)
		leaderboard = NewLeaderboard()
	}

	return &GameModel{
		board:                    board,
		solution:                 solution,
		initialBoard:             initialBoard,
		KeyMap:                   Keys,
		cellsLeft:                cellsLeft,
		errCoordinates:           make(map[coordinate]bool),
		startTime:                time.Now(),
		width:                    width,
		height:                   height,
		difficulty:               difficulty,
		originalBgColor:          env.BackgroundColor(),
		output:                   env.DefaultOutput(),
		state:                    Playing,
		menuOptions:              []string{"Resume Game", "New Game", "View Leaderboard", "Quit"},
		selectedOption:           0,
		leaderboard:              leaderboard,
		playerName:               "",
		nameEntered:              false,
		adminPassword:            strings.TrimSpace(string(adminPassword)),
		adminPasswordAttempt:     "",
		adminMode:                false,
		selectedLeaderboardEntry: 0,
		adminModeBuffer:          "",
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
					if len(m.playerName) < 20 {
						m.playerName += string(msg.Runes)
					}
				}
			} else {
				if msg.Type == tea.KeyEsc || msg.String() == "q" {
					return NewMenuModel(m.width, m.height), nil
				}
			}

		case m.state == ViewingLeaderboard:
			if !m.adminMode {
				if msg.String() == "a" || (len(msg.Runes) > 0 && msg.Runes[0] == 'a') || msg.Type == tea.KeyRunes && string(msg.Runes) == "a" {
					if m.adminPassword == "" {
					} else {
						m.state = AdminPasswordEntry
					}
					return m, nil
				}
				if msg.Type == tea.KeyEsc || msg.String() == "q" {
					if m.nameEntered {
						return NewMenuModel(m.width, m.height), nil
					} else {
						m.state = Playing
					}
					return m, nil
				}
			} else {

				switch msg.String() {
				case "up", "k":
					m.selectedLeaderboardEntry = max(0, m.selectedLeaderboardEntry-1)
				case "down", "j":
					m.selectedLeaderboardEntry = min(len(m.leaderboard.Entries)-1, m.selectedLeaderboardEntry+1)
				case "d":
					m.leaderboard.DeleteEntry(m.selectedLeaderboardEntry)
					m.leaderboard.SaveToFile("sudoku_leaderboard.json")
					m.selectedLeaderboardEntry = max(0, m.selectedLeaderboardEntry-1)
				case "q", "esc":
					m.adminMode = false
					m.selectedLeaderboardEntry = 0
				}
			}

		case m.state == AdminPasswordEntry:
			switch msg.Type {
			case tea.KeyEnter:
				if strings.TrimSpace(m.adminPasswordAttempt) == strings.TrimSpace(m.adminPassword) {
					m.adminMode = true
					m.state = ViewingLeaderboard
					m.adminPasswordAttempt = ""
					fmt.Println("Admin mode activated.")
				} else {
					m.adminPasswordAttempt = ""
					fmt.Println("Incorrect password. Please try again.")
				}
				return m, nil
			case tea.KeyBackspace:
				if len(m.adminPasswordAttempt) > 0 {
					m.adminPasswordAttempt = m.adminPasswordAttempt[:len(m.adminPasswordAttempt)-1]
				}
			case tea.KeyRunes:
				m.adminPasswordAttempt += string(msg.Runes)
			case tea.KeyEsc:
				m.state = ViewingLeaderboard
				m.adminPasswordAttempt = ""
				return m, nil
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

		case key.Matches(msg, m.KeyMap.Number):
			if m.state == Playing || m.state == NeedsCorrection {
				m.set(m.cursor.row, m.cursor.col, int(msg.String()[0]-'0'))
			}

		case key.Matches(msg, m.KeyMap.Clear):
			if m.initialBoard[m.cursor.row][m.cursor.col] == 0 {
				m.clear(m.cursor.row, m.cursor.col)
			}
			// If after clearing, the board is full, check for correctness
			if m.cellsLeft == 0 {
				checkMsg := m.check()()
				return m.Update(checkMsg)
			}
			if key.Matches(msg, m.KeyMap.ViewLeaderboard) {
				m.state = ViewingLeaderboard
				m.nameEntered = false // Reset this flag when viewing leaderboard from game
				return m, nil
			}
		case key.Matches(msg, m.KeyMap.ClearAll):
			m.clearAllCells()

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
	var content string
	switch m.state {
	case InMenu:
		content = m.renderMenu()
	case Won:
		content = m.renderWinScreen()
	case ViewingLeaderboard:
		content = m.renderLeaderboard()
	case AdminPasswordEntry:
		content = m.renderAdminPasswordEntry()
	default:
		content = m.renderGame()
	}

	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		content)
}

var selectedMarkerStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("205"))

func (m GameModel) renderMenu() string {
	var s strings.Builder
	s.WriteString("Menu\n\n")
	for i, option := range m.menuOptions {
		if i == m.selectedOption {
			// Apply the style to the '>' character
			s.WriteString(selectedMarkerStyle.Render("> "))
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
		line := fmt.Sprintf("%-4d %-20s %-10s %-10s",
			i+1,
			truncateString(entry.Name, 20),
			formattedTime,
			formattedDate,
		)
		if m.adminMode && i == m.selectedLeaderboardEntry {
			line = "> " + line
		} else {
			line = "  " + line
		}
		s.WriteString(line + "\n")
	}

	if m.adminMode {
		s.WriteString("\nAdmin Mode: Use up/down to select, 'd' to delete, 'q' to exit admin mode")
	} else {
		s.WriteString("\nPress 'a' for admin mode, 'q' or 'esc' to return to menu")
	}

	return s.String()
}

func (m GameModel) renderAdminPasswordEntry() string {
	prompt := "Enter admin password: "
	maskedPassword := strings.Repeat("*", len(m.adminPasswordAttempt))

	message := fmt.Sprintf("%s%s\n\nPress Enter to submit or Esc to cancel", prompt, maskedPassword)

	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		message)
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
	var boardView strings.Builder

	for i := 0; i < sudokuLen; i++ {
		var row strings.Builder
		for j := 0; j < sudokuLen; j++ {
			value := m.board[i][j]
			cellValue := " "
			if value != 0 {
				cellValue = fmt.Sprintf("%d", value)
			}

			isInitial := m.initialBoard[i][j] != 0
			isCursor := m.cursor.row == i && m.cursor.col == j
			coord := coordinate{i, j}

			// Determine if the cell is an error cell based on remainingErrCoordinates
			isError := m.remainingErrCoordinates[coord]

			// Format the cell
			cellStr := formatCell(isError, isCursor, !isInitial, i, j, cellValue)
			row.WriteString(cellStr)
		}
		// Format the row (add separators if needed)
		rowStr := formatRow(i, row.String())
		boardView.WriteString(rowStr + "\n")
	}
	return boardView.String()
}

func (m GameModel) renderInfo() string {
	// Style definitions
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("15")).
		Bold(true).
		Padding(0, 1)

	infoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("15")).
		Padding(0, 1)

	controlsStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("33")).
		Italic(true).
		Padding(0, 1)

	// Calculate elapsed time
	var elapsedTime time.Duration
	if m.state == Won {
		elapsedTime = m.elapsedTimeOnWin
	} else {
		elapsedTime = time.Since(m.startTime).Round(time.Second)
	}

	header := headerStyle.Render(fmt.Sprintf("Sudoku - %s", m.difficulty))

	gameInfo := infoStyle.Render(fmt.Sprintf("Cells left: %d\n"+
		"Elapsed time: %02d:%02d",
		m.cellsLeft,
		int(elapsedTime.Minutes()), int(elapsedTime.Seconds())%60))

	controls := controlsStyle.Render("q/esc: quit • m: menu • b: leaderboard • ⌫ clear cell • C: clear all\n" +
		"Use arrow keys to move, numbers to fill")

	// Combine sections
	info := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		gameInfo,
		"\n",
		controls,
	)

	return lipgloss.NewStyle().
		Padding(1).
		Render(info)
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

		coord := coordinate{row, col}

		if m.state == NeedsCorrection {
			// If the cell was previously incorrect and hasn't been modified yet
			if m.originalErrCoordinates[coord] && m.remainingErrCoordinates[coord] {
				// Remove the cell from remaining incorrect cells
				delete(m.remainingErrCoordinates, coord)
			}

			// Remove the cell from errCoordinates
			delete(m.errCoordinates, coord)

			// Check if all incorrect cells have been modified
			if len(m.remainingErrCoordinates) == 0 {
				// All incorrect cells have been modified; recheck the board
				m.recheckBoard()
			}
		}

		// Update the game state
		m.updateGameState()
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

func (m *GameModel) recheckBoard() {
	m.updateErrCoordinates()
	m.updateGameState()
}

func (m *GameModel) set(row, col, value int) {
	if m.initialBoard[row][col] == 0 {
		previousValue := m.board[row][col]
		m.board[row][col] = value

		// Update cellsLeft based on the change
		if previousValue == 0 && value != 0 {
			m.cellsLeft--
		} else if previousValue != 0 && value == 0 {
			m.cellsLeft++
		}

		if m.cellsLeft == 0 {
			coord := coordinate{row, col}
			if m.state == NeedsCorrection {
				// Remove the cell from remainingErrCoordinates if it was originally incorrect
				if m.originalErrCoordinates[coord] && m.remainingErrCoordinates[coord] {
					delete(m.remainingErrCoordinates, coord)
				}

				// Recheck the board if all incorrect cells have been modified
				if len(m.remainingErrCoordinates) == 0 {
					// All incorrect cells have been modified; recheck the board
					m.recheckBoard()
				}
			} else {
				// First time the board is filled, check for correctness
				m.updateErrCoordinates()
				m.updateGameState()
			}
		} else {
			// Board is not full, reset error tracking and state
			m.errCoordinates = make(map[coordinate]bool)
			m.originalErrCoordinates = make(map[coordinate]bool)
			m.remainingErrCoordinates = make(map[coordinate]bool)
			m.state = Playing
		}
	}
}

func (m *GameModel) copyErrCoordinates() map[coordinate]bool {
	copy := make(map[coordinate]bool)
	for coord := range m.errCoordinates {
		copy[coord] = true
	}
	return copy
}

func (m *GameModel) copyCoordinates(src map[coordinate]bool) map[coordinate]bool {
	dst := make(map[coordinate]bool)
	for coord := range src {
		dst[coord] = true
	}
	return dst
}

func (m *GameModel) updateErrCoordinates() {
	m.errCoordinates = make(map[coordinate]bool)
	for i := 0; i < sudokuLen; i++ {
		for j := 0; j < sudokuLen; j++ {
			cellValue := m.board[i][j]
			if cellValue != 0 && cellValue != m.solution[i][j] {
				coord := coordinate{i, j}
				m.errCoordinates[coord] = true
			}
		}
	}

	if m.state != NeedsCorrection {
		// Only initialize originalErrCoordinates the first time
		m.originalErrCoordinates = m.copyCoordinates(m.errCoordinates)
	}
	// Always update remainingErrCoordinates
	m.remainingErrCoordinates = m.copyCoordinates(m.errCoordinates)
}

func (m *GameModel) updateGameState() {
	if m.cellsLeft == 0 {
		if len(m.errCoordinates) == 0 {
			m.state = Won
			m.elapsedTimeOnWin = time.Since(m.startTime)
		} else {
			m.state = NeedsCorrection
		}
	} else {
		// Board is incomplete
		if len(m.errCoordinates) == 0 {
			m.state = Playing
		} else {
			m.state = NeedsCorrection
		}
	}
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
			fmt.Println("Error saving leaderboard:", err)
		}
	}
	//m.nameEntered = false
}

func formatDuration(d time.Duration) string {
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	return fmt.Sprintf("%dm %ds", m, s)
}

func truncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	return s[:maxLength-3] + "..."
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (m GameModel) getPasswordFileStatus() string {
	passwordFilePath, err := findAdminPasswordFile()
	if err != nil {
		return fmt.Sprintf("Admin password file not found: %v", err)
	}

	fileInfo, err := os.Stat(passwordFilePath)
	if err != nil {
		return fmt.Sprintf("Error checking file at %s: %v", passwordFilePath, err)
	}

	content, err := os.ReadFile(passwordFilePath)
	if err != nil {
		return fmt.Sprintf("Error reading file at %s: %v", passwordFilePath, err)
	}
	if len(strings.TrimSpace(string(content))) == 0 {
		return fmt.Sprintf("File at %s is empty", passwordFilePath)
	}
	return fmt.Sprintf("File exists at %s, size: %d bytes, last modified: %s",
		passwordFilePath, fileInfo.Size(), fileInfo.ModTime())
}

func (m *GameModel) clearAllCells() {
	for i := 0; i < sudokuLen; i++ {
		for j := 0; j < sudokuLen; j++ {
			if m.initialBoard[i][j] == 0 {
				if m.board[i][j] != 0 {
					// Only increment cellsLeft if the cell was filled
					m.board[i][j] = 0
					m.cellsLeft++
				}
				// If the cell is already empty, do nothing
			}
		}
	}
	// Reset error tracking and game state
	m.errCoordinates = make(map[coordinate]bool)
	m.originalErrCoordinates = make(map[coordinate]bool)
	m.remainingErrCoordinates = make(map[coordinate]bool)
	m.state = Playing
}

type GameWon struct{}
type GameNeedsCorrection struct{}
type ForceRender struct{}
