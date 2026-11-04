package handler

import (
	"context"
	"errors"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"product_categories/internal/app/repository"
)

// currentUserID — заглушка вместо аутентификации (будет в следующих лабах).
const currentUserID uint = 1

const (
	defaultImageURL = "/static/default/image.jpg"
	defaultVideoURL = "/static/default/video.mp4"
)

type Handler struct {
	repo      *repository.Repo
	log       *logrus.Logger
	staticDir string // resources/static
}

func New(repo *repository.Repo, log *logrus.Logger, staticDir string) *Handler {
	return &Handler{repo: repo, log: log, staticDir: staticDir}
}

// ============================================================
// GET 1/3 — лента: одна строка published (или ?next=true)
// URL: /product_categories/feed/:id
// ============================================================
func (h *Handler) FeedByID(c *gin.Context) {
	raw := c.Param("id")
	id64, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		h.renderFeedNotFound(c)
		return
	}
	id := uint(id64)
	next := c.Query("next") == "true"

	svc, err := h.repo.FeedByID(id, next)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			h.log.WithField("id", id).Warn("feed: service not found or not published")
		} else {
			h.log.WithError(err).Error("feed: db error")
		}
		h.renderFeedNotFound(c)
		return
	}

	imgURL, vidURL := h.resolveMedia(svc)
	c.HTML(http.StatusOK, "product_categories_feed.html", gin.H{
		"Title":  "Лента",
		"Order":  svc,
		"ImgURL": imgURL,
		"VidURL": vidURL,
		"Active": "feed",
		"Likes":  len(svc.Likes),
	})
}
func (h *Handler) FeedFirst(c *gin.Context) {
	svc, err := h.repo.FeedFirst()
	if err != nil {
		h.renderFeedNotFound(c)
		return
	}
	c.Redirect(http.StatusFound, "/product_categories/feed/"+strconv.FormatUint(uint64(svc.ID), 10))
}

// GET 2/3 — страница добавления/публикации черновика текущего пользователя
// URL: /product_categories/add
func (h *Handler) Add(c *gin.Context) {
	draft, err := h.repo.DraftByCreator(currentUserID)
	hasDraft := err == nil
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		h.log.WithError(err).Error("add: db error")
	}

	data := gin.H{
		"Title":      "Добавить",
		"Active":     "add",
		"HasDraft":   hasDraft,
		"Categories": repository.Categories,
	}
	if hasDraft {
		imgURL, vidURL := h.resolveMedia(draft)
		data["Order"] = draft
		data["ImgURL"] = imgURL
		data["VidURL"] = vidURL
	}
	c.HTML(http.StatusOK, "product_categories_add.html", data)
}

// GET 3/3 — плитка: список published с фильтром по цене
// URL: /product_categories/grid?min=&max=
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

	list, err := h.repo.Grid(minV, maxV)
	if err != nil {
		h.log.WithError(err).Error("grid: db error")
		list = nil
	}

	type gridItem struct {
		Service repository.Service
		ImgURL  string
		Likes   int
	}
	items := make([]gridItem, 0, len(list))
	for i := range list {
		imgURL, _ := h.resolveMedia(&list[i])
		items = append(items, gridItem{
			Service: list[i],
			ImgURL:  imgURL,
			Likes:   len(list[i].Likes),
		})
	}

	c.HTML(http.StatusOK, "product_categories_grid.html", gin.H{
		"Title":  "Плитка",
		"Items":  items,
		"Min":    minStr,
		"Max":    maxStr,
		"Active": "grid",
	})
}

