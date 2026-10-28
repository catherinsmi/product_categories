package repository

import "time"

type User struct {
	ID        uint   `gorm:"primaryKey"`
	Username  string `gorm:"size:64;uniqueIndex;not null"`
	Email     string `gorm:"size:128;uniqueIndex;not null"`
	CreatedAt time.Time
	Likes     []Like `gorm:"foreignKey:UserID"`
}

type Service struct {
	ID          uint      `gorm:"primaryKey"`
	Title       string    `gorm:"size:128;not null"`
	Description string    `gorm:"type:text"`
	Status      string    `gorm:"size:16;not null;default:draft;check:status IN ('draft','published','deleted')"`
	ImageURL    string    `gorm:"size:512"`
	VideoURL    string    `gorm:"size:512"`
	Price       float64   `gorm:"type:numeric(10,2);not null;default:0"`
	Category    string    `gorm:"size:64;not null;default:''"`
	CreatorID   uint      `gorm:"not null"`
	Creator     User      `gorm:"foreignKey:CreatorID"`
	CreatedAt   time.Time `gorm:"not null"`
	PublishedAt *time.Time
	Likes       []Like `gorm:"foreignKey:ServiceID"`
}

type Like struct {
	ID        uint `gorm:"primaryKey"`
	UserID    uint `gorm:"not null"`
	ServiceID uint `gorm:"not null"`
	CreatedAt time.Time
	User      User    `gorm:"foreignKey:UserID"`
	Service   Service `gorm:"foreignKey:ServiceID"`
}
