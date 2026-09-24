package handlers

import (
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// LoginRequest — тело запроса на авторизацию
type LoginRequest struct {
	Login    string `json:"login"    example:"admin"`
	Password string `json:"password" example:"admin123"`
}

// LoginResponse — ответ с JWT
type LoginResponse struct {
	Token string `json:"token" example:"eyJhbGciOiJIUzI1NiIs..."`
}

// Login godoc
// @Summary      Авторизация
// @Description  Проверяет login/password из .env и возвращает JWT
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        creds body LoginRequest true "Учётные данные"
// @Success      200 {object} LoginResponse
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Router       /login [post]
func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}

	if req.Login != os.Getenv("LOGIN") || req.Password != os.Getenv("PASSWORD") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": req.Login,
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	signed, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot sign token"})
		return
	}

	c.JSON(http.StatusOK, LoginResponse{Token: signed})
}