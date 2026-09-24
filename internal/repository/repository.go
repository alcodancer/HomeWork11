// internal/repository/repository.go
package repository

import (
	"chess/internal/model/board"
	"chess/internal/model/game"
	"chess/internal/model/player"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Entity интерфейс для всех сущностей
type Entity interface {
	GetID() int
}

// Repository интерфейс для работы с хранилищем
type Repository interface {
	Save(entity interface{})
	GetPlayers() []*player.Player
	GetGames() []*game.Game
	GetMoves() []game.Move
	GetBoards() []*board.Board
	GetMovesByGameID(gameID int) []game.Move
	ClearAll()
	LoadFromFiles() error
	SaveAll() error
}

type repository struct {
	players []*player.Player
	games   []*game.Game
	moves   []game.Move
	boards  []*board.Board
	mu      sync.RWMutex
	dataDir string
}

// Структуры для сериализации в JSON
type PlayerData struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type GameData struct {
	ID          int       `json:"id"`
	Size        int       `json:"size"`
	Moves       []MoveData `json:"moves"`
	IsFinished  bool      `json:"is_finished"`
	Winner      int       `json:"winner"`
	CurrentTurn int       `json:"current_turn"`
	StartTime   time.Time `json:"start_time"`
}

type MoveData struct {
	FromRow   int       `json:"from_row"`
	FromCol   int       `json:"from_col"`
	ToRow     int       `json:"to_row"`
	ToCol     int       `json:"to_col"`
	PieceType string    `json:"piece_type"`
	Color     string    `json:"color"`
	Symbol    string    `json:"symbol"`
	PlayerID  int       `json:"player_id"`
	GameID    int       `json:"game_id"`
	Timestamp time.Time `json:"timestamp"`
	Duration  int64     `json:"duration"` // в наносекундах
}

type BoardData struct {
	ID     int        `json:"id"`
	GameID int        `json:"game_id"`
	Size   int        `json:"size"`
	Cells  [][]string `json:"cells"` // символы фигур или пустые строки
}

func NewRepository(dataDir string) Repository {
	if dataDir == "" {
		dataDir = "data"
	}
	
	// Создаем директорию для данных, если её нет
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		fmt.Printf("Ошибка создания директории %s: %v\n", dataDir, err)
	}

	repo := &repository{
		players: make([]*player.Player, 0),
		games:   make([]*game.Game, 0),
		moves:   make([]game.Move, 0),
		boards:  make([]*board.Board, 0),
		dataDir: dataDir,
	}

	// Загружаем данные из файлов при создании
	if err := repo.LoadFromFiles(); err != nil {
		fmt.Printf("Ошибка загрузки данных: %v\n", err)
	}

	return repo
}

func (r *repository) Save(entity interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()

	switch v := entity.(type) {
	case *player.Player:
		r.players = append(r.players, v)
		r.savePlayersToFile()
	case *game.Game:
		r.games = append(r.games, v)
		r.saveGamesToFile()
	case game.Move:
		r.moves = append(r.moves, v)
		r.saveMovesToFile()
	case *board.Board:
		r.boards = append(r.boards, v)
		r.saveBoardsToFile()
	default:
		fmt.Printf("[Repository] Неизвестный тип: %T\n", v)
	}
}

func (r *repository) GetPlayers() []*player.Player {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.players
}

func (r *repository) GetGames() []*game.Game {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.games
}

func (r *repository) GetMoves() []game.Move {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.moves
}

func (r *repository) GetBoards() []*board.Board {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.boards
}

func (r *repository) GetMovesByGameID(gameID int) []game.Move {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	var result []game.Move
	for _, move := range r.moves {
		if move.GetGameID() == gameID {
			result = append(result, move)
		}
	}
	return result
}

func (r *repository) ClearAll() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.players = make([]*player.Player, 0)
	r.games = make([]*game.Game, 0)
	r.moves = make([]game.Move, 0)
	r.boards = make([]*board.Board, 0)
	
	// Удаляем файлы
	os.Remove(filepath.Join(r.dataDir, "players.json"))
	os.Remove(filepath.Join(r.dataDir, "games.json"))
	os.Remove(filepath.Join(r.dataDir, "moves.json"))
	os.Remove(filepath.Join(r.dataDir, "boards.json"))
}

