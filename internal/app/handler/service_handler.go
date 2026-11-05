package handler

import (
	"net/http"
	"strconv"

	"product_categories/internal/app/repository"
	"product_categories/internal/storage"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type ServiceHandler struct {
	repo *repository.Repo
	st   *storage.Storage
	log  *logrus.Logger
}

func NewServiceHandler(repo *repository.Repo, st *storage.Storage, log *logrus.Logger) *ServiceHandler {
	return &ServiceHandler{repo: repo, st: st, log: log}
}

// GET /api/services — список с фильтрацией
func (h *ServiceHandler) GetServices(c *gin.Context) {
	min, _ := strconv.ParseFloat(c.Query("min"), 64)
	max, _ := strconv.ParseFloat(c.Query("max"), 64)

	services, err := h.repo.Grid(min, max)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}

	currentID := CurrentUserID()
	result := make([]ServiceListResponse, 0, len(services))
	for _, s := range services {
		result = append(result, ServiceListResponse{
			ID:       s.ID,
			Title:    s.Title,
			Price:    s.Price,
			Category: s.Category,
			ImageURL: h.getMediaURL(s.ImageURL),
			IsOwner:  s.CreatorID == currentID,
		})
	}

	c.JSON(http.StatusOK, result)
}

// GET /api/services/feed — лента
func (h *ServiceHandler) GetFeed(c *gin.Context) {
	idStr := c.Query("id")
	next := c.Query("next") == "true"

	var id uint
	if idStr != "" {
		parsed, _ := strconv.ParseUint(idStr, 10, 64)
		id = uint(parsed)
	}

	var svc *repository.Service
	var err error
	if id == 0 {
		svc, err = h.repo.FeedFirst()
	} else {
		svc, err = h.repo.FeedByID(id, next)
	}

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	c.JSON(http.StatusOK, h.toFeedResponse(*svc))
}

// GET /api/services/draft — получение черновика
func (h *ServiceHandler) GetDraft(c *gin.Context) {
	draft, err := h.repo.DraftByCreator(CurrentUserID())
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "draft not found"})
		return
	}

	c.JSON(http.StatusOK, DraftResponse{
		ID:          draft.ID,
		Title:       draft.Title,
		Description: draft.Description,
		Price:       draft.Price,
		Category:    draft.Category,
		ImageURL:    h.getMediaURL(draft.ImageURL),
		VideoURL:    h.getMediaURL(draft.VideoURL),
		CreatedAt:   draft.CreatedAt,
	})
}

// POST /api/services — добавление (multipart/form-data)
func (h *ServiceHandler) CreateService(c *gin.Context) {
	title := c.PostForm("title")
	description := c.PostForm("description")
	price, _ := strconv.ParseFloat(c.PostForm("price"), 64)
	category := c.PostForm("category")

	var imgKey, vidKey string
	if imageFile, err := c.FormFile("image"); err == nil {
		key, err := h.st.UploadFile(imageFile)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "image upload failed"})
			return
		}
		imgKey = key
	}

	if videoFile, err := c.FormFile("video"); err == nil {
		key, err := h.st.UploadFile(videoFile)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "video upload failed"})
			return
		}
		vidKey = key
	}

	svc := &repository.Service{
		Title:       title,
		Description: description,
		Price:       price,
		Category:    category,
		ImageURL:    imgKey,
		VideoURL:    vidKey,
		CreatorID:   CurrentUserID(),
	}

	if err := h.repo.CreateDraft(svc); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": svc.ID})
}

// PUT /api/services/:id/publish — публикация
func (h *ServiceHandler) Publish(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	err := h.repo.Publish(uint(id), CurrentUserID())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot publish"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "published"})
}

// DELETE /api/services/:id — удаление (soft)
func (h *ServiceHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	err := h.repo.DeleteServiceSQL(uint(id))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// POST /api/services/:id/like — лайк (0/1)
func (h *ServiceHandler) Like(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var req struct {
		Like int `json:"like"` // 0 или 1
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}

	if req.Like == 1 {
		h.repo.AddLike(CurrentUserID(), uint(id))
	} else {
		h.repo.RemoveLike(CurrentUserID(), uint(id))
	}

	c.JSON(http.StatusOK, gin.H{"message": "success"})
}
