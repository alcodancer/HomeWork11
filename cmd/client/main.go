// cmd/client/main.go
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	serverURL string
	gameID    int
	playerID  int
	client    *http.Client
}

type Game struct {
	ID          int    `json:"id"`
	Size        int    `json:"size"`
	CurrentTurn int    `json:"current_turn"`
	IsFinished  bool   `json:"is_finished"`
	Winner      int    `json:"winner"`
}

type MoveRequest struct {
	GameID   int    `json:"game_id"`
	From     string `json:"from"`
	To       string `json:"to"`
	PlayerID int    `json:"player_id"`
}

func NewClient(serverURL string) *Client {
	return &Client{
		serverURL: serverURL,
		client:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) CreateGame(size int, player1, player2 string) (int, error) {
	reqBody := map[string]interface{}{
		"size":    size,
		"player1": player1,
		"player2": player2,
	}
	jsonData, _ := json.Marshal(reqBody)

	resp, err := c.client.Post(c.serverURL+"/api/games", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("server error: %s", string(body))
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	return int(result["id"].(float64)), nil
}

func (c *Client) MakeMove(gameID, playerID int, from, to string) error {
	reqBody := MoveRequest{
		GameID:   gameID,
		From:     from,
		To:       to,
		PlayerID: playerID,
	}
	jsonData, _ := json.Marshal(reqBody)

	resp, err := c.client.Post(c.serverURL+"/api/moves", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("move failed: %s", string(body))
	}

	return nil
}

func (c *Client) GetGame(id int) (*Game, error) {
	resp, err := c.client.Get(c.serverURL + "/api/games/" + strconv.Itoa(id))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("game not found")
	}

	var game Game
	if err := json.NewDecoder(resp.Body).Decode(&game); err != nil {
		return nil, err
	}
	return &game, nil
}

func (c *Client) GetGames() ([]Game, error) {
	resp, err := c.client.Get(c.serverURL + "/api/games")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var games []Game
	if err := json.NewDecoder(resp.Body).Decode(&games); err != nil {
		return nil, err
	}
	return games, nil
}

func (c *Client) GetMoves() ([]map[string]interface{}, error) {
	resp, err := c.client.Get(c.serverURL + "/api/moves")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var moves []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&moves); err != nil {
		return nil, err
	}
	return moves, nil
}

func main() {
	serverURL := "http://localhost:8080"
	if len(os.Args) > 1 {
		serverURL = os.Args[1]
	}

	client := NewClient(serverURL)
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("=== Шахматы - Консольный клиент ===")
	fmt.Printf("Сервер: %s\n\n", serverURL)

	// Выбор игры
	fmt.Print("Введите номер игры (или 0 для создания новой): ")
	scanner.Scan()
	gameIDStr := strings.TrimSpace(scanner.Text())
	gameID, _ := strconv.Atoi(gameIDStr)

	if gameID == 0 {
		// Создание новой игры
		fmt.Print("Размер доски (4-20): ")
		scanner.Scan()
		size, _ := strconv.Atoi(strings.TrimSpace(scanner.Text()))
		if size < 4 || size > 20 {
			size = 8
		}

		fmt.Print("Имя первого игрока: ")
		scanner.Scan()
		player1 := strings.TrimSpace(scanner.Text())

		fmt.Print("Имя второго игрока: ")
		scanner.Scan()
		player2 := strings.TrimSpace(scanner.Text())

		id, err := client.CreateGame(size, player1, player2)
		if err != nil {
			fmt.Printf("Ошибка создания игры: %v\n", err)
			return
		}
		gameID = id
		fmt.Printf("Игра создана! ID: %d\n\n", gameID)
	}

	fmt.Printf("Подключен к игре #%d\n", gameID)
	fmt.Println("Введите ход в формате: A1 B2")
	fmt.Println("Команды: 'status' - статус игры, 'moves' - история ходов, 'exit' - выход")

	// Получаем ID игрока
	fmt.Print("Ваш ID игрока (1 или 2): ")
	scanner.Scan()
	playerID, _ := strconv.Atoi(strings.TrimSpace(scanner.Text()))
	if playerID < 1 || playerID > 2 {
		playerID = 1
	}

	for {
		fmt.Print("\n> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		parts := strings.Fields(input)

		switch strings.ToLower(parts[0]) {
		case "exit", "quit":
			fmt.Println("Выход из клиента")
			return

		case "status":
			game, err := client.GetGame(gameID)
			if err != nil {
				fmt.Printf("Ошибка: %v\n", err)
				continue
			}
			fmt.Printf("Игра #%d\n", game.ID)
			fmt.Printf("Размер: %dx%d\n", game.Size, game.Size)
			fmt.Printf("Ходов: %d\n", game.CurrentTurn)
			fmt.Printf("Статус: %s\n", map[bool]string{true: "Завершена", false: "Активна"}[game.IsFinished])
			if game.Winner > 0 {
				fmt.Printf("Победитель: Игрок %d\n", game.Winner)
			}

			// Показываем последние ходы
			moves, _ := client.GetMoves()
			gameMoves := []map[string]interface{}{}
			for _, m := range moves {
				if int(m["game_id"].(float64)) == gameID {
					gameMoves = append(gameMoves, m)
				}
			}
			if len(gameMoves) > 0 {
				fmt.Println("\nПоследние ходы:")
				for i := len(gameMoves) - 1; i >= 0 && i >= len(gameMoves)-5; i-- {
					m := gameMoves[i]
					fmt.Printf("  %s: %s -> %s (%s)\n",
						m["piece"], m["from"], m["to"], m["duration"])
				}
			}

		case "moves":
			moves, err := client.GetMoves()
			if err != nil {
				fmt.Printf("Ошибка: %v\n", err)
				continue
			}
			count := 0
			for _, m := range moves {
				if int(m["game_id"].(float64)) == gameID {
					count++
					fmt.Printf("%d: %s -> %s (%s)\n",
						count, m["from"], m["to"], m["piece"])
				}
			}
			if count == 0 {
				fmt.Println("Ходов пока нет")
			}

		default:
			// Обработка хода
			if len(parts) != 2 {
				fmt.Println("Неверный формат. Используйте: A1 B2")
				continue
			}

			from := strings.ToUpper(parts[0])
			to := strings.ToUpper(parts[1])

			err := client.MakeMove(gameID, playerID, from, to)
			if err != nil {
				fmt.Printf("Ошибка: %v\n", err)
			} else {
				fmt.Printf("Ход выполнен: %s -> %s\n", from, to)
			}
		}
	}
}