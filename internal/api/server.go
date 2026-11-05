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

func NewServer(repo *repository.Repo, st *storage.Storage, log *logrus.Logger) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(loggerMiddleware(log))
	r.Use(corsMiddleware()) // Добавим CORS для будущего SPA

	// Инициализируем контроллеры
	svcHandler := handler.NewServiceHandler(repo, st, log)
	usrHandler := handler.NewUserHandler(repo, log)

	// Группируем API
	api := r.Group("/api")
	{
		// === Домен: Услуги (Services) ===
		services := api.Group("/services")
		{
			services.GET("", svcHandler.GetServices)         // Список с фильтром
			services.GET("/feed", svcHandler.GetFeed)        // Лента
			services.GET("/draft", svcHandler.GetDraft)      // Черновик
			services.POST("", svcHandler.CreateService)      // Добавление (multipart)
			services.PUT("/:id/publish", svcHandler.Publish) // Публикация
			services.DELETE("/:id", svcHandler.Delete)       // Удаление (soft)
			services.POST("/:id/like", svcHandler.Like)      // Лайк (0/1)
		}

		// === Домен: Пользователи (Users) ===
		users := api.Group("/users")
		{
			users.POST("/register", usrHandler.Register) // Регистрация
			users.POST("/login", usrHandler.Login)       // Аутентификация (заглушка)
			users.POST("/logout", usrHandler.Logout)     // Деавторизация (заглушка)
		}
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
		}).Info("api")
	}
}

// Заготовка для CORS (чтобы фронтенд с другого порта мог стучаться)
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
