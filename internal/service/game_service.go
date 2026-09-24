// internal/service/game_service.go
package service

import (
	"chess/internal/model/board"
	"chess/internal/model/game"
	"chess/internal/model/player"
	"chess/internal/repository"
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

type MoveRequest struct {
	GameID     int
	FromRow    int
	FromCol    int
	ToRow      int
	ToCol      int
	PlayerID   int
	IsAuto     bool
	ResultChan chan bool
}

type GameService struct {
	repo          repository.Repository
	games         map[int]*GameState
	players       []*player.Player
	moveChan      chan MoveRequest
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	mu            sync.RWMutex
	gameCounter   int
	isMultiBoard  bool
	displayTicker *time.Ticker
}

type GameState struct {
	Game          *game.Game
	Board         *board.Board
	Player1       *player.Player
	Player2       *player.Player
	CurrentPlayer int
	Moves         []game.Move
	mu            sync.RWMutex
}

func NewGameService(repo repository.Repository) *GameService {
	ctx, cancel := context.WithCancel(context.Background())
	
	gs := &GameService{
		repo:         repo,
		games:        make(map[int]*GameState),
		players:      make([]*player.Player, 0),
		moveChan:     make(chan MoveRequest, 100),
		ctx:          ctx,
		cancel:       cancel,
		gameCounter:  0,
		isMultiBoard: false,
	}

	// Запускаем обработчик ходов
	gs.wg.Add(1)
	go gs.processMoves()

	return gs
}

func (s *GameService) processMoves() {
	defer s.wg.Done()
	
	for {
		select {
		case req := <-s.moveChan:
			success := s.MakeMove(req.GameID, req.FromRow, req.FromCol, req.ToRow, req.ToCol)
			if req.ResultChan != nil {
				select {
				case req.ResultChan <- success:
				case <-s.ctx.Done():
					return
				}
			}
		case <-s.ctx.Done():
			fmt.Println("[GameService] Остановка обработчика ходов")
			return
		}
	}
}

func (s *GameService) InitGame(boardSize int, player1, player2 *player.Player) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	gameID := s.gameCounter
	s.gameCounter++

	gameObj := game.NewGame(gameID, boardSize)
	boardObj := board.SetupInitialBoard(gameID, gameID, boardSize)

	state := &GameState{
		Game:          gameObj,
		Board:         boardObj,
		Player1:       player1,
		Player2:       player2,
		CurrentPlayer: 0,
		Moves:         make([]game.Move, 0),
	}

	s.games[gameID] = state
	s.players = []*player.Player{player1, player2}

	s.repo.Save(player1)
	s.repo.Save(player2)
	s.repo.Save(gameObj)
	s.repo.Save(boardObj)

	return gameID
}

func (s *GameService) MakeMove(gameID, fromRow, fromCol, toRow, toCol int) bool {
	state := s.getGameState(gameID)
	if state == nil {
		return false
	}

	state.mu.Lock()
	defer state.mu.Unlock()

	if state.Game.IsFinished() {
		return false
	}

	piece := state.Board.GetPiece(fromRow, fromCol)
	if piece == nil {
		return false
	}

	if !state.Board.IsEmpty(toRow, toCol) {
		return false
	}

	currentColor := game.White
	if state.CurrentPlayer == 1 {
		currentColor = game.Black
	}
	if piece.(*game.Piece).GetColor() != currentColor {
		return false
	}

	if !state.Board.MovePiece(fromRow, fromCol, toRow, toCol) {
		return false
	}

	move := game.NewMove(fromRow, fromCol, toRow, toCol, 
		piece.(*game.Piece), state.CurrentPlayer+1, gameID)
	
	startTime := time.Now()
	state.Game.AddMove(move)
	move.SetDuration(time.Since(startTime))
	
	state.Moves = append(state.Moves, move)

	s.repo.Save(move)

	state.CurrentPlayer = (state.CurrentPlayer + 1) % 2

	return true
}

func (s *GameService) getGameState(gameID int) *GameState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.games[gameID]
}

func (s *GameService) StartMultiBoardSimulation(numBoards, boardSize int) {
	s.mu.Lock()
	s.isMultiBoard = true
	s.mu.Unlock()

	// Создаем несколько досок
	for i := 0; i < numBoards; i++ {
		player1 := player.NewPlayer(fmt.Sprintf("Игрок %d (белые)", i+1), i*2+1)
		player2 := player.NewPlayer(fmt.Sprintf("Игрок %d (черные)", i+1), i*2+2)
		s.InitGame(boardSize, player1, player2)
	}

	// Запускаем симуляцию
	s.wg.Add(1)
	go s.simulateMultiBoard()

	// Запускаем отображение
	s.displayTicker = time.NewTicker(1 * time.Second)
	s.wg.Add(1)
	go s.displayBoards()
}

