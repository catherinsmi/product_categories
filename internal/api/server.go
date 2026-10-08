package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"product_categories/internal/app/handler"
	"product_categories/internal/app/repository"
	"product_categories/internal/storage"
)

// NewServer собирает роутер Gin со всеми страницами, статикой и прокси /media.
func NewServer(repo *repository.Repository, st *storage.Storage, log *logrus.Logger) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(loggerMiddleware(log))

	// ВАЖНО: НЕ используем LoadHTMLGlob("templates/**/*.html") — Gin не поддерживает **.
	// Один вызов LoadHTMLFiles со всеми файлами: партиалы + страницы.
	// LoadHTMLFiles не перезаписывает ранее загруженные шаблоны.
	r.LoadHTMLFiles(
		"templates/partials/head.html",
		"templates/partials/bottom_nav.html",
		"templates/partials/like_icon.html",
		"templates/feed.html",
		"templates/add.html",
		"templates/grid.html",
	)

	// Статика (CSS).
	r.Static("/styles", "./resources/styles")

	h := handler.New(repo, st, log)

	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/feed")
	})
	r.GET("/feed", h.FeedIndex)
	r.GET("/feed/:id", h.FeedByID)
	r.GET("/add", h.Add)
	r.GET("/grid", h.Grid)
	r.GET("/media/*key", h.Media)

	return r
}

func loggerMiddleware(log *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		log.WithFields(logrus.Fields{
			"status":  c.Writer.Status(),
			"method":  c.Request.Method,
			"path":    c.Request.URL.Path,
			"elapsed": time.Since(start).String(),
		}).Info("http")
	}
}
