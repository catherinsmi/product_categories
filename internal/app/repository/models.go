package repository

import (
	"fmt"
	"time"
)

// User — пользователь приложения (лайки м-м).
type User struct {
	ID        uint      `gorm:"primaryKey"`
	Username  string    `gorm:"size:64;uniqueIndex;not null"`
	Email     string    `gorm:"size:128;uniqueIndex;not null"`
	CreatedAt time.Time `gorm:"not null"`
	Likes     []Like    `gorm:"foreignKey:UserID"`
}

func (User) TableName() string { return "users" }

// Service — услуга (карточка). Поля по теме: Price, Category.
// Статусы: draft / published / deleted.
type Service struct {
	ID          uint      `gorm:"primaryKey"`
	Title       string    `gorm:"size:128;not null"`
	Description string    `gorm:"type:text"`
	Status      string    `gorm:"size:16;not null;default:draft;check:status IN ('draft','published','deleted')"`
	ImageURL    string    `gorm:"size:512"`
	VideoURL    string    `gorm:"size:512"`
	Price       float64   `gorm:"type:numeric(10,2);not null;default:0"`
	Category    string    `gorm:"size:64;not null;default:''"`
	CreatorID   uint      `gorm:"not null;index"`
	Creator     User      `gorm:"foreignKey:CreatorID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	CreatedAt   time.Time `gorm:"not null"`
	PublishedAt *time.Time
	Likes       []Like `gorm:"foreignKey:ServiceID"`
}

func (Service) TableName() string { return "services" }

// Like — м-м пользователь-услуга.
type Like struct {
	ID        uint      `gorm:"primaryKey"`
	UserID    uint      `gorm:"not null;uniqueIndex:ux_likes_user_service"`
	ServiceID uint      `gorm:"not null;uniqueIndex:ux_likes_user_service"`
	CreatedAt time.Time `gorm:"not null"`
	User      User      `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	Service   Service   `gorm:"foreignKey:ServiceID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
}

func (Like) TableName() string { return "likes" }

// ===== Хелперы для шаблонов =====

func (s Service) PriceStr() string {
	return fmt.Sprintf("%.0f ₽", s.Price)
}

func (s Service) CategoryStr() string {
	if s.Category == "" {
		return "—"
	}
	return s.Category
}

func (s Service) LikesCount() int {
	return len(s.Likes)
}

func (s Service) IsPublished() bool { return s.Status == StatusPublished }
func (s Service) IsDraft() bool     { return s.Status == StatusDraft }
func (s Service) IsDeleted() bool   { return s.Status == StatusDeleted }
