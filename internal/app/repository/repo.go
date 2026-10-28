package repository

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

type Repo struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

// ===== 3 GET (через ORM) =====

// FeedByID — одна строка published, ?next=true — следующая после id
func (r *Repo) FeedByID(id uint, next bool) (*Service, error) {
	var s Service
	if next {
		// следующий published после id
		err := r.db.Where("status = ? AND id > ?", "published", id).
			Order("id").First(&s).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// циклически — первая published
			err = r.db.Where("status = ?", "published").Order("id").First(&s).Error
		}
		if err != nil {
			return nil, err
		}
		return &s, nil
	}
	// обычный просмотр: только published
	err := r.db.Where("id = ? AND status = ?", id, "published").First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// FeedFirst — первая published (для /feed без id)
func (r *Repo) FeedFirst() (*Service, error) {
	var s Service
	err := r.db.Where("status = ?", "published").Order("id").First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// DraftByCreator — черновик текущего пользователя (для /add)
func (r *Repo) DraftByCreator(creatorID uint) (*Service, error) {
	var s Service
	err := r.db.Where("creator_id = ? AND status = ?", creatorID, "draft").First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// Grid — список published с фильтром по цене (через ORM)
func (r *Repo) Grid(min, max float64) ([]Service, error) {
	q := r.db.Where("status = ?", "published")
	if min > 0 {
		q = q.Where("price >= ?", min)
	}
	if max > 0 {
		q = q.Where("price <= ?", max)
	}
	var list []Service
	err := q.Order("id").Find(&list).Error
	return list, err
}

// ===== 2 POST через ORM =====

// CreateDraft — кнопка «Далее»: создать черновик
func (r *Repo) CreateDraft(s *Service) error {
	s.Status = "draft"
	s.CreatedAt = time.Now()
	return r.db.Create(s).Error
}

// Publish — кнопка «Опубликовать»: draft → published
func (r *Repo) Publish(serviceID, creatorID uint) error {
	now := time.Now()
	res := r.db.Model(&Service{}).
		Where("id = ? AND creator_id = ? AND status = ?", serviceID, creatorID, "draft").
		Updates(map[string]interface{}{
			"status":       "published",
			"published_at": &now,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("draft not found")
	}
	return nil
}

// ===== 1 POST — удаление через сырой SQL UPDATE (не ORM) =====

// DeleteServiceSQL — published → deleted, строго через db.Exec / QueryRow
// «SQL курсор»: используем QueryRow с RETURNING, чтобы получить результат
func (r *Repo) DeleteServiceSQL(id uint) error {
	var newStatus string
	err := r.db.Raw(
		`UPDATE services SET status = 'deleted' WHERE id = ? AND status = 'published' RETURNING status`,
		id,
	).Scan(&newStatus).Error

	if err != nil {
		return err
	}
	if newStatus == "" {
		return errors.New("service not found or not published")
	}
	return nil
}
