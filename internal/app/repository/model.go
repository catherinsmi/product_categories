package repository

import "fmt"

// Status — статус услуги.
type Status string

const (
	StatusDraft     Status = "draft"
	StatusPublished Status = "published"
	StatusDeleted   Status = "deleted"
)

// Order — единственная модель-коллекция. Все поля скаляры (1НФ),
// кроме Likes — []int (ID пользователей), это разрешено ТЗ.
type Order struct {
	ID          int
	Title       string
	Description string
	ImageKey    string
	VideoKey    string
	Price       float64
	FlightDate  string
	Likes       []int
	Status      Status
}

// ImageURL возвращает URL для /media/<key> или пустую строку.
func (o Order) ImageURL() string {
	if o.ImageKey == "" {
		return ""
	}
	return "/media/" + o.ImageKey
}

// VideoURL возвращает URL для /media/<key> или пустую строку.
func (o Order) VideoURL() string {
	if o.VideoKey == "" {
		return ""
	}
	return "/media/" + o.VideoKey
}

// LikesCount возвращает количество лайков (считает контроллер через модель, не шаблон).
func (o Order) LikesCount() int {
	return len(o.Likes)
}

// PriceStr — строковое представление цены без копеек.
func (o Order) PriceStr() string {
	return fmt.Sprintf("%.0f ₽", o.Price)
}
