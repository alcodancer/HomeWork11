// internal/model/game/piece.go
package game

import "fmt"

type Color string

const (
	White Color = "white"
	Black Color = "black"
)

type Piece struct {
	pieceType string
	color     Color
	symbol    rune
}

func NewPiece(pieceType string, color Color, symbol rune) *Piece {
	return &Piece{
		pieceType: pieceType,
		color:     color,
		symbol:    symbol,
	}
}

func (p *Piece) GetType() string {
	return p.pieceType
}

func (p *Piece) GetColor() Color {
	return p.color
}

func (p *Piece) GetSymbol() rune {
	return p.symbol
}

func (p *Piece) SetSymbol(symbol rune) {
	p.symbol = symbol
}

func (p *Piece) String() string {
	return string(p.symbol)
}