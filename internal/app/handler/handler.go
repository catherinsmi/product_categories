package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"product_categories/internal/app/repository"
	"product_categories/internal/storage"
)

type Handler struct {
	repo *repository.Repository
	st   *storage.Storage
	log  *logrus.Logger
}

func New(repo *repository.Repository, st *storage.Storage, log *logrus.Logger) *Handler {
	return &Handler{repo: repo, st: st, log: log}
}

// FeedIndex — GET /feed
func (h *Handler) FeedIndex(c *gin.Context) {
	list := h.repo.Published()
	if len(list) == 0 {
		c.HTML(http.StatusOK, "feed.html", gin.H{
			"Title":  "Лента",
			"Empty":  true,
			"Active": "feed",
		})
		return
	}
	c.HTML(http.StatusOK, "feed.html", gin.H{
		"Title":  "Лента",
		"Order":  list[0],
		"Active": "feed",
	})
}

// FeedByID — GET /feed/:id и GET /feed/:id?next=true
func (h *Handler) FeedByID(c *gin.Context) {
	raw := c.Param("id")
	id, err := strconv.Atoi(raw)
	if err != nil {
		h.log.WithField("id", raw).Warn("invalid feed id")
		h.renderFeedNotFound(c)
		return
	}

	order, ok := h.repo.ByID(id)
	if !ok || order.Status != repository.StatusPublished {
		h.log.WithField("id", id).Warn("order not found or not published")
		h.renderFeedNotFound(c)
		return
	}

	if c.Query("next") == "true" {
		if nx, ok := h.repo.NextPublished(id); ok {
			order = nx
		}
	}

	c.HTML(http.StatusOK, "feed.html", gin.H{
		"Title":  "Лента",
		"Order":  order,
		"Active": "feed",
	})
}

func (h *Handler) renderFeedNotFound(c *gin.Context) {
	c.HTML(http.StatusNotFound, "feed.html", gin.H{
		"Title":  "Лента",
		"Empty":  true,
		"Active": "feed",
	})
}

// Add — GET /add (показывает единственный черновик)
func (h *Handler) Add(c *gin.Context) {
	draft, ok := h.repo.Draft()
	c.HTML(http.StatusOK, "add.html", gin.H{
		"Title":    "Добавить",
		"Order":    draft,
		"HasDraft": ok,
		"Active":   "add",
	})
}

// Grid — GET /grid?min=&max=
func (h *Handler) Grid(c *gin.Context) {
	minStr := c.Query("min")
	maxStr := c.Query("max")

	var minV, maxV float64
	if v, err := strconv.ParseFloat(minStr, 64); err == nil {
		minV = v
	}
	if v, err := strconv.ParseFloat(maxStr, 64); err == nil {
		maxV = v
	}

	orders := h.repo.FilterByPrice(minV, maxV)

	c.HTML(http.StatusOK, "grid.html", gin.H{
		"Title":  "Плитка",
		"Orders": orders,
		"Min":    minStr,
		"Max":    maxStr,
		"Active": "grid",
	})
}

// Media — GET /media/*key — проксирует объект из MinIO.
func (h *Handler) Media(c *gin.Context) {
	key := strings.TrimPrefix(c.Param("key"), "/")
	if key == "" {
		c.Status(http.StatusNotFound)
		return
	}

	if err := h.st.Serve(c.Writer, c.Request, key); err != nil {
		h.log.WithError(err).WithField("key", key).Warn("media fetch failed")
		c.Status(http.StatusNotFound)
		return
	}
}
