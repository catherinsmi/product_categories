package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"product_categories/internal/app/handler"
	"product_categories/internal/app/repository"
)

// NewServer собирает Gin-роутер.
// st *storage.Storage больше не нужен — MinIO убран, media отдаётся статикой.
func NewServer(repo *repository.Repo, log *logrus.Logger, staticDir string) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(loggerMiddleware(log))

	// Gin не поддерживает ** в LoadHTMLGlob — перечисляем файлы явно.
	r.LoadHTMLFiles(
		"templates/partials/head.html",
		"templates/partials/bottom_nav.html",
		"templates/partials/like_icon.html",
		"templates/product_categories_feed.html",
		"templates/product_categories_add.html",
		"templates/product_categories_grid.html",
	)

	// Статика: /static/* → resources/static/*
	// Здесь лежат styles/app.css, media/*, default/*.
	r.Static("/static", staticDir)

	h := handler.New(repo, log, staticDir)

	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/product_categories/feed")
	})

	pc := r.Group("/product_categories")
	{
		// ===== 3 GET =====
		pc.GET("/feed", h.FeedFirst)    // первая published → редирект на /feed/:id
		pc.GET("/feed/:id", h.FeedByID) // одна строка published (или ?next=true)
		pc.GET("/add", h.Add)           // черновик текущего пользователя
		pc.GET("/grid", h.Grid)         // список published + фильтр по цене

		// ===== 3 POST =====
		pc.POST("/add", h.CreateDraft) // «Далее» — создать draft (ORM)
		pc.POST("/publish", h.Publish) // «Опубликовать» — draft→published (ORM)
		pc.POST("/delete", h.Delete)   // логическое удаление (SQL UPDATE, не ORM)
	}

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