// ============================================================
// POST 1/3 — «Далее»: создать черновик (через ORM)
// URL: /product_categories/add
// ============================================================
func (h *Handler) CreateDraft(c *gin.Context) {
	// повторная проверка: не более одного draft на пользователя
	if _, err := h.repo.DraftByCreator(currentUserID); err == nil {
		c.Redirect(http.StatusFound, "/product_categories/add")
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		h.log.WithError(err).Error("create draft: check error")
		c.Status(http.StatusInternalServerError)
		return
	}

	price, _ := strconv.ParseFloat(strings.ReplaceAll(c.PostForm("price"), " ", ""), 64)
	category := c.PostForm("category")
	if !repository.IsValidCategory(category) {
		category = "Другое"
	}

	svc := &repository.Service{
		Title:       strings.TrimSpace(c.PostForm("title")),
		Description: strings.TrimSpace(c.PostForm("description")),
		// Новые фото/видео не сохраняем реально — пишем заведомо неверные URL,
		// чтобы сработал fallback на default media (требование лабы 2).
		ImageURL:  "/broken/new-image.jpg",
		VideoURL:  "/broken/new-video.mp4",
		Price:     price,
		Category:  category,
		CreatorID: currentUserID,
	}
	if svc.Title == "" {
		svc.Title = "Новая карточка"
	}

	if err := h.repo.CreateDraft(svc); err != nil {
		h.log.WithError(err).Error("create draft: db error")
		c.Status(http.StatusInternalServerError)
		return
	}
	c.Redirect(http.StatusFound, "/product_categories/add")
}

// POST 2/3 — «Опубликовать»: draft → published (через ORM)
// URL: /product_categories/publish
func (h *Handler) Publish(c *gin.Context) {
	id64, err := strconv.ParseUint(c.PostForm("id"), 10, 64)
	if err != nil {
		c.Redirect(http.StatusFound, "/product_categories/add")
		return
	}
	if err := h.repo.Publish(uint(id64), currentUserID); err != nil {
		h.log.WithError(err).Warn("publish: failed")
		c.Redirect(http.StatusFound, "/product_categories/add")
		return
	}
	c.Redirect(http.StatusFound, "/product_categories/grid")
}

// POST 3/3 — логическое удаление: published → deleted (сырой SQL UPDATE, не ORM)
// URL: /product_categories/delete
func (h *Handler) Delete(c *gin.Context) {
	id64, err := strconv.ParseUint(c.PostForm("id"), 10, 64)
	if err != nil {
		c.Redirect(http.StatusFound, "/product_categories/grid")
		return
	}
	if err := h.repo.DeleteServiceSQL(uint(id64)); err != nil {
		h.log.WithError(err).Warn("delete: failed")
	}
	c.Redirect(http.StatusFound, "/product_categories/grid")
}

// ============================================================
// Вспомогательное
// ============================================================

func (h *Handler) renderFeedNotFound(c *gin.Context) {
	c.HTML(http.StatusNotFound, "product_categories_feed.html", gin.H{
		"Title":  "Лента",
		"Empty":  true,
		"Active": "feed",
	})
}

// resolveMedia — если URL пустой или файл недоступен локально — отдаём default.
func (h *Handler) resolveMedia(s *repository.Service) (imgURL, vidURL string) {
	imgURL = s.ImageURL
	if imgURL == "" || !h.localFileExists(imgURL) {
		imgURL = defaultImageURL
	}
	vidURL = s.VideoURL
	if vidURL == "" || !h.localFileExists(vidURL) {
		vidURL = defaultVideoURL
	}
	return
}

func (h *Handler) localFileExists(url string) bool {
	if !strings.HasPrefix(url, "/static/") {
		return false
	}
	rel := strings.TrimPrefix(url, "/static/")
	rel = filepath.Clean("/" + rel)
	rel = strings.TrimPrefix(rel, "/")
	full := filepath.Join(h.staticDir, rel)

	absBase, err := filepath.Abs(h.staticDir)
	if err != nil {
		return false
	}
	absFull, err := filepath.Abs(full)
	if err != nil {
		return false
	}
	if !strings.HasPrefix(absFull, absBase+string(os.PathSeparator)) && absFull != absBase {
		return false
	}

	info, err := os.Stat(absFull)
	return err == nil && !info.IsDir()
}

// Проверка доступности по HTTP (не используется сейчас, оставлено для будущего).
func (h *Handler) remoteFileExists(url string) bool {
	if url == "" {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 400
}

var _ = mime.TypeByExtension // заглушка, чтобы не ругался импорт, если уберёте remoteFileExists
