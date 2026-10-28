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

func NewServer(repo *repository.Repository, st *storage.Storage, log *logrus.Logger, fallbackDir string) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(loggerMiddleware(log))

	r.LoadHTMLFiles(
		"templates/partials/head.html",
		"templates/partials/bottom_nav.html",
		"templates/partials/like_icon.html",
		"templates/product_categories_feed.html",
		"templates/product_categories_add.html",
		"templates/product_categories_grid.html",
	)

	r.Static("/product_categories/styles", "./resources/styles")

	h := handler.New(repo, st, log, fallbackDir)

	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/product_categories/feed")
	})

	pc := r.Group("/product_categories")
	{
		pc.GET("/feed", h.FeedIndex)
		pc.GET("/feed/:id", h.FeedByID)
		pc.GET("/add", h.Add)
		pc.GET("/grid", h.Grid)
		pc.GET("/media/*key", h.Media)
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