// Сохранение игроков в JSON
func (r *repository) savePlayersToFile() {
	playersData := make([]PlayerData, len(r.players))
	for i, p := range r.players {
		playersData[i] = PlayerData{
			ID:   p.GetID(),
			Name: p.GetName(),
		}
	}
	r.saveToFile("players.json", playersData)
}

// Сохранение игр в JSON
func (r *repository) saveGamesToFile() {
	gamesData := make([]GameData, len(r.games))
	for i, g := range r.games {
		moves := g.GetMoves()
		movesData := make([]MoveData, len(moves))
		for j, m := range moves {
			movesData[j] = MoveData{
				FromRow:   m.GetFromRow(),
				FromCol:   m.GetFromCol(),
				ToRow:     m.GetToRow(),
				ToCol:     m.GetToCol(),
				PieceType: m.GetPiece().GetType(),
				Color:     string(m.GetPiece().GetColor()),
				Symbol:    string(m.GetPiece().GetSymbol()),
				PlayerID:  m.GetPlayerID(),
				GameID:    m.GetGameID(),
				Timestamp: m.GetTimestamp(),
				Duration:  m.GetDuration().Nanoseconds(),
			}
		}
		
		gamesData[i] = GameData{
			ID:          g.GetID(),
			Size:        g.GetSize(),
			Moves:       movesData,
			IsFinished:  g.IsFinished(),
			Winner:      g.GetWinner(),
			CurrentTurn: g.GetCurrentTurn(),
			StartTime:   g.GetStartTime(),
		}
	}
	r.saveToFile("games.json", gamesData)
}

// Сохранение ходов в JSON
func (r *repository) saveMovesToFile() {
	movesData := make([]MoveData, len(r.moves))
	for i, m := range r.moves {
		movesData[i] = MoveData{
			FromRow:   m.GetFromRow(),
			FromCol:   m.GetFromCol(),
			ToRow:     m.GetToRow(),
			ToCol:     m.GetToCol(),
			PieceType: m.GetPiece().GetType(),
			Color:     string(m.GetPiece().GetColor()),
			Symbol:    string(m.GetPiece().GetSymbol()),
			PlayerID:  m.GetPlayerID(),
			GameID:    m.GetGameID(),
			Timestamp: m.GetTimestamp(),
			Duration:  m.GetDuration().Nanoseconds(),
		}
	}
	r.saveToFile("moves.json", movesData)
}

// Сохранение досок в JSON
func (r *repository) saveBoardsToFile() {
	boardsData := make([]BoardData, len(r.boards))
	for i, b := range r.boards {
		cells := make([][]string, b.GetSize())
		for row := 0; row < b.GetSize(); row++ {
			cells[row] = make([]string, b.GetSize())
			for col := 0; col < b.GetSize(); col++ {
				piece := b.GetPiece(row, col)
				if piece != nil {
					cells[row][col] = string(piece.GetSymbol())
				} else {
					cells[row][col] = ""
				}
			}
		}
		
		boardsData[i] = BoardData{
			ID:     b.GetID(),
			GameID: b.GetGameID(),
			Size:   b.GetSize(),
			Cells:  cells,
		}
	}
	r.saveToFile("boards.json", boardsData)
}

// Вспомогательный метод для сохранения в файл
func (r *repository) saveToFile(filename string, data interface{}) {
	filepath := filepath.Join(r.dataDir, filename)
	
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Printf("Ошибка маршалинга %s: %v\n", filename, err)
		return
	}
	
	if err := os.WriteFile(filepath, jsonData, 0644); err != nil {
		fmt.Printf("Ошибка записи в файл %s: %v\n", filename, err)
	}
}

// Загрузка данных из файлов
func (r *repository) LoadFromFiles() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Загружаем игроков
	if err := r.loadPlayersFromFile(); err != nil {
		return err
	}
	
	// Загружаем игры
	if err := r.loadGamesFromFile(); err != nil {
		return err
	}
	
	// Загружаем ходы
	if err := r.loadMovesFromFile(); err != nil {
		return err
	}
	
	// Загружаем доски
	if err := r.loadBoardsFromFile(); err != nil {
		return err
	}
	
	fmt.Printf("[Repository] Загружено: %d игроков, %d игр, %d ходов, %d досок\n",
		len(r.players), len(r.games), len(r.moves), len(r.boards))
	
	return nil
}

