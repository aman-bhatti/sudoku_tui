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

// ANSI color codes
const (
	reset     = "\033[0m"
	red       = "\033[31m"
	green     = "\033[32m"
	yellow    = "\033[33m"
	blue      = "\033[34m"
	magenta   = "\033[35m"
	cyan      = "\033[36m"
	white     = "\033[37m"
	boldWhite = "\033[1;37m"
)

type model struct {
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
}

func main() {
	rand.Seed(time.Now().UnixNano())

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
	board, solution := generateSudoku()
	m := model{
		board:        board,
		initialBoard: board,
		solution:     solution,
		cursor:       [2]int{0, 0},
		width:        pty.Window.Width,
		height:       pty.Window.Height,
	}
	return m, []tea.ProgramOption{tea.WithAltScreen()}
}

func (m model) Init() tea.Cmd {
	return tea.EnterAltScreen
}

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
	}
	return m, nil
}

func (m model) View() string {
	s := boldWhite + "Sudoku" + reset + "\n\n"

	s += m.renderBoard() + "\n"

	s += white + "Use arrow keys to move, numbers to fill, 'q' to quit" + reset

	if m.err != "" {
		s += "\n" + red + m.err + reset
	}

	if m.message != "" {
		s += "\n" + green + m.message + reset
	}

	return s
}

func (m model) renderBoard() string {
	var boardView string
	for i, row := range m.board {
		for j, cell := range row {
			if m.cursor[0] == i && m.cursor[1] == j {
				boardView += magenta + "[" + reset
			} else {
				boardView += " "
			}

			if cell == 0 {
				boardView += cyan + "·" + reset
			} else if m.initialBoard[i][j] != 0 {
				boardView += white + fmt.Sprintf("%d", cell) + reset
			} else {
				boardView += red + fmt.Sprintf("%d", cell) + reset
			}

			if m.cursor[0] == i && m.cursor[1] == j {
				boardView += magenta + "]" + reset
			} else {
				boardView += " "
			}

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

func (m *model) checkSolution() bool {
	for i := 0; i < 9; i++ {
		for j := 0; j < 9; j++ {
			if m.board[i][j] != m.solution[i][j] {
				return false
			}
		}
	}
	return true
}

func generateSudoku() ([9][9]int, [9][9]int) {
	var board, solution [9][9]int
	fillBoard(&solution)
	board = solution
	removeCells(&board)
	return board, solution
}

func fillBoard(board *[9][9]int) bool {
	for i := 0; i < 9; i++ {
		for j := 0; j < 9; j++ {
			if board[i][j] == 0 {
				for num := 1; num <= 9; num++ {
					if isValid(*board, i, j, num) {
						board[i][j] = num
						if fillBoard(board) {
							return true
						}
						board[i][j] = 0
					}
				}
				return false
			}
		}
	}
	return true
}

func removeCells(board *[9][9]int) {
	cellsToRemove := 40 // Adjust this number to control difficulty
	for cellsToRemove > 0 {
		row := rand.Intn(9)
		col := rand.Intn(9)
		if board[row][col] != 0 {
			board[row][col] = 0
			cellsToRemove--
		}
	}
}

func isValid(board [9][9]int, row, col, num int) bool {
	// Check row
	for i := 0; i < 9; i++ {
		if board[row][i] == num {
			return false
		}
	}

	// Check column
	for i := 0; i < 9; i++ {
		if board[i][col] == num {
			return false
		}
	}

	// Check 3x3 sub-box
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
