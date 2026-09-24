// internal/handler/game_handler.go
package handler

import (
	"chess/internal/model/board"
	"chess/internal/model/game"
	"chess/internal/model/player"
	"chess/internal/repository"
	"chess/internal/service"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type GameHandler struct {
	gameService *service.GameService
	repo        repository.Repository
	upgrader    websocket.Upgrader
	clients     map[*websocket.Conn]bool
	mu          sync.RWMutex
}

func NewGameHandler(gameService *service.GameService, repo repository.Repository) *GameHandler {
	return &GameHandler{
		gameService: gameService,
		repo:        repo,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		clients: make(map[*websocket.Conn]bool),
	}
}

// ServeIndex - отдает HTML страницу для наблюдателей
func (h *GameHandler) ServeIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	html := `
<!DOCTYPE html>
<html>
<head>
    <title>Шахматы - Наблюдатель</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background: #f0f0f0; }
        .container { max-width: 1200px; margin: 0 auto; }
        .board-container { 
            display: inline-block; 
            margin: 10px; 
            padding: 20px; 
            background: white; 
            border-radius: 10px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        .board { font-family: monospace; font-size: 20px; line-height: 1.4; }
        .info { margin-top: 10px; padding: 10px; background: #f8f8f8; border-radius: 5px; }
        .status { 
            padding: 5px 10px; 
            border-radius: 5px; 
            display: inline-block;
            font-weight: bold;
        }
        .status.active { background: #4CAF50; color: white; }
        .status.finished { background: #f44336; color: white; }
        .moves { max-height: 200px; overflow-y: auto; font-size: 14px; }
        #boards { display: flex; flex-wrap: wrap; gap: 20px; }
        #status-bar {
            background: #333;
            color: white;
            padding: 10px;
            margin-bottom: 20px;
            border-radius: 5px;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>♚ Шахматы - Панель наблюдателя</h1>
        <div id="status-bar">
            <span id="connection-status">Подключение...</span>
            <span style="margin-left: 20px;">Досок: <span id="board-count">0</span></span>
            <span style="margin-left: 20px;">Игроков: <span id="player-count">0</span></span>
        </div>
        <div id="boards"></div>
    </div>
    <script>
        const ws = new WebSocket('ws://localhost:8080/ws');
        
        ws.onopen = function() {
            document.getElementById('connection-status').textContent = '✅ Подключено';
            document.getElementById('connection-status').style.color = '#4CAF50';
        };
        
        ws.onclose = function() {
            document.getElementById('connection-status').textContent = '❌ Отключено';
            document.getElementById('connection-status').style.color = '#f44336';
        };
        
        ws.onmessage = function(event) {
            const data = JSON.parse(event.data);
            updateBoards(data);
        };
        
        function updateBoards(data) {
            const boardsDiv = document.getElementById('boards');
            boardsDiv.innerHTML = '';
            
            document.getElementById('board-count').textContent = data.games ? data.games.length : 0;
            document.getElementById('player-count').textContent = data.players ? data.players.length : 0;
            
            if (data.games) {
                data.games.forEach(game => {
                    const boardDiv = document.createElement('div');
                    boardDiv.className = 'board-container';
                    
                    let boardHTML = '<div class="board">';
                    // Здесь должна быть отрисовка доски
                    boardHTML += '<pre>' + (game.board || 'Доска не доступна') + '</pre>';
                    boardHTML += '</div>';
                    
                    boardHTML += '<div class="info">';
                    boardHTML += '<strong>Игра #' + game.id + '</strong><br>';
                    boardHTML += 'Ход: ' + (game.current_turn || 0) + '<br>';
                    boardHTML += 'Статус: <span class="status ' + (game.is_finished ? 'finished' : 'active') + '">' + 
                                (game.is_finished ? 'Завершена' : 'Активна') + '</span><br>';
                    if (game.winner > 0) {
                        boardHTML += 'Победитель: Игрок ' + game.winner + '<br>';
                    }
                    boardHTML += '</div>';
                    
                    boardDiv.innerHTML = boardHTML;
                    boardsDiv.appendChild(boardDiv);
                });
            }
        }
        
        // Обновление каждые 2 секунды
        setInterval(() => {
            if (ws.readyState === WebSocket.OPEN) {
                ws.send(JSON.stringify({ type: 'refresh' }));
            }
        }, 2000);
    </script>
</body>
</html>`
	w.Write([]byte(html))
}

