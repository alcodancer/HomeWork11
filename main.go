package main

import (
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "chess/docs" // сгенерированные docs (swag init)
	"chess/internal/handlers"
	"chess/internal/middleware"
)

// @title           Items API
// @version         1.0
// @description     Сервер управления данными с JWT-авторизацией
// @host            localhost:8080
// @BasePath        /api/v1
//
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Введите токен в формате: Bearer <JWT>
func main() {
	_ = godotenv.Load()

	r := gin.Default()

	api := r.Group("/api/v1")
	{
		// Публичный роут авторизации
		api.POST("/login", handlers.Login)

		// Публичные чтения
		api.GET("/items", handlers.ListItems)
		api.GET("/items/:id", handlers.GetItem)

		// Защищённые изменения
		auth := api.Group("")
		auth.Use(middleware.JWTAuth())
		{
			auth.POST("/items", handlers.CreateItem)
			auth.PUT("/items/:id", handlers.UpdateItem)
			auth.DELETE("/items/:id", handlers.DeleteItem)
		}
	}

	// Страница Swagger UI
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	_ = r.Run(":" + os.Getenv("PORT"))
}