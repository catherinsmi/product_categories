package repository

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// ============================================================
// Инициализация
// ============================================================

type Repo struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

// DB — доступ к *gorm.DB (нужно для миграций/сидов в main.go).
func (r *Repo) DB() *gorm.DB { return r.db }

// ============================================================
// Чтение (GET)
// ============================================================

// FeedByID — одна строка published.
// next=true: следующая published после id; если нет — циклически первая.
func (r *Repo) FeedByID(id uint, next bool) (*Service, error) {
	var s Service

	if next {
		err := r.db.Preload("Likes").
			Where("status = ? AND id > ?", StatusPublished, id).
			Order("id ASC").
			First(&s).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = r.db.Preload("Likes").
				Where("status = ?", StatusPublished).
				Order("id ASC").
				First(&s).Error
		}
		if err != nil {
			return nil, err
		}
		return &s, nil
	}

	err := r.db.Preload("Likes").
		Where("id = ? AND status = ?", id, StatusPublished).
		First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// FeedFirst — первая published (для /feed без id).
func (r *Repo) FeedFirst() (*Service, error) {
	var s Service
	err := r.db.Preload("Likes").
		Where("status = ?", StatusPublished).
		Order("id ASC").
		First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// DraftByCreator — черновик пользователя (не более одного по индексу БД).
func (r *Repo) DraftByCreator(creatorID uint) (*Service, error) {
	var s Service
	err := r.db.Preload("Likes").
		Where("creator_id = ? AND status = ?", creatorID, StatusDraft).
		First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// Grid — published + фильтр по цене (через SQL, не в Go).
func (r *Repo) Grid(min, max float64) ([]Service, error) {
	q := r.db.Preload("Likes").Where("status = ?", StatusPublished)
	if min > 0 {
		q = q.Where("price >= ?", min)
	}
	if max > 0 {
		q = q.Where("price <= ?", max)
	}
	var list []Service
	err := q.Order("id ASC").Find(&list).Error
	return list, err
}

// ============================================================
// Изменение (POST/PUT/DELETE)
// ============================================================

// CreateDraft — «Далее»: создать черновик.
func (r *Repo) CreateDraft(s *Service) error {
	s.Status = StatusDraft
	s.CreatedAt = time.Now()
	return r.db.Create(s).Error
}

// Publish — «Опубликовать»: draft → published только для своего черновика.
func (r *Repo) Publish(serviceID, creatorID uint) error {
	now := time.Now()
	res := r.db.Model(&Service{}).
		Where("id = ? AND creator_id = ? AND status = ?", serviceID, creatorID, StatusDraft).
		Updates(map[string]interface{}{
			"status":       StatusPublished,
			"published_at": &now,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("draft not found or not owned")
	}
	return nil
}

// DeleteServiceSQL — published → deleted, строго через db.Raw (сырой SQL).
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

// ============================================================
// Лайки и Пользователи (Лабораторная №3)
// ============================================================

// AddLike — поставить лайк (если его еще нет)
func (r *Repo) AddLike(userID, serviceID uint) error {
	like := &Like{
		UserID:    userID,
		ServiceID: serviceID,
		CreatedAt: time.Now(),
	}
	// FirstOrCreate ищет по условиям. Если не находит — создает.
	return r.db.Where("user_id = ? AND service_id = ?", userID, serviceID).
		FirstOrCreate(like).Error
}

// RemoveLike — убрать лайк
func (r *Repo) RemoveLike(userID, serviceID uint) error {
	return r.db.Where("user_id = ? AND service_id = ?", userID, serviceID).
		Delete(&Like{}).Error
}

// CreateUser — регистрация нового пользователя
func (r *Repo) CreateUser(user *User) error {
	return r.db.Create(user).Error
}
