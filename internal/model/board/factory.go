// internal/model/board/factory.go
package board

import (
	"chess/internal/model/game"
)

func SetupInitialBoard(boardID, gameID, boardSize int) *Board {
	board := NewBoard(boardID, gameID, boardSize)

	whiteBackRow := []string{"♖", "♘", "♗", "♔", "♕", "♗", "♘", "♖"}
	whitePawns := []string{"♙", "♙", "♙", "♙", "♙", "♙", "♙", "♙"}

	blackBackRow := []string{"♜", "♞", "♝", "♚", "♛", "♝", "♞", "♜"}
	blackPawns := []string{"♟", "♟", "♟", "♟", "♟", "♟", "♟", "♟"}

	for i := 0; i < 8 && i < boardSize; i++ {
		if i < len(blackPawns) {
			piece := game.NewPiece("pawn", game.Black, []rune(blackPawns[i])[0])
			board.SetPiece(1, i, piece)
		}
		if i < len(blackBackRow) {
			piece := game.NewPiece("rook", game.Black, []rune(blackBackRow[i])[0])
			board.SetPiece(0, i, piece)
		}

		if i < len(whitePawns) {
			piece := game.NewPiece("pawn", game.White, []rune(whitePawns[i])[0])
			board.SetPiece(boardSize-2, i, piece)
		}
		if i < len(whiteBackRow) {
			piece := game.NewPiece("rook", game.White, []rune(whiteBackRow[i])[0])
			board.SetPiece(boardSize-1, i, piece)
		}
	}

	return board
}