package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	bm "github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
)

const (
	host = "0.0.0.0"
	port = "22"
)

var (
	highlightColor  = lipgloss.Color("205")
	normalColor     = lipgloss.Color("15") // White color for pre-filled numbers
	userInputColor  = lipgloss.Color("1")  // Red color for user-added numbers
	emptyColor      = lipgloss.Color("8")
	boardBackground = lipgloss.Color("236") // Dark gray background
)

type model struct {
	board         [9][9]int
	initialBoard  [9][9]int // Store the initial board to differentiate between pre-filled and user-added numbers
	cursor        [2]int
	userInput     string
	err           string
	completed     bool
	width         int
	height        int
	cursorVisible bool
}

func main() {
	s, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(host, port)),
		wish.WithHostKeyPath(".ssh/term_info_ed25519"),
		wish.WithMiddleware(
			bm.Middleware(teaHandler),
			activeterm.Middleware(),
			logging.Middleware(),
		),
	)
	if err != nil {
		log.Error("could not start server", "error", err)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	log.Info("Starting SSH server", "host", host, "port", port)

	go func() {
		if err = s.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			log.Error("could not start server", "error", err)
			done <- nil
		}
	}()

	<-done
	log.Info("Stopping SSH server")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer func() { cancel() }()
	if err := s.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		log.Error("could not stop server", "error", err)
	}
}

func teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	pty, _, _ := s.Pty()
	initialBoard := generateSudoku()
	m := model{
		board:         initialBoard,
		initialBoard:  initialBoard,
		cursor:        [2]int{0, 0},
		width:         pty.Window.Width,
		height:        pty.Window.Height,
		cursorVisible: true,
	}
	return m, []tea.ProgramOption{tea.WithAltScreen()}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		blinkCursor(time.Second),
	)
}

func blinkCursor(duration time.Duration) tea.Cmd {
	return tea.Tick(duration, func(time.Time) tea.Msg {
		return cursorBlinkMsg{}
	})
}

type cursorBlinkMsg struct{}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case cursorBlinkMsg:
		m.cursorVisible = !m.cursorVisible
		return m, blinkCursor(time.Second)
	}
	return m, nil
}

func (m model) View() string {
	// Title
	s := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Render("Sudoku") + "\n\n"

	// Board rendering with a background applied
	boardWithBackground := lipgloss.NewStyle().
		Background(boardBackground).
		Padding(1).
		Render(m.renderBoard())

	s += boardWithBackground + "\n"

	// Instructions
	s += lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Render("Use arrow keys to move, numbers to fill, 'q' to quit")

	// Error messages
	if m.err != "" {
		s += "\n" + lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")).
			Render(m.err)
	}

	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Render(s)
}

func (m model) renderBoard() string {
	var boardView string
	for i, row := range m.board {
		for j, cell := range row {
			style := lipgloss.NewStyle().
				Width(3).
				Align(lipgloss.Center)

			// Highlight cursor position
			if m.cursor[0] == i && m.cursor[1] == j {
				if m.cursorVisible {
					style = style.Background(highlightColor)
				} else {
					style = style.Foreground(highlightColor)
				}
			}

			// Render the cell
			if cell == 0 {
				boardView += style.Render("·")
			} else if m.initialBoard[i][j] != 0 {
				// Pre-filled numbers in white
				boardView += style.Foreground(normalColor).Render(fmt.Sprintf("%d", cell))
			} else {
				// User-added numbers in red
				boardView += style.Foreground(userInputColor).Render(fmt.Sprintf("%d", cell))
			}

			// Add spacing between 3x3 grids
			if j == 2 || j == 5 {
				boardView += " "
			}
		}
		boardView += "\n"
		if i == 2 || i == 5 {
			boardView += "\n"
		}
	}
	return boardView
}

func (m *model) moveCursor(direction string) {
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

func (m *model) updateBoard() {
	row, col := m.cursor[0], m.cursor[1]
	if m.initialBoard[row][col] == 0 { // Only allow updates to cells that were initially empty
		num, _ := strconv.Atoi(m.userInput)
		if isValid(m.board, row, col, num) {
			m.board[row][col] = num
			m.err = ""
			if isBoardFull(m.board) {
				m.completed = true
			}
		} else {
			m.err = "Invalid move"
		}
	} else {
		m.err = "Cell is not empty"
	}
}

func generateSudoku() [9][9]int {
	var board [9][9]int
	for i := 0; i < 20; i++ {
		row, col := rand.Intn(9), rand.Intn(9)
		num := rand.Intn(9) + 1
		if isValid(board, row, col, num) {
			board[row][col] = num
		}
	}
	return board
}

func isValid(board [9][9]int, row, col, num int) bool {
	for i := 0; i < 9; i++ {
		if board[row][i] == num || board[i][col] == num {
			return false
		}
	}
	startRow, startCol := row-row%3, col-col%3
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if board[i+startRow][j+startCol] == num {
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

