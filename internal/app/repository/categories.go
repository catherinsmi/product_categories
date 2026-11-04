package repository

// Categories — допустимые значения поля Category (поле по теме).
var Categories = []string{
	"Макияж",
	"Уход за лицом",
	"Уход за телом",
	"Парфюм",
	"Волосы",
	"Другое",
}

func IsValidCategory(c string) bool {
	for _, x := range Categories {
		if x == c {
			return true
		}
	}
	return false
}

// Статусы услуг (строковые константы, совпадают с CHECK в БД).
const (
	StatusDraft     = "draft"
	StatusPublished = "published"
	StatusDeleted   = "deleted"
)