func (s *GameService) simulateMultiBoard() {
	defer s.wg.Done()
	
	rand.Seed(time.Now().UnixNano())

	s.mu.RLock()
	gameIDs := make([]int, 0, len(s.games))
	for id := range s.games {
		gameIDs = append(gameIDs, id)
	}
	s.mu.RUnlock()

	for {
		select {
		case <-s.ctx.Done():
			fmt.Println("[GameService] Остановка симуляции многодосковой игры")
			return
		default:
			// Проверяем, все ли игры завершены
			allFinished := true
			s.mu.RLock()
			for _, state := range s.games {
				if !state.Game.IsFinished() {
					allFinished = false
					break
				}
			}
			s.mu.RUnlock()

			if allFinished {
				fmt.Println("[GameService] Все игры завершены")
				return
			}

			// Делаем ход в случайной игре
			gameID := gameIDs[rand.Intn(len(gameIDs))]
			state := s.getGameState(gameID)
			if state == nil || state.Game.IsFinished() {
				continue
			}

			// Находим случайную фигуру текущего игрока
			state.mu.Lock()
			currentColor := game.White
			if state.CurrentPlayer == 1 {
				currentColor = game.Black
			}
			state.mu.Unlock()

			var fromRow, fromCol, toRow, toCol int
			found := false

			for attempts := 0; attempts < 100 && !found; attempts++ {
				fromRow = rand.Intn(state.Board.GetSize())
				fromCol = rand.Intn(state.Board.GetSize())

				piece := state.Board.GetPiece(fromRow, fromCol)
				if piece == nil {
					continue
				}

				if piece.(*game.Piece).GetColor() != currentColor {
					continue
				}

				for attempts2 := 0; attempts2 < 20; attempts2++ {
					toRow = rand.Intn(state.Board.GetSize())
					toCol = rand.Intn(state.Board.GetSize())

					if state.Board.IsEmpty(toRow, toCol) {
						found = true
						break
					}
				}
			}

			if !found {
				state.mu.Lock()
				state.Game.SetFinished(true)
				state.mu.Unlock()
				continue
			}

			// Отправляем запрос на ход
			req := MoveRequest{
				GameID:   gameID,
				FromRow:  fromRow,
				FromCol:  fromCol,
				ToRow:    toRow,
				ToCol:    toCol,
				PlayerID: state.CurrentPlayer + 1,
				IsAuto:   true,
			}
			
			select {
			case s.moveChan <- req:
			case <-s.ctx.Done():
				return
			}

			// Небольшая задержка между ходами
			select {
			case <-time.After(time.Duration(rand.Intn(500)+500) * time.Millisecond):
			case <-s.ctx.Done():
				return
			}
		}
	}
}

func (s *GameService) displayBoards() {
	defer s.wg.Done()
	
	for {
		select {
		case <-s.displayTicker.C:
			s.printAllBoards()
		case <-s.ctx.Done():
			s.displayTicker.Stop()
			fmt.Println("[GameService] Остановка отображения досок")
			return
		}
	}
}

func (s *GameService) printAllBoards() {
	fmt.Print("\033[H\033[2J")
	
	s.mu.RLock()
	defer s.mu.RUnlock()

	fmt.Println("=== МНОГОДОСКОВАЯ ИГРА ===")
	fmt.Printf("Активных досок: %d\n\n", len(s.games))

	for id, state := range s.games {
		state.mu.RLock()
		fmt.Printf("Игра %d:\n", id)
		fmt.Printf("  Игроки: %s (белые) vs %s (черные)\n", 
			state.Player1.GetName(), state.Player2.GetName())
		fmt.Printf("  Ход: %d\n", state.Game.GetCurrentTurn())
		fmt.Printf("  Текущий игрок: %d\n", state.CurrentPlayer+1)
		fmt.Printf("  Время игры: %v\n", state.Game.GetGameDuration())
		
		if state.Game.IsFinished() {
			fmt.Printf("  Статус: ЗАВЕРШЕНА\n")
			if winner := state.Game.GetWinner(); winner > 0 {
				fmt.Printf("  Победитель: Игрок %d\n", winner)
			}
		}
		
		if len(state.Moves) > 0 {
			lastMove := state.Moves[len(state.Moves)-1]
			fmt.Printf("  Последний ход: %v\n", lastMove.String())
		}
		
		state.mu.RUnlock()
		fmt.Println()
	}
	
	fmt.Println("Нажмите Enter для выхода...")
}

func (s *GameService) SetMultiBoardMode(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.isMultiBoard = enabled
}

func (s *GameService) IsMultiBoard() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isMultiBoard
}

func (s *GameService) Stop() {
	fmt.Println("[GameService] Получен сигнал остановки")
	s.cancel()
	s.wg.Wait()
	fmt.Println("[GameService] Все горутины остановлены")
}

func (s *GameService) GetGameState(gameID int) *GameState {
	return s.getGameState(gameID)
}

func (s *GameService) GetAllGames() map[int]*GameState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.games
}