package main

import (
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func generateSudoku(difficulty Difficulty) ([9][9]int, [9][9]int) {
	var board, solution [9][9]int
	fillBoard(&solution)
	board = solution
	removeCells(&board, difficulty)
	return board, solution
}

func fillBoard(board *[9][9]int) bool {
	for i := 0; i < 9; i++ {
		for j := 0; j < 9; j++ {
			if board[i][j] == 0 {
				nums := rand.Perm(9)
				for _, n := range nums {
					num := n + 1
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

func removeCells(board *[9][9]int, difficulty Difficulty) {
	cellsToRemove := 0
	switch difficulty {
	case Easy:
		cellsToRemove = 30
	case Medium:
		cellsToRemove = 40
	case Hard:
		cellsToRemove = 50
	}

	attempts := cellsToRemove + 20 // Extra attempts to ensure unique solution
	for cellsToRemove > 0 && attempts > 0 {
		row := rand.Intn(9)
		col := rand.Intn(9)
		if board[row][col] != 0 {
			backup := board[row][col]
			board[row][col] = 0

			tempBoard := *board
			solutions := countSolutions(tempBoard)

			if solutions != 1 {
				board[row][col] = backup
				attempts--
			} else {
				cellsToRemove--
			}
		}
	}
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

func countSolutions(board [9][9]int) int {
	count := 0
	solve(&board, &count)
	return count
}

func solve(board *[9][9]int, count *int) {
	if *count > 1 {
		return
	}
	for i := 0; i < 9; i++ {
		for j := 0; j < 9; j++ {
			if board[i][j] == 0 {
				nums := rand.Perm(9)
				for _, n := range nums {
					num := n + 1
					if isValid(*board, i, j, num) {
						board[i][j] = num
						solve(board, count)
						board[i][j] = 0
					}
				}
				return
			}
		}
	}
	*count++
}

