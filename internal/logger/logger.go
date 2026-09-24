// internal/logger/logger.go
package logger

import (
	"chess/internal/repository"
	"context"
	"fmt"
	"sync"
	"time"
)

type Logger struct {
	repo            repository.Repository
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	lastPlayerCount int
	lastGameCount   int
	lastMoveCount   int
	lastBoardCount  int
	mu              sync.RWMutex
}

func NewLogger(repo repository.Repository) *Logger {
	ctx, cancel := context.WithCancel(context.Background())
	return &Logger{
		repo:            repo,
		ctx:             ctx,
		cancel:          cancel,
		lastPlayerCount: 0,
		lastGameCount:   0,
		lastMoveCount:   0,
		lastBoardCount:  0,
	}
}

func (l *Logger) Start() {
	l.wg.Add(1)
	go l.run()
}

func (l *Logger) run() {
	defer l.wg.Done()
	
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			l.checkAndLog()
		case <-l.ctx.Done():
			fmt.Println("[Logger] Остановка логгера")
			return
		}
	}
}

func (l *Logger) Stop() {
	fmt.Println("[Logger] Получен сигнал остановки")
	l.cancel()
	l.wg.Wait()
	fmt.Println("[Logger] Логгер остановлен")
}

func (l *Logger) checkAndLog() {
	l.mu.Lock()
	defer l.mu.Unlock()

	players := l.repo.GetPlayers()
	games := l.repo.GetGames()
	moves := l.repo.GetMoves()
	boards := l.repo.GetBoards()

	currentPlayerCount := len(players)
	currentGameCount := len(games)
	currentMoveCount := len(moves)
	currentBoardCount := len(boards)

	now := time.Now().Format("15:04:05.000")

	if currentPlayerCount > l.lastPlayerCount {
		newPlayers := players[l.lastPlayerCount:]
		fmt.Printf("[%s] LOGGER: Добавлены игроки: ", now)
		for _, p := range newPlayers {
			fmt.Printf("%s ", p.GetName())
		}
		fmt.Println()
	}

	if currentGameCount > l.lastGameCount {
		newGames := games[l.lastGameCount:]
		fmt.Printf("[%s] LOGGER: Добавлены игры: ", now)
		for _, g := range newGames {
			fmt.Printf("%s ", g.String())
		}
		fmt.Println()
	}

	if currentMoveCount > l.lastMoveCount {
		newMoves := moves[l.lastMoveCount:]
		fmt.Printf("[%s] LOGGER: Добавлены ходы: ", now)
		for _, m := range newMoves {
			fmt.Printf("%s ", m.String())
		}
		fmt.Println()
	}

	if currentBoardCount > l.lastBoardCount {
		newBoards := boards[l.lastBoardCount:]
		fmt.Printf("[%s] LOGGER: Добавлены доски: ", now)
		for _, b := range newBoards {
			fmt.Printf("Board{id=%d, gameID=%d, size=%d} ", b.GetID(), b.GetGameID(), b.GetSize())
		}
		fmt.Println()
	}

	l.lastPlayerCount = currentPlayerCount
	l.lastGameCount = currentGameCount
	l.lastMoveCount = currentMoveCount
	l.lastBoardCount = currentBoardCount
}