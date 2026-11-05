package handler

import (
	"net/http"

	"product_categories/internal/app/repository"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type UserHandler struct {
	repo *repository.Repo
	log  *logrus.Logger
}

func NewUserHandler(repo *repository.Repo, log *logrus.Logger) *UserHandler {
	return &UserHandler{repo: repo, log: log}
}

// POST /api/users/register
func (h *UserHandler) Register(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid data"})
		return
	}

	user := &repository.User{Username: req.Username, Email: req.Email}
	if err := h.repo.CreateUser(user); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "user already exists"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": user.ID, "message": "registered"})
}

// POST /api/users/login (заглушка)
func (h *UserHandler) Login(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "auth stub for lab 4"})
}

// POST /api/users/logout (заглушка)
func (h *UserHandler) Logout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "logout stub for lab 4"})
}
