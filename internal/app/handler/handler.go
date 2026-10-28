package handler

import (
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"product_categories/internal/app/repository"
	"product_categories/internal/storage"
)

// Handler — набор HTTP-обработчиков.
type Handler struct {
	repo        *repository.Repository
	st          *storage.Storage
	log         *logrus.Logger
	fallbackDir string // например, "resources/seed"
}

// New создаёт Handler. fallbackDir — куда смотреть, если MinIO недоступен.
func New(repo *repository.Repository, st *storage.Storage, log *logrus.Logger, fallbackDir string) *Handler {
	return &Handler{repo: repo, st: st, log: log, fallbackDir: fallbackDir}
}

// FeedIndex — GET /product_categories/feed
func (h *Handler) FeedIndex(c *gin.Context) {
	list := h.repo.Published()
	if len(list) == 0 {
		c.HTML(http.StatusOK, "product_categories_feed.html", gin.H{
			"Title":  "Лента",
			"Empty":  true,
			"Active": "feed",
		})
		return
	}
	c.HTML(http.StatusOK, "product_categories_feed.html", gin.H{
		"Title":  "Лента",
		"Order":  list[0],
		"Active": "feed",
	})
}

// FeedByID — GET /product_categories/feed/:id и .../feed/:id?next=true
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

	c.HTML(http.StatusOK, "product_categories_feed.html", gin.H{
		"Title":  "Лента",
		"Order":  order,
		"Active": "feed",
	})
}

// renderFeedNotFound — рисует пустую ленту с кодом 404.
func (h *Handler) renderFeedNotFound(c *gin.Context) {
	c.HTML(http.StatusNotFound, "product_categories_feed.html", gin.H{
		"Title":  "Лента",
		"Empty":  true,
		"Active": "feed",
	})
}

// Add — GET /product_categories/add
func (h *Handler) Add(c *gin.Context) {
	draft, ok := h.repo.Draft()

	c.HTML(http.StatusOK, "product_categories_add.html", gin.H{
		"Title":      "Добавить",
		"Order":      draft,
		"HasDraft":   ok,
		"Active":     "add",
		"Categories": repository.Categories,
	})
}

// Grid — GET /product_categories/grid?min=&max=
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

	c.HTML(http.StatusOK, "product_categories_grid.html", gin.H{
		"Title":  "Плитка",
		"Orders": orders,
		"Min":    minStr,
		"Max":    maxStr,
		"Active": "grid",
	})
}

// Media — GET /product_categories/media/*key
// Сначала пробует MinIO, при неудаче — локальную папку fallbackDir.
func (h *Handler) Media(c *gin.Context) {
	key := strings.TrimPrefix(c.Param("key"), "/")
	if key == "" {
		c.Status(http.StatusNotFound)
		return
	}

	// 1) MinIO
	if h.st.Enabled() {
		if err := h.st.Serve(c.Writer, c.Request, key); err == nil {
			return
		} else {
			h.log.WithError(err).WithField("key", key).Warn("minio media failed, falling back to local")
		}
	}

	// 2) локальный fallback
	if h.serveLocal(c, key) {
		return
	}

	h.log.WithField("key", key).Warn("media not found anywhere")
	c.Status(http.StatusNotFound)
}

// serveLocal ищет файл в fallbackDir и отдаёт его.
func (h *Handler) serveLocal(c *gin.Context, key string) bool {
	if h.fallbackDir == "" {
		return false
	}

	localName := key
	if i := strings.IndexByte(key, '/'); i >= 0 {
		prefix := key[:i]
		if prefix == filepath.Base(h.fallbackDir) {
			localName = key[i+1:]
		}
	}

	// защита от path traversal
	localName = filepath.Clean("/" + localName)
	localName = strings.TrimPrefix(localName, "/")
	full := filepath.Join(h.fallbackDir, localName)

	absBase, err := filepath.Abs(h.fallbackDir)
	if err != nil {
		return false
	}
	absFull, err := filepath.Abs(full)
	if err != nil {
		return false
	}
	if !strings.HasPrefix(absFull, absBase+string(os.PathSeparator)) && absFull != absBase {
		h.log.WithField("key", key).Warn("path traversal attempt blocked")
		return false
	}

	f, err := os.Open(absFull)
	if err != nil {
		return false
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil || info.IsDir() {
		return false
	}

	ct := mime.TypeByExtension(strings.ToLower(filepath.Ext(absFull)))
	if ct == "" {
		ct = "application/octet-stream"
	}
	c.Header("Content-Type", ct)
	c.Header("Content-Length", strconv.FormatInt(info.Size(), 10))
	c.Header("Cache-Control", "public, max-age=3600")

	http.ServeContent(c.Writer, c.Request, info.Name(), info.ModTime(), f)
	return true
}
