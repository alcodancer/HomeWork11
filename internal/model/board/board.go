// internal/model/board/board.go (добавляем методы)
package board

import (
	"fmt"
	"strings"
	"sync"
)

type PieceInterface interface {
	GetSymbol() rune
	String() string
}

type Board struct {
	id     int
	gameID int
	cells  [][]Cell
	size   int
	mu     sync.RWMutex
}

type Cell struct {
	row     int
	col     int
	piece   PieceInterface
	isEmpty bool
}

func NewBoard(id, gameID, size int) *Board {
	cells := make([][]Cell, size)
	for i := 0; i < size; i++ {
		cells[i] = make([]Cell, size)
		for j := 0; j < size; j++ {
			cells[i][j] = Cell{
				row:     i,
				col:     j,
				isEmpty: true,
				piece:   nil,
			}
		}
	}
	return &Board{
		id:     id,
		gameID: gameID,
		cells:  cells,
		size:   size,
	}
}

// ... остальные методы ...

func (b *Board) GetID() int {
	return b.id
}

func (b *Board) GetGameID() int {
	return b.gameID
}

func (b *Board) GetSize() int {
	return b.size
}