// cmd/server/main.go
package main

import (
	"chess/internal/handler"
	"chess/internal/repository"
	"chess/internal/service"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Создаем репозиторий
	repo := repository.NewRepository("data")

	// Создаем сервис
	gameService := service.NewGameService(repo)

	// Создаем handlers
	gameHandler := handler.NewGameHandler(gameService, repo)

	// Настраиваем роутер
	mux := http.NewServeMux()

	// Статические файлы
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./web/static"))))

	// HTML страница для наблюдателей
	mux.HandleFunc("/", gameHandler.ServeIndex)

	// API endpoints
	mux.HandleFunc("POST /api/games", gameHandler.CreateGame)
	mux.HandleFunc("GET /api/games", gameHandler.GetGames)
	mux.HandleFunc("GET /api/games/{id}", gameHandler.GetGameByID)
	mux.HandleFunc("PUT /api/games/{id}", gameHandler.UpdateGame)
	mux.HandleFunc("DELETE /api/games/{id}", gameHandler.DeleteGame)

	mux.HandleFunc("POST /api/moves", gameHandler.CreateMove)
	mux.HandleFunc("GET /api/moves", gameHandler.GetMoves)
	mux.HandleFunc("GET /api/moves/{id}", gameHandler.GetMoveByID)

	mux.HandleFunc("POST /api/players", gameHandler.CreatePlayer)
	mux.HandleFunc("GET /api/players", gameHandler.GetPlayers)
	mux.HandleFunc("GET /api/players/{id}", gameHandler.GetPlayerByID)

	mux.HandleFunc("POST /api/boards", gameHandler.CreateBoard)
	mux.HandleFunc("GET /api/boards", gameHandler.GetBoards)
	mux.HandleFunc("GET /api/boards/{id}", gameHandler.GetBoardByID)

	// WebSocket для наблюдателей
	mux.HandleFunc("/ws", gameHandler.HandleWebSocket)

	// Создаем сервер
	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Запускаем сервер в горутине
	go func() {
		log.Printf("Сервер запущен на http://localhost:8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Ошибка запуска сервера: %v", err)
		}
	}()

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("Получен сигнал остановки, завершаем работу...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Ошибка при завершении сервера: %v", err)
	}

	// Сохраняем данные
	if err := repo.SaveAll(); err != nil {
		log.Printf("Ошибка сохранения данных: %v", err)
	}

	log.Println("Сервер остановлен")
}