// HandleWebSocket - обрабатывает WebSocket соединения
func (h *GameHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Ошибка WebSocket: %v", err)
		return
	}
	defer conn.Close()

	h.mu.Lock()
	h.clients[conn] = true
	h.mu.Unlock()

	log.Printf("Новый наблюдатель подключен")

	// Отправляем текущее состояние
	h.broadcastState()

	for {
		var msg map[string]interface{}
		err := conn.ReadJSON(&msg)
		if err != nil {
			h.mu.Lock()
			delete(h.clients, conn)
			h.mu.Unlock()
			log.Printf("Наблюдатель отключен")
			break
		}

		if msg["type"] == "refresh" {
			h.broadcastState()
		}
	}
}

// broadcastState - отправляет состояние всем наблюдателям
func (h *GameHandler) broadcastState() {
	state := h.getCurrentState()
	
	h.mu.RLock()
	clients := make([]*websocket.Conn, 0, len(h.clients))
	for conn := range h.clients {
		clients = append(clients, conn)
	}
	h.mu.RUnlock()

	for _, conn := range clients {
		if err := conn.WriteJSON(state); err != nil {
			conn.Close()
			h.mu.Lock()
			delete(h.clients, conn)
			h.mu.Unlock()
		}
	}
}

// getCurrentState - возвращает текущее состояние игры
func (h *GameHandler) getCurrentState() map[string]interface{} {
	games := h.repo.GetGames()
	players := h.repo.GetPlayers()
	moves := h.repo.GetMoves()
	boards := h.repo.GetBoards()

	gamesData := make([]map[string]interface{}, len(games))
	for i, g := range games {
		gamesData[i] = map[string]interface{}{
			"id":            g.GetID(),
			"size":          g.GetSize(),
			"current_turn":  g.GetCurrentTurn(),
			"is_finished":   g.IsFinished(),
			"winner":        g.GetWinner(),
			"total_moves":   len(g.GetMoves()),
			"duration":      g.GetGameDuration().String(),
		}
		
		// Находим соответствующую доску
		for _, b := range boards {
			if b.GetGameID() == g.GetID() {
				gamesData[i]["board"] = b.String()
				break
			}
		}
	}

	playersData := make([]map[string]interface{}, len(players))
	for i, p := range players {
		playersData[i] = map[string]interface{}{
			"id":   p.GetID(),
			"name": p.GetName(),
		}
	}

	movesData := make([]map[string]interface{}, len(moves))
	for i, m := range moves {
		movesData[i] = map[string]interface{}{
			"id":        i,
			"game_id":   m.GetGameID(),
			"player_id": m.GetPlayerID(),
			"from":      fmt.Sprintf("%c%d", 'A'+m.GetFromCol(), m.GetFromRow()+1),
			"to":        fmt.Sprintf("%c%d", 'A'+m.GetToCol(), m.GetToRow()+1),
			"piece":     m.GetPiece().String(),
			"duration":  m.GetDuration().String(),
		}
	}

	return map[string]interface{}{
		"games":   gamesData,
		"players": playersData,
		"moves":   movesData,
		"time":    time.Now().Format("15:04:05"),
	}
}

// API Handlers

// CreateGame - POST /api/games
func (h *GameHandler) CreateGame(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Size       int    `json:"size"`
		Player1    string `json:"player1"`
		Player2    string `json:"player2"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Size < 4 || req.Size > 20 {
		http.Error(w, "Size must be between 4 and 20", http.StatusBadRequest)
		return
	}
	if req.Player1 == "" || req.Player2 == "" {
		http.Error(w, "Player names are required", http.StatusBadRequest)
		return
	}

	player1 := player.NewPlayer(req.Player1, 1)
	player2 := player.NewPlayer(req.Player2, 2)
	
	gameID := h.gameService.InitGame(req.Size, player1, player2)
	
	h.repo.SaveAll()
	h.broadcastState()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":     gameID,
		"status": "created",
	})
}

// GetGames - GET /api/games
func (h *GameHandler) GetGames(w http.ResponseWriter, r *http.Request) {
	games := h.repo.GetGames()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(games)
}

// GetGameByID - GET /api/games/{id}
func (h *GameHandler) GetGameByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	games := h.repo.GetGames()
	for _, g := range games {
		if g.GetID() == id {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(g)
			return
		}
	}
	http.Error(w, "Game not found", http.StatusNotFound)
}

// UpdateGame - PUT /api/games/{id}
func (h *GameHandler) UpdateGame(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var req struct {
		Size       int    `json:"size"`
		IsFinished bool   `json:"is_finished"`
		Winner     int    `json:"winner"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	games := h.repo.GetGames()
	for _, g := range games {
		if g.GetID() == id {
			if req.Size >= 4 && req.Size <= 20 {
				g.SetSize(req.Size)
			}
			g.SetFinished(req.IsFinished)
			g.SetWinner(req.Winner)
			
			h.repo.SaveAll()
			h.broadcastState()
			
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "updated",
				"id":     id,
			})
			return
		}
	}
	http.Error(w, "Game not found", http.StatusNotFound)
}