// Загрузка игроков из файла
func (r *repository) loadPlayersFromFile() error {
	var playersData []PlayerData
	if err := r.loadFromFile("players.json", &playersData); err != nil {
		return err
	}
	
	r.players = make([]*player.Player, len(playersData))
	for i, pd := range playersData {
		r.players[i] = player.NewPlayer(pd.Name, pd.ID)
	}
	return nil
}

// Загрузка игр из файла
func (r *repository) loadGamesFromFile() error {
	var gamesData []GameData
	if err := r.loadFromFile("games.json", &gamesData); err != nil {
		return err
	}
	
	r.games = make([]*game.Game, len(gamesData))
	for i, gd := range gamesData {
		gameObj := game.NewGame(gd.ID, gd.Size)
		gameObj.SetFinished(gd.IsFinished)
		gameObj.SetWinner(gd.Winner)
		
		// Восстанавливаем ходы
		for _, md := range gd.Moves {
			color := game.Color(md.Color)
			piece := game.NewPiece(md.PieceType, color, []rune(md.Symbol)[0])
			move := game.NewMove(md.FromRow, md.FromCol, md.ToRow, md.ToCol, piece, md.PlayerID, md.GameID)
			move.SetDuration(time.Duration(md.Duration))
			gameObj.AddMove(move)
		}
		
		r.games[i] = gameObj
	}
	return nil
}

// Загрузка ходов из файла
func (r *repository) loadMovesFromFile() error {
	var movesData []MoveData
	if err := r.loadFromFile("moves.json", &movesData); err != nil {
		return err
	}
	
	r.moves = make([]game.Move, len(movesData))
	for i, md := range movesData {
		color := game.Color(md.Color)
		piece := game.NewPiece(md.PieceType, color, []rune(md.Symbol)[0])
		move := game.NewMove(md.FromRow, md.FromCol, md.ToRow, md.ToCol, piece, md.PlayerID, md.GameID)
		move.SetDuration(time.Duration(md.Duration))
		r.moves[i] = move
	}
	return nil
}

// Загрузка досок из файла
func (r *repository) loadBoardsFromFile() error {
	var boardsData []BoardData
	if err := r.loadFromFile("boards.json", &boardsData); err != nil {
		return err
	}
	
	r.boards = make([]*board.Board, len(boardsData))
	for i, bd := range boardsData {
		boardObj := board.NewBoard(bd.ID, bd.GameID, bd.Size)
		
		// Восстанавливаем фигуры
		for row := 0; row < bd.Size; row++ {
			for col := 0; col < bd.Size; col++ {
				symbol := bd.Cells[row][col]
				if symbol != "" {
					// Определяем тип и цвет фигуры по символу
					pieceType := "piece"
					color := game.White
					
					// Простая логика определения по символу
					switch symbol {
					case "♔", "♕", "♖", "♗", "♘", "♙":
						color = game.White
					case "♚", "♛", "♜", "♝", "♞", "♟":
						color = game.Black
					}
					
					switch symbol {
					case "♔", "♚":
						pieceType = "king"
					case "♕", "♛":
						pieceType = "queen"
					case "♖", "♜":
						pieceType = "rook"
					case "♗", "♝":
						pieceType = "bishop"
					case "♘", "♞":
						pieceType = "knight"
					case "♙", "♟":
						pieceType = "pawn"
					}
					
					piece := game.NewPiece(pieceType, color, []rune(symbol)[0])
					boardObj.SetPiece(row, col, piece)
				}
			}
		}
		
		r.boards[i] = boardObj
	}
	return nil
}

// Вспомогательный метод для загрузки из файла
func (r *repository) loadFromFile(filename string, data interface{}) error {
	filepath := filepath.Join(r.dataDir, filename)
	
	// Проверяем существование файла
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		return nil // Файл не существует - не ошибка
	}
	
	jsonData, err := os.ReadFile(filepath)
	if err != nil {
		return err
	}
	
	if err := json.Unmarshal(jsonData, data); err != nil {
		return err
	}
	
	return nil
}

// Сохранение всех данных
func (r *repository) SaveAll() error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	r.savePlayersToFile()
	r.saveGamesToFile()
	r.saveMovesToFile()
	r.saveBoardsToFile()
	
	fmt.Println("[Repository] Все данные сохранены")
	return nil
}