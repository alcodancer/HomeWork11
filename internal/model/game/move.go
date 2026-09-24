// internal/model/game/move.go (добавляем методы)
package game

import (
	"fmt"
	"time"
)

type Move struct {
	fromRow   int
	fromCol   int
	toRow     int
	toCol     int
	piece     *Piece
	playerID  int
	gameID    int
	timestamp time.Time
	duration  time.Duration
}

func NewMove(fromRow, fromCol, toRow, toCol int, piece *Piece, playerID, gameID int) Move {
	return Move{
		fromRow:   fromRow,
		fromCol:   fromCol,
		toRow:     toRow,
		toCol:     toCol,
		piece:     piece,
		playerID:  playerID,
		gameID:    gameID,
		timestamp: time.Now(),
	}
}

// ... остальные методы ...

func (m *Move) SetDuration(duration time.Duration) {
	m.duration = duration
}

func (m *Move) GetDuration() time.Duration {
	return m.duration
}

func (m *Move) GetTimestamp() time.Time {
	return m.timestamp
}