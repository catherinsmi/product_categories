package repository

import "fmt"

// Status — статус услуги.
type Status string

const (
	StatusDraft     Status = "draft"
	StatusPublished Status = "published"
	StatusDeleted   Status = "deleted"
)

// ProductCategories — единственная модель-коллекция. Все поля скаляры (1НФ),
// кроме Likes — []int (ID пользователей), это разрешено ТЗ.
type ProductCategories struct {
	ID          int
	Title       string
	Description string
	ImageKey    string
	VideoKey    string
	Price       float64
	Category    string
	Likes       []int
	Status      Status
}

// ImageURL возвращает URL для /product_categories/media/<key> или пустую строку.
func (o ProductCategories) ImageURL() string {
	if o.ImageKey == "" {
		return ""
	}
	return "/product_categories/media/" + o.ImageKey
}

// VideoURL возвращает URL для /product_categories/media/<key> или пустую строку.
func (o ProductCategories) VideoURL() string {
	if o.VideoKey == "" {
		return ""
	}
	return "/product_categories/media/" + o.VideoKey
}

// LikesCount возвращает количество лайков (считает контроллер через модель, не шаблон).
func (o ProductCategories) LikesCount() int {
	return len(o.Likes)
}

// DeadlineStr — строковое представление даты рейса.

// PriceStr — строковое представление цены без копеек.
func (o ProductCategories) PriceStr() string {
	return fmt.Sprintf("%.0f ₽", o.Price)
}

func (o ProductCategories) CategoryStr() string {
	if o.Category == "" {
		return "—"
	}
	return o.Category
}