// DeleteGame - DELETE /api/games/{id}
func (h *GameHandler) DeleteGame(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	// В реальном приложении нужно реализовать удаление из репозитория
	// Для простоты возвращаем ошибку
	http.Error(w, "Delete not implemented", http.StatusNotImplemented)
}

// CreateMove - POST /api/moves
func (h *GameHandler) CreateMove(w http.ResponseWriter, r *http.Request) {
	var req struct {
		GameID  int    `json:"game_id"`
		From    string `json:"from"`
		To      string `json:"to"`
		PlayerID int   `json:"player_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Парсим координаты
	fromRow, fromCol, err := parseCell(req.From)
	if err != nil {
		http.Error(w, "Invalid 'from' cell", http.StatusBadRequest)
		return
	}
	toRow, toCol, err := parseCell(req.To)
	if err != nil {
		http.Error(w, "Invalid 'to' cell", http.StatusBadRequest)
		return
	}

	// Корректируем для внутреннего представления
	fromRow--
	fromCol--
	toRow--
	toCol--

	success := h.gameService.MakeMove(req.GameID, fromRow, fromCol, toRow, toCol)
	if !success {
		http.Error(w, "Invalid move", http.StatusBadRequest)
		return
	}

	h.repo.SaveAll()
	h.broadcastState()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "move_executed",
	})
}

// GetMoves - GET /api/moves
func (h *GameHandler) GetMoves(w http.ResponseWriter, r *http.Request) {
	moves := h.repo.GetMoves()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(moves)
}

// GetMoveByID - GET /api/moves/{id}
func (h *GameHandler) GetMoveByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	moves := h.repo.GetMoves()
	if id >= 0 && id < len(moves) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(moves[id])
		return
	}
	http.Error(w, "Move not found", http.StatusNotFound)
}

// CreatePlayer - POST /api/players
func (h *GameHandler) CreatePlayer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	players := h.repo.GetPlayers()
	newID := len(players) + 1
	p := player.NewPlayer(req.Name, newID)
	h.repo.Save(p)
	h.repo.SaveAll()
	h.broadcastState()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":   newID,
		"name": req.Name,
	})
}

// GetPlayers - GET /api/players
func (h *GameHandler) GetPlayers(w http.ResponseWriter, r *http.Request) {
	players := h.repo.GetPlayers()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(players)
}

// GetPlayerByID - GET /api/players/{id}
func (h *GameHandler) GetPlayerByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	players := h.repo.GetPlayers()
	for _, p := range players {
		if p.GetID() == id {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(p)
			return
		}
	}
	http.Error(w, "Player not found", http.StatusNotFound)
}

// CreateBoard - POST /api/boards
func (h *GameHandler) CreateBoard(w http.ResponseWriter, r *http.Request) {
	var req struct {
		GameID int `json:"game_id"`
		Size   int `json:"size"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Size < 4 || req.Size > 20 {
		http.Error(w, "Size must be between 4 and 20", http.StatusBadRequest)
		return
	}

	boards := h.repo.GetBoards()
	newID := len(boards) + 1
	b := board.NewBoard(newID, req.GameID, req.Size)
	h.repo.Save(b)
	h.repo.SaveAll()
	h.broadcastState()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":      newID,
		"game_id": req.GameID,
		"size":    req.Size,
	})
}

// GetBoards - GET /api/boards
func (h *GameHandler) GetBoards(w http.ResponseWriter, r *http.Request) {
	boards := h.repo.GetBoards()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(boards)
}

// GetBoardByID - GET /api/boards/{id}
func (h *GameHandler) GetBoardByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	boards := h.repo.GetBoards()
	for _, b := range boards {
		if b.GetID() == id {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(b)
			return
		}
	}
	http.Error(w, "Board not found", http.StatusNotFound)
}

// Вспомогательные функции
func parseCell(cell string) (int, int, error) {
	if len(cell) < 2 {
		return 0, 0, fmt.Errorf("invalid cell format")
	}

	col := int(cell[0] - 'A')
	if col < 0 || col > 25 {
		return 0, 0, fmt.Errorf("invalid column")
	}

	row, err := strconv.Atoi(cell[1:])
	if err != nil {
		return 0, 0, err
	}

	return row, col, nil
}