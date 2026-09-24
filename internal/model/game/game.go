// internal/model/game/game.go (добавляем методы)
package game

import (
	"fmt"
	"sync"
	"time"
)

type Game struct {
	id           int
	size         int
	moves        []Move
	isFinished   bool
	winner       int
	currentTurn  int
	mu           sync.RWMutex
	startTime    time.Time
	lastMoveTime time.Time
}

func NewGame(id, size int) *Game {
	now := time.Now()
	return &Game{
		id:           id,
		size:         size,
		moves:        make([]Move, 0),
		isFinished:   false,
		winner:       0,
		currentTurn:  0,
		startTime:    now,
		lastMoveTime: now,
	}
}

// ... остальные методы ...

func (g *Game) GetStartTime() time.Time {
	return g.startTime
}

func (g *Game) GetLastMoveTime() time.Time {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.lastMoveTime
}

func (g *Game) GetGameDuration() time.Duration {
	return time.Since(g.startTime)
}