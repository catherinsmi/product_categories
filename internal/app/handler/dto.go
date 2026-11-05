package handler

import (
	"product_categories/internal/app/repository"
	"time"
)

// CurrentUserID — заглушка (Singleton), как требует задание.
// В лабе 4 мы будем брать ID из токена авторизации.
func CurrentUserID() uint {
	return 1 // Фиксированный ID (alice)
}

// ServiceListResponse — для GET /api/services
type ServiceListResponse struct {
	ID       uint    `json:"id"`
	Title    string  `json:"title"`
	Price    float64 `json:"price"`
	Category string  `json:"category"`
	ImageURL string  `json:"image_url"`
	IsOwner  bool    `json:"is_owner"` // 1 если создатель совпадает с текущим
}

// FeedResponse — для GET /api/services/feed
type FeedResponse struct {
	ID       uint    `json:"id"`
	Title    string  `json:"title"`
	Price    float64 `json:"price"`
	Category string  `json:"category"`
	VideoURL string  `json:"video_url"`
	ImageURL string  `json:"image_url"`
	IsLiked  bool    `json:"is_liked"` // 1 если текущий пользователь лайкнул
}

// DraftResponse — для GET /api/services/draft
type DraftResponse struct {
	ID          uint      `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Price       float64   `json:"price"`
	Category    string    `json:"category"`
	ImageURL    string    `json:"image_url"`
	VideoURL    string    `json:"video_url"`
	CreatedAt   time.Time `json:"created_at"`
}

// Вспомогательный метод для получения полного URL файла из MinIO
func (h *ServiceHandler) getMediaURL(key string) string {
	if key == "" {
		return ""
	}
	return h.st.GetFileURL(key)
}

// Преобразование модели в DTO для ленты
func (h *ServiceHandler) toFeedResponse(s repository.Service) FeedResponse {
	isLiked := false
	currentID := CurrentUserID()
	for _, like := range s.Likes {
		if like.UserID == currentID {
			isLiked = true
			break
		}
	}

	return FeedResponse{
		ID:       s.ID,
		Title:    s.Title,
		Price:    s.Price,
		Category: s.Category,
		VideoURL: h.getMediaURL(s.VideoURL),
		ImageURL: h.getMediaURL(s.ImageURL),
		IsLiked:  isLiked,
	}
}
