package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Item — модель данных
type Item struct {
	ID    int    `json:"id"    example:"1"`
	Name  string `json:"name"  example:"Молоко"`
	Price int    `json:"price" example:"120"`
}

var (
	items  = map[int]Item{}
	nextID = 1
)

// ListItems godoc
// @Summary      Список элементов
// @Tags         items
// @Produce      json
// @Success      200 {array} Item
// @Router       /items [get]
func ListItems(c *gin.Context) {
	out := make([]Item, 0, len(items))
	for _, it := range items {
		out = append(out, it)
	}
	c.JSON(http.StatusOK, out)
}

// GetItem godoc
// @Summary      Получить элемент по ID
// @Tags         items
// @Produce      json
// @Param        id path int true "ID элемента"
// @Success      200 {object} Item
// @Failure      404 {object} map[string]string
// @Router       /items/{id} [get]
func GetItem(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	it, ok := items[id]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, it)
}

// CreateItem godoc
// @Summary      Создать элемент
// @Tags         items
// @Accept       json
// @Produce      json
// @Param        item body Item true "Новый элемент"
// @Success      201 {object} Item
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Security     BearerAuth
// @Router       /items [post]
func CreateItem(c *gin.Context) {
	var it Item
	if err := c.ShouldBindJSON(&it); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	it.ID = nextID
	nextID++
	items[it.ID] = it
	c.JSON(http.StatusCreated, it)
}

// UpdateItem godoc
// @Summary      Обновить элемент
// @Tags         items
// @Accept       json
// @Produce      json
// @Param        id   path int  true "ID элемента"
// @Param        item body Item true "Новые данные"
// @Success      200 {object} Item
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Security     BearerAuth
// @Router       /items/{id} [put]
func UpdateItem(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if _, ok := items[id]; !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	var it Item
	if err := c.ShouldBindJSON(&it); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	it.ID = id
	items[id] = it
	c.JSON(http.StatusOK, it)
}

// DeleteItem godoc
// @Summary      Удалить элемент
// @Tags         items
// @Produce      json
// @Param        id path int true "ID элемента"
// @Success      204
// @Failure      401 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Security     BearerAuth
// @Router       /items/{id} [delete]
func DeleteItem(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if _, ok := items[id]; !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	delete(items, id)
	c.Status(http.StatusNoContent)
